package managedblockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Network represents a Managed Blockchain network.
type Network struct {
	ID          string
	Name        string
	Description string
	Framework   string
	FrameworkVersion string
	Status      string
	CreationDate time.Time
	lifecycle   *lifecycle.Machine
}

// Member represents a network member.
type Member struct {
	ID          string
	NetworkID   string
	Name        string
	Description string
	Status      string
	CreationDate time.Time
}

// Node represents a peer node.
type Node struct {
	ID               string
	NetworkID        string
	MemberID         string
	InstanceType     string
	AvailabilityZone string
	Status           string
	CreationDate     time.Time
	lifecycle        *lifecycle.Machine
}

// Proposal represents a network proposal.
type Proposal struct {
	ProposalID  string
	NetworkID   string
	Description string
	ProposedByMemberID string
	Status      string
	CreationDate time.Time
	ExpirationDate time.Time
}

// Store manages Managed Blockchain resources in memory.
type Store struct {
	mu          sync.RWMutex
	networks    map[string]*Network
	members     map[string]map[string]*Member   // networkID -> memberID -> Member
	nodes       map[string]map[string]*Node     // networkID -> nodeID -> Node
	proposals   map[string]map[string]*Proposal // networkID -> proposalID -> Proposal
	accountID   string
	region      string
	lcConfig    *lifecycle.Config
	netSeq      int
	memberSeq   int
	nodeSeq     int
	proposalSeq int
}

// NewStore returns a new empty Managed Blockchain Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		networks:  make(map[string]*Network),
		members:   make(map[string]map[string]*Member),
		nodes:     make(map[string]map[string]*Node),
		proposals: make(map[string]map[string]*Proposal),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

// CreateNetwork creates a new network.
func (s *Store) CreateNetwork(name, description, framework, frameworkVersion string) (*Network, *Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.netSeq++
	netID := fmt.Sprintf("n-%012d", s.netSeq)

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "AVAILABLE", Delay: 3 * time.Second},
	}

	net := &Network{
		ID:               netID,
		Name:             name,
		Description:      description,
		Framework:        framework,
		FrameworkVersion: frameworkVersion,
		Status:           "CREATING",
		CreationDate:     time.Now().UTC(),
	}
	net.lifecycle = lifecycle.NewMachine("CREATING", transitions, s.lcConfig)
	net.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		net.Status = string(to)
	})

	s.networks[netID] = net
	s.members[netID] = make(map[string]*Member)
	s.nodes[netID] = make(map[string]*Node)
	s.proposals[netID] = make(map[string]*Proposal)

	// Create initial member
	s.memberSeq++
	memberID := fmt.Sprintf("m-%012d", s.memberSeq)
	member := &Member{
		ID:           memberID,
		NetworkID:    netID,
		Name:         name + "-member",
		Status:       "AVAILABLE",
		CreationDate: time.Now().UTC(),
	}
	s.members[netID][memberID] = member

	return net, member, nil
}

// GetNetwork retrieves a network.
func (s *Store) GetNetwork(id string) (*Network, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	net, ok := s.networks[id]
	return net, ok
}

// ListNetworks returns all networks.
func (s *Store) ListNetworks() []*Network {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Network, 0, len(s.networks))
	for _, n := range s.networks {
		out = append(out, n)
	}
	return out
}

// DeleteNetwork removes a network.
func (s *Store) DeleteNetwork(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	net, ok := s.networks[id]
	if !ok {
		return false
	}
	if net.lifecycle != nil {
		net.lifecycle.Stop()
	}
	// Stop all node lifecycles
	for _, node := range s.nodes[id] {
		if node.lifecycle != nil {
			node.lifecycle.Stop()
		}
	}
	delete(s.networks, id)
	delete(s.members, id)
	delete(s.nodes, id)
	delete(s.proposals, id)
	return true
}

// GetMember retrieves a member.
func (s *Store) GetMember(networkID, memberID string) (*Member, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mMap, ok := s.members[networkID]
	if !ok {
		return nil, false
	}
	m, ok := mMap[memberID]
	return m, ok
}

// ListMembers returns all members for a network.
func (s *Store) ListMembers(networkID string) []*Member {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mMap := s.members[networkID]
	out := make([]*Member, 0, len(mMap))
	for _, m := range mMap {
		out = append(out, m)
	}
	return out
}

// CreateNode creates a node in a network.
func (s *Store) CreateNode(networkID, memberID, instanceType, az string) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.networks[networkID]; !ok {
		return nil, fmt.Errorf("network not found: %s", networkID)
	}

	s.nodeSeq++
	nodeID := fmt.Sprintf("nd-%012d", s.nodeSeq)

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "AVAILABLE", Delay: 3 * time.Second},
	}

	node := &Node{
		ID:               nodeID,
		NetworkID:        networkID,
		MemberID:         memberID,
		InstanceType:     instanceType,
		AvailabilityZone: az,
		Status:           "CREATING",
		CreationDate:     time.Now().UTC(),
	}
	node.lifecycle = lifecycle.NewMachine("CREATING", transitions, s.lcConfig)
	node.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		node.Status = string(to)
	})

	s.nodes[networkID][nodeID] = node
	return node, nil
}

// GetNode retrieves a node.
func (s *Store) GetNode(networkID, nodeID string) (*Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nMap, ok := s.nodes[networkID]
	if !ok {
		return nil, false
	}
	n, ok := nMap[nodeID]
	return n, ok
}

// ListNodes returns all nodes for a network.
func (s *Store) ListNodes(networkID string) []*Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nMap := s.nodes[networkID]
	out := make([]*Node, 0, len(nMap))
	for _, n := range nMap {
		out = append(out, n)
	}
	return out
}

// DeleteNode removes a node.
func (s *Store) DeleteNode(networkID, nodeID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	nMap, ok := s.nodes[networkID]
	if !ok {
		return false
	}
	node, ok := nMap[nodeID]
	if !ok {
		return false
	}
	if node.lifecycle != nil {
		node.lifecycle.Stop()
	}
	delete(nMap, nodeID)
	return true
}

// CreateProposal creates a proposal.
func (s *Store) CreateProposal(networkID, description, memberID string) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.networks[networkID]; !ok {
		return nil, fmt.Errorf("network not found: %s", networkID)
	}

	s.proposalSeq++
	propID := fmt.Sprintf("p-%012d", s.proposalSeq)

	proposal := &Proposal{
		ProposalID:         propID,
		NetworkID:          networkID,
		Description:        description,
		ProposedByMemberID: memberID,
		Status:             "IN_PROGRESS",
		CreationDate:       time.Now().UTC(),
		ExpirationDate:     time.Now().UTC().Add(24 * time.Hour),
	}
	s.proposals[networkID][propID] = proposal
	return proposal, nil
}

// GetProposal retrieves a proposal.
func (s *Store) GetProposal(networkID, proposalID string) (*Proposal, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pMap, ok := s.proposals[networkID]
	if !ok {
		return nil, false
	}
	p, ok := pMap[proposalID]
	return p, ok
}

// ListProposals returns all proposals for a network.
func (s *Store) ListProposals(networkID string) []*Proposal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pMap := s.proposals[networkID]
	out := make([]*Proposal, 0, len(pMap))
	for _, p := range pMap {
		out = append(out, p)
	}
	return out
}
