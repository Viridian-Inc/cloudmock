package gateway

import (
	"encoding/binary"
	"log"
	"net"
	"strings"
)

// StartDNSServer starts a minimal UDP DNS server on the given port.
// Any A-record query for domain or *.domain is answered with 127.0.0.1.
// All other queries receive an NXDOMAIN response (no forwarding).
//
// This is intentionally dependency-free — it parses and constructs DNS
// packets manually using only the standard library.
//
// Typical usage (non-privileged port):
//
//	go gateway.StartDNSServer(15353, "local.autotend.io")
func StartDNSServer(port int, domain string) {
	addr := &net.UDPAddr{Port: port, IP: net.ParseIP("127.0.0.1")}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("dns: failed to listen on UDP :%d: %v", port, err)
		return
	}
	log.Printf("dns: listening on UDP :%d (resolving *.%s → 127.0.0.1)", port, domain)

	buf := make([]byte, 512)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("dns: read error: %v", err)
			continue
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		go handleDNSQuery(conn, src, pkt, domain)
	}
}

// handleDNSQuery processes a single DNS query packet.
func handleDNSQuery(conn *net.UDPConn, src *net.UDPAddr, pkt []byte, domain string) {
	if len(pkt) < 12 {
		return // too short to be a valid DNS packet
	}

	// DNS header layout (12 bytes):
	//   0-1  : ID
	//   2-3  : Flags
	//   4-5  : QDCOUNT
	//   6-7  : ANCOUNT
	//   8-9  : NSCOUNT
	//   10-11: ARCOUNT
	id := pkt[0:2]
	qdCount := int(binary.BigEndian.Uint16(pkt[4:6]))

	if qdCount == 0 {
		return
	}

	// Parse the first question section.
	qName, qType, offset, ok := parseQuestion(pkt, 12)
	if !ok {
		return
	}
	_ = offset // additional questions ignored

	// Determine whether this query matches our domain.
	matches := dnsNameMatches(qName, domain)

	// Build response.
	resp := buildResponse(id, pkt[12:12+questionWireLen(pkt, 12)], qName, qType, matches)
	if resp == nil {
		return
	}

	if _, err := conn.WriteToUDP(resp, src); err != nil {
		log.Printf("dns: write error: %v", err)
	}
}

// dnsNameMatches returns true if name equals domain or is a subdomain of it.
// Comparison is case-insensitive.
func dnsNameMatches(name, domain string) bool {
	name = strings.ToLower(strings.TrimSuffix(name, "."))
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	return name == domain || strings.HasSuffix(name, "."+domain)
}

// parseQuestion parses a DNS question starting at offset in pkt.
// Returns the decoded name, qtype, the offset after the question, and ok.
func parseQuestion(pkt []byte, offset int) (name string, qtype uint16, end int, ok bool) {
	labels := []string{}
	for {
		if offset >= len(pkt) {
			return "", 0, 0, false
		}
		length := int(pkt[offset])
		offset++
		if length == 0 {
			break
		}
		// Pointer compression (0xC0 prefix) — rare in queries, skip gracefully.
		if length&0xC0 == 0xC0 {
			offset++ // skip pointer
			break
		}
		if offset+length > len(pkt) {
			return "", 0, 0, false
		}
		labels = append(labels, string(pkt[offset:offset+length]))
		offset += length
	}
	if offset+4 > len(pkt) {
		return "", 0, 0, false
	}
	qtype = binary.BigEndian.Uint16(pkt[offset:])
	// qclass = binary.BigEndian.Uint16(pkt[offset+2:]) // always IN=1
	offset += 4
	return strings.Join(labels, "."), qtype, offset, true
}

// questionWireLen returns the number of bytes the question section occupies
// starting at offset in pkt (used to copy the raw question into the response).
func questionWireLen(pkt []byte, offset int) int {
	start := offset
	for {
		if offset >= len(pkt) {
			return 0
		}
		length := int(pkt[offset])
		offset++
		if length == 0 {
			break
		}
		if length&0xC0 == 0xC0 {
			offset++
			break
		}
		offset += length
	}
	offset += 4 // QTYPE + QCLASS
	return offset - start
}

const (
	dnsTypeA   = 1
	dnsClassIN = 1
)

// buildResponse constructs a DNS response packet.
// If matches is true and qtype is A (or ANY), an A record for 127.0.0.1 is included.
// Otherwise an NXDOMAIN response is returned.
// rawQuestion is the raw bytes of the question section to echo back.
func buildResponse(id []byte, rawQuestion []byte, qName string, qtype uint16, matches bool) []byte {
	const loopback = "\x7f\x00\x00\x01" // 127.0.0.1

	// Flags:
	//   QR=1 (response), Opcode=0, AA=1, TC=0, RD=0, RA=0, Z=0, RCODE=0 (NOERROR)
	//   For NXDOMAIN: RCODE=3
	var flags uint16
	var anCount uint16

	wantA := qtype == dnsTypeA || qtype == 255 // 255 = ANY

	if matches && wantA {
		flags = 0x8400 // QR=1, AA=1, RCODE=0
		anCount = 1
	} else if matches {
		// Matched domain but wrong record type — NOERROR with no answers.
		flags = 0x8400
		anCount = 0
	} else {
		// Not our domain — NXDOMAIN.
		flags = 0x8403 // QR=1, AA=1, RCODE=3
		anCount = 0
	}

	// Build the DNS name in wire format for the answer section.
	var wireQName []byte
	for _, label := range strings.Split(strings.TrimSuffix(qName, "."), ".") {
		if label == "" {
			continue
		}
		wireQName = append(wireQName, byte(len(label)))
		wireQName = append(wireQName, []byte(label)...)
	}
	wireQName = append(wireQName, 0x00)

	// Header (12 bytes)
	header := make([]byte, 12)
	copy(header[0:2], id)
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], 1) // QDCOUNT=1
	binary.BigEndian.PutUint16(header[6:8], anCount)
	// NSCOUNT and ARCOUNT remain 0

	var resp []byte
	resp = append(resp, header...)
	resp = append(resp, rawQuestion...)

	if anCount == 1 {
		// Answer RR:
		//   NAME  (wire format)
		//   TYPE  2 bytes = A (1)
		//   CLASS 2 bytes = IN (1)
		//   TTL   4 bytes = 60
		//   RDLEN 2 bytes = 4
		//   RDATA 4 bytes = 127.0.0.1
		rr := make([]byte, 0, len(wireQName)+14)
		rr = append(rr, wireQName...)
		rr = append(rr, 0x00, byte(dnsTypeA))   // TYPE A
		rr = append(rr, 0x00, byte(dnsClassIN)) // CLASS IN
		rr = append(rr, 0x00, 0x00, 0x00, 0x3C) // TTL 60
		rr = append(rr, 0x00, 0x04)             // RDLENGTH 4
		rr = append(rr, loopback...)
		resp = append(resp, rr...)
	}

	return resp
}
