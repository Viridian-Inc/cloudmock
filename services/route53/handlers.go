package route53

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Route 53 XML namespace.
const r53NS = "https://route53.amazonaws.com/doc/2013-04-01/"

// newUUID generates a random request ID.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// xmlOK returns a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlCreated returns a 201 XML response.
func xmlCreated(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr wraps an AWSError for XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

// ---- XML types ----

type xmlChangeInfo struct {
	Id          string `xml:"Id"`
	Status      string `xml:"Status"`
	SubmittedAt string `xml:"SubmittedAt"`
}

type xmlNameServer struct {
	Value string `xml:",chardata"`
}

type xmlDelegationSet struct {
	NameServers []string `xml:"NameServers>NameServer"`
}

type xmlHostedZone struct {
	Id              string        `xml:"Id"`
	Name            string        `xml:"Name"`
	CallerReference string        `xml:"CallerReference"`
	Config          xmlZoneConfig `xml:"Config"`
}

type xmlZoneConfig struct {
	Comment     string `xml:"Comment,omitempty"`
	PrivateZone bool   `xml:"PrivateZone"`
}

// ---- CreateHostedZone ----

type xmlCreateHostedZoneRequest struct {
	XMLName         xml.Name `xml:"CreateHostedZoneRequest"`
	Name            string   `xml:"Name"`
	CallerReference string   `xml:"CallerReference"`
	HostedZoneConfig struct {
		Comment     string `xml:"Comment"`
		PrivateZone bool   `xml:"PrivateZone"`
	} `xml:"HostedZoneConfig"`
}

type xmlCreateHostedZoneResponse struct {
	XMLName      xml.Name         `xml:"CreateHostedZoneResponse"`
	NS           string           `xml:"xmlns,attr"`
	HostedZone   xmlHostedZone    `xml:"HostedZone"`
	ChangeInfo   xmlChangeInfo    `xml:"ChangeInfo"`
	DelegationSet xmlDelegationSet `xml:"DelegationSet"`
}

func handleCreateHostedZone(ctx *service.RequestContext, store *ZoneStore) (*service.Response, error) {
	var req xmlCreateHostedZoneRequest
	if err := xml.Unmarshal(ctx.Body, &req); err != nil {
		return xmlErr(service.ErrValidation("invalid XML body: " + err.Error()))
	}

	if req.Name == "" {
		return xmlErr(service.ErrValidation("Name is required."))
	}
	if req.CallerReference == "" {
		return xmlErr(service.ErrValidation("CallerReference is required."))
	}

	config := ZoneConfig{
		Comment:     req.HostedZoneConfig.Comment,
		PrivateZone: req.HostedZoneConfig.PrivateZone,
	}

	zone, err := store.CreateZone(req.Name, req.CallerReference, config)
	if err != nil {
		return xmlErr(service.ErrValidation(err.Error()))
	}

	resp := &xmlCreateHostedZoneResponse{
		NS: r53NS,
		HostedZone: xmlHostedZone{
			Id:              zone.Id,
			Name:            zone.Name,
			CallerReference: zone.CallerReference,
			Config: xmlZoneConfig{
				Comment:     zone.Config.Comment,
				PrivateZone: zone.Config.PrivateZone,
			},
		},
		ChangeInfo: xmlChangeInfo{
			Id:          "/change/" + newUUID(),
			Status:      "INSYNC",
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		},
		DelegationSet: delegationSetFromZone(zone),
	}
	return xmlCreated(resp)
}

// ---- ListHostedZones ----

type xmlListHostedZonesResponse struct {
	XMLName     xml.Name        `xml:"ListHostedZonesResponse"`
	NS          string          `xml:"xmlns,attr"`
	HostedZones []xmlHostedZone `xml:"HostedZones>HostedZone"`
	IsTruncated bool            `xml:"IsTruncated"`
	MaxItems    string          `xml:"MaxItems"`
}

func handleListHostedZones(ctx *service.RequestContext, store *ZoneStore) (*service.Response, error) {
	zones := store.ListZones()

	xmlZones := make([]xmlHostedZone, 0, len(zones))
	for _, z := range zones {
		xmlZones = append(xmlZones, xmlHostedZone{
			Id:              z.Id,
			Name:            z.Name,
			CallerReference: z.CallerReference,
			Config: xmlZoneConfig{
				Comment:     z.Config.Comment,
				PrivateZone: z.Config.PrivateZone,
			},
		})
	}

	resp := &xmlListHostedZonesResponse{
		NS:          r53NS,
		HostedZones: xmlZones,
		IsTruncated: false,
		MaxItems:    "100",
	}
	return xmlOK(resp)
}

// ---- GetHostedZone ----

type xmlGetHostedZoneResponse struct {
	XMLName       xml.Name         `xml:"GetHostedZoneResponse"`
	NS            string           `xml:"xmlns,attr"`
	HostedZone    xmlHostedZone    `xml:"HostedZone"`
	DelegationSet xmlDelegationSet `xml:"DelegationSet"`
}

func handleGetHostedZone(ctx *service.RequestContext, store *ZoneStore, zoneID string) (*service.Response, error) {
	zone, ok := store.GetZone(zoneID)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchHostedZone",
			"No hosted zone found with ID: "+zoneID, http.StatusNotFound))
	}

	resp := &xmlGetHostedZoneResponse{
		NS: r53NS,
		HostedZone: xmlHostedZone{
			Id:              zone.Id,
			Name:            zone.Name,
			CallerReference: zone.CallerReference,
			Config: xmlZoneConfig{
				Comment:     zone.Config.Comment,
				PrivateZone: zone.Config.PrivateZone,
			},
		},
		DelegationSet: delegationSetFromZone(zone),
	}
	return xmlOK(resp)
}

// ---- DeleteHostedZone ----

type xmlDeleteHostedZoneResponse struct {
	XMLName    xml.Name      `xml:"DeleteHostedZoneResponse"`
	NS         string        `xml:"xmlns,attr"`
	ChangeInfo xmlChangeInfo `xml:"ChangeInfo"`
}

func handleDeleteHostedZone(ctx *service.RequestContext, store *ZoneStore, zoneID string) (*service.Response, error) {
	if !store.DeleteZone(zoneID) {
		return xmlErr(service.NewAWSError("NoSuchHostedZone",
			"No hosted zone found with ID: "+zoneID, http.StatusNotFound))
	}

	resp := &xmlDeleteHostedZoneResponse{
		NS: r53NS,
		ChangeInfo: xmlChangeInfo{
			Id:          "/change/" + newUUID(),
			Status:      "INSYNC",
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	return xmlOK(resp)
}

// ---- ChangeResourceRecordSets ----

type xmlChangeResourceRecordSetsRequest struct {
	XMLName     xml.Name        `xml:"ChangeResourceRecordSetsRequest"`
	ChangeBatch xmlChangeBatch  `xml:"ChangeBatch"`
}

type xmlChangeBatch struct {
	Comment string      `xml:"Comment"`
	Changes []xmlChange `xml:"Changes>Change"`
}

type xmlChange struct {
	Action            string              `xml:"Action"`
	ResourceRecordSet xmlResourceRecordSet `xml:"ResourceRecordSet"`
}

type xmlResourceRecordSet struct {
	Name            string              `xml:"Name"`
	Type            string              `xml:"Type"`
	TTL             int64               `xml:"TTL"`
	ResourceRecords []xmlResourceRecord `xml:"ResourceRecords>ResourceRecord"`
}

type xmlResourceRecord struct {
	Value string `xml:"Value"`
}

type xmlChangeResourceRecordSetsResponse struct {
	XMLName    xml.Name      `xml:"ChangeResourceRecordSetsResponse"`
	NS         string        `xml:"xmlns,attr"`
	ChangeInfo xmlChangeInfo `xml:"ChangeInfo"`
}

func handleChangeResourceRecordSets(ctx *service.RequestContext, store *ZoneStore, zoneID string) (*service.Response, error) {
	var req xmlChangeResourceRecordSetsRequest
	if err := xml.Unmarshal(ctx.Body, &req); err != nil {
		return xmlErr(service.ErrValidation("invalid XML body: " + err.Error()))
	}

	changes := make([]Change, 0, len(req.ChangeBatch.Changes))
	for _, ch := range req.ChangeBatch.Changes {
		rrs := ResourceRecordSet{
			Name: ch.ResourceRecordSet.Name,
			Type: ch.ResourceRecordSet.Type,
			TTL:  ch.ResourceRecordSet.TTL,
		}
		for _, rr := range ch.ResourceRecordSet.ResourceRecords {
			rrs.ResourceRecords = append(rrs.ResourceRecords, ResourceRecord{Value: rr.Value})
		}
		changes = append(changes, Change{
			Action: ch.Action,
			RRSet:  rrs,
		})
	}

	if err := store.ChangeRecords(zoneID, changes); err != nil {
		return xmlErr(service.NewAWSError("InvalidChangeBatch", err.Error(), http.StatusBadRequest))
	}

	resp := &xmlChangeResourceRecordSetsResponse{
		NS: r53NS,
		ChangeInfo: xmlChangeInfo{
			Id:          "/change/" + newUUID(),
			Status:      "INSYNC",
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	return xmlOK(resp)
}

// ---- ListResourceRecordSets ----

type xmlListResourceRecordSetsResponse struct {
	XMLName            xml.Name              `xml:"ListResourceRecordSetsResponse"`
	NS                 string                `xml:"xmlns,attr"`
	ResourceRecordSets []xmlResourceRecordSet `xml:"ResourceRecordSets>ResourceRecordSet"`
	IsTruncated        bool                  `xml:"IsTruncated"`
	MaxItems           string                `xml:"MaxItems"`
}

func handleListResourceRecordSets(ctx *service.RequestContext, store *ZoneStore, zoneID string) (*service.Response, error) {
	records, ok := store.ListRecords(zoneID)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchHostedZone",
			"No hosted zone found with ID: "+zoneID, http.StatusNotFound))
	}

	xmlRRSets := make([]xmlResourceRecordSet, 0, len(records))
	for _, rs := range records {
		xmlRRs := make([]xmlResourceRecord, 0, len(rs.ResourceRecords))
		for _, rr := range rs.ResourceRecords {
			xmlRRs = append(xmlRRs, xmlResourceRecord{Value: rr.Value})
		}
		xmlRRSets = append(xmlRRSets, xmlResourceRecordSet{
			Name:            rs.Name,
			Type:            rs.Type,
			TTL:             rs.TTL,
			ResourceRecords: xmlRRs,
		})
	}

	resp := &xmlListResourceRecordSetsResponse{
		NS:                 r53NS,
		ResourceRecordSets: xmlRRSets,
		IsTruncated:        false,
		MaxItems:           "300",
	}
	return xmlOK(resp)
}

// ---- helpers ----

// delegationSetFromZone extracts the NS records from a zone and returns a DelegationSet.
func delegationSetFromZone(zone *HostedZone) xmlDelegationSet {
	for _, rs := range zone.RecordSets {
		if rs.Type == "NS" {
			ns := make([]string, 0, len(rs.ResourceRecords))
			for _, rr := range rs.ResourceRecords {
				ns = append(ns, rr.Value)
			}
			return xmlDelegationSet{NameServers: ns}
		}
	}
	return xmlDelegationSet{}
}

// zoneIDFromPath extracts the short zone ID from a path like /2013-04-01/hostedzone/ZABCDEF123456.
// It returns the segment after "hostedzone/".
func zoneIDFromPath(path string) string {
	// Path patterns:
	//   /2013-04-01/hostedzone/Z123
	//   /2013-04-01/hostedzone/Z123/rrset
	const marker = "/hostedzone/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(marker):]
	// Strip anything after the next slash (e.g. /rrset).
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	return rest
}
