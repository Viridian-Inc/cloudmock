package identitystore

import (
	"fmt"
	"sync"
	"time"
)

// User represents an Identity Store user.
type User struct {
	UserID        string
	IdentityStoreID string
	UserName      string
	DisplayName   string
	Name          *Name
	Emails        []Email
	Addresses     []Address
	PhoneNumbers  []PhoneNumber
	ExternalIds   []ExternalID
}

// Name represents a user's name.
type Name struct {
	GivenName  string
	FamilyName string
	MiddleName string
	Formatted  string
}

// Email represents a user email.
type Email struct {
	Value   string
	Type    string
	Primary bool
}

// Address represents a user address.
type Address struct {
	StreetAddress string
	Locality      string
	Region        string
	PostalCode    string
	Country       string
	Type          string
	Primary       bool
}

// PhoneNumber represents a user phone number.
type PhoneNumber struct {
	Value   string
	Type    string
	Primary bool
}

// ExternalID represents an external ID.
type ExternalID struct {
	Issuer string
	ID     string
}

// Group represents an Identity Store group.
type Group struct {
	GroupID         string
	IdentityStoreID string
	DisplayName     string
	Description     string
	ExternalIds     []ExternalID
}

// GroupMembership represents a group membership.
type GroupMembership struct {
	MembershipID    string
	IdentityStoreID string
	GroupID         string
	MemberID        string
}

// Store manages Identity Store resources in memory.
type Store struct {
	mu          sync.RWMutex
	users       map[string]*User                   // userID -> user
	groups      map[string]*Group                  // groupID -> group
	memberships map[string]*GroupMembership         // membershipID -> membership
	accountID   string
	region      string
	userSeq     int
	groupSeq    int
	memberSeq   int
}

// NewStore returns a new empty Identity Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		users:       make(map[string]*User),
		groups:      make(map[string]*Group),
		memberships: make(map[string]*GroupMembership),
		accountID:   accountID,
		region:      region,
	}
}

// CreateUser creates a new user.
func (s *Store) CreateUser(identityStoreID, userName, displayName string, name *Name, emails []Email) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate username
	for _, u := range s.users {
		if u.IdentityStoreID == identityStoreID && u.UserName == userName {
			return nil, fmt.Errorf("user with username %s already exists", userName)
		}
	}

	s.userSeq++
	userID := fmt.Sprintf("%012d-%04d-%04d-%04d-%012d",
		time.Now().UnixNano()%1000000000000, s.userSeq, 0, 0, s.userSeq)

	user := &User{
		UserID:          userID,
		IdentityStoreID: identityStoreID,
		UserName:        userName,
		DisplayName:     displayName,
		Name:            name,
		Emails:          emails,
	}
	s.users[userID] = user
	return user, nil
}

// GetUser retrieves a user by ID.
func (s *Store) GetUser(identityStoreID, userID string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[userID]
	if !ok || u.IdentityStoreID != identityStoreID {
		return nil, false
	}
	return u, true
}

// ListUsers returns all users for an identity store.
func (s *Store) ListUsers(identityStoreID string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0)
	for _, u := range s.users {
		if u.IdentityStoreID == identityStoreID {
			out = append(out, u)
		}
	}
	return out
}

// DeleteUser removes a user.
func (s *Store) DeleteUser(identityStoreID, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok || u.IdentityStoreID != identityStoreID {
		return false
	}
	delete(s.users, userID)
	// Remove memberships
	for id, m := range s.memberships {
		if m.MemberID == userID {
			delete(s.memberships, id)
		}
	}
	return true
}

// CreateGroup creates a new group.
func (s *Store) CreateGroup(identityStoreID, displayName, description string) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.groupSeq++
	groupID := fmt.Sprintf("%012d-%04d-%04d-%04d-%012d",
		time.Now().UnixNano()%1000000000000, s.groupSeq, 0, 0, s.groupSeq)

	group := &Group{
		GroupID:         groupID,
		IdentityStoreID: identityStoreID,
		DisplayName:    displayName,
		Description:    description,
	}
	s.groups[groupID] = group
	return group, nil
}

// GetGroup retrieves a group by ID.
func (s *Store) GetGroup(identityStoreID, groupID string) (*Group, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[groupID]
	if !ok || g.IdentityStoreID != identityStoreID {
		return nil, false
	}
	return g, true
}

// ListGroups returns all groups for an identity store.
func (s *Store) ListGroups(identityStoreID string) []*Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Group, 0)
	for _, g := range s.groups {
		if g.IdentityStoreID == identityStoreID {
			out = append(out, g)
		}
	}
	return out
}

// DeleteGroup removes a group.
func (s *Store) DeleteGroup(identityStoreID, groupID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[groupID]
	if !ok || g.IdentityStoreID != identityStoreID {
		return false
	}
	delete(s.groups, groupID)
	// Remove memberships
	for id, m := range s.memberships {
		if m.GroupID == groupID {
			delete(s.memberships, id)
		}
	}
	return true
}

// CreateGroupMembership creates a group membership.
func (s *Store) CreateGroupMembership(identityStoreID, groupID, memberID string) (*GroupMembership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	for _, m := range s.memberships {
		if m.IdentityStoreID == identityStoreID && m.GroupID == groupID && m.MemberID == memberID {
			return nil, fmt.Errorf("membership already exists")
		}
	}

	s.memberSeq++
	membershipID := fmt.Sprintf("%012d-%04d-%04d-%04d-%012d",
		time.Now().UnixNano()%1000000000000, s.memberSeq, 0, 0, s.memberSeq)

	membership := &GroupMembership{
		MembershipID:    membershipID,
		IdentityStoreID: identityStoreID,
		GroupID:         groupID,
		MemberID:        memberID,
	}
	s.memberships[membershipID] = membership
	return membership, nil
}

// GetGroupMembership retrieves a group membership.
func (s *Store) GetGroupMembership(identityStoreID, membershipID string) (*GroupMembership, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.memberships[membershipID]
	if !ok || m.IdentityStoreID != identityStoreID {
		return nil, false
	}
	return m, true
}

// ListGroupMemberships returns all memberships for a group.
func (s *Store) ListGroupMemberships(identityStoreID, groupID string) []*GroupMembership {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*GroupMembership, 0)
	for _, m := range s.memberships {
		if m.IdentityStoreID == identityStoreID && m.GroupID == groupID {
			out = append(out, m)
		}
	}
	return out
}

// DeleteGroupMembership removes a group membership.
func (s *Store) DeleteGroupMembership(identityStoreID, membershipID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.memberships[membershipID]
	if !ok || m.IdentityStoreID != identityStoreID {
		return false
	}
	delete(s.memberships, membershipID)
	return true
}
