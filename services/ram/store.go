package ram

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// ResourceShare holds a RAM resource share.
type ResourceShare struct {
	ResourceShareArn     string
	Name                 string
	OwningAccountId      string
	Status               string // ACTIVE, PENDING, FAILED, DELETING, DELETED
	AllowExternalPrincipals bool
	CreationTime         time.Time
	LastUpdatedTime      time.Time
	StatusMessage        string
	FeatureSet           string
	Tags                 []Tag
}

// ResourceShareAssociation tracks principals and resources in a share.
type ResourceShareAssociation struct {
	ResourceShareArn     string
	AssociatedEntity     string // Principal ARN or resource ARN
	AssociationType      string // PRINCIPAL or RESOURCE
	Status               string
	CreationTime         time.Time
	LastUpdatedTime      time.Time
	External             bool
	StatusMessage        string
}

// ResourceShareInvitation holds a pending invitation.
type ResourceShareInvitation struct {
	ResourceShareInvitationArn string
	ResourceShareArn           string
	ResourceShareName          string
	SenderAccountId            string
	ReceiverAccountId          string
	Status                     string // PENDING, ACCEPTED, REJECTED, EXPIRED
	InvitationTimestamp        time.Time
}

// Store is the in-memory store for RAM resources.
type Store struct {
	mu           sync.RWMutex
	shares       map[string]*ResourceShare
	associations []ResourceShareAssociation
	invitations  map[string]*ResourceShareInvitation
	accountID    string
	region       string
}

// NewStore creates an empty RAM Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		shares:       make(map[string]*ResourceShare),
		associations: make([]ResourceShareAssociation, 0),
		invitations:  make(map[string]*ResourceShareInvitation),
		accountID:    accountID,
		region:       region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) buildShareArn(id string) string {
	return fmt.Sprintf("arn:aws:ram:%s:%s:resource-share/%s", s.region, s.accountID, id)
}

func (s *Store) buildInvitationArn(id string) string {
	return fmt.Sprintf("arn:aws:ram:%s:%s:resource-share-invitation/%s", s.region, s.accountID, id)
}

// CreateResourceShare creates a new resource share.
func (s *Store) CreateResourceShare(name string, allowExternal bool, principals, resourceArns []string, tags []Tag) (*ResourceShare, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Name is required.")
	}

	id := newUUID()
	now := time.Now().UTC()
	share := &ResourceShare{
		ResourceShareArn:     s.buildShareArn(id),
		Name:                 name,
		OwningAccountId:      s.accountID,
		Status:               "ACTIVE",
		AllowExternalPrincipals: allowExternal,
		CreationTime:         now,
		LastUpdatedTime:      now,
		FeatureSet:           "STANDARD",
		Tags:                 tags,
	}
	s.shares[share.ResourceShareArn] = share

	// Create associations for principals
	for _, principal := range principals {
		s.associations = append(s.associations, ResourceShareAssociation{
			ResourceShareArn: share.ResourceShareArn,
			AssociatedEntity: principal,
			AssociationType:  "PRINCIPAL",
			Status:           "ASSOCIATED",
			CreationTime:     now,
			LastUpdatedTime:  now,
		})
	}

	// Create associations for resources
	for _, resourceArn := range resourceArns {
		s.associations = append(s.associations, ResourceShareAssociation{
			ResourceShareArn: share.ResourceShareArn,
			AssociatedEntity: resourceArn,
			AssociationType:  "RESOURCE",
			Status:           "ASSOCIATED",
			CreationTime:     now,
			LastUpdatedTime:  now,
		})
	}

	return share, nil
}

// GetResourceShares returns resource shares matching criteria.
func (s *Store) GetResourceShares(resourceOwner string) []*ResourceShare {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ResourceShare, 0, len(s.shares))
	for _, share := range s.shares {
		if share.Status == "DELETED" {
			continue
		}
		if resourceOwner == "SELF" && share.OwningAccountId == s.accountID {
			out = append(out, share)
		} else if resourceOwner == "OTHER-ACCOUNTS" && share.OwningAccountId != s.accountID {
			out = append(out, share)
		} else if resourceOwner == "" {
			out = append(out, share)
		}
	}
	return out
}

// UpdateResourceShare updates a resource share.
func (s *Store) UpdateResourceShare(arn, name string, allowExternal *bool) (*ResourceShare, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	share, ok := s.shares[arn]
	if !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			fmt.Sprintf("Resource share %s not found.", arn), http.StatusNotFound)
	}
	if name != "" {
		share.Name = name
	}
	if allowExternal != nil {
		share.AllowExternalPrincipals = *allowExternal
	}
	share.LastUpdatedTime = time.Now().UTC()
	return share, nil
}

// DeleteResourceShare marks a resource share as deleted.
func (s *Store) DeleteResourceShare(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	share, ok := s.shares[arn]
	if !ok {
		return service.NewAWSError("UnknownResourceException",
			fmt.Sprintf("Resource share %s not found.", arn), http.StatusNotFound)
	}
	share.Status = "DELETED"
	share.LastUpdatedTime = time.Now().UTC()
	return nil
}

// AssociateResourceShare associates principals or resources with a share.
func (s *Store) AssociateResourceShare(arn string, principals, resourceArns []string) ([]ResourceShareAssociation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shares[arn]; !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			fmt.Sprintf("Resource share %s not found.", arn), http.StatusNotFound)
	}
	now := time.Now().UTC()
	var newAssocs []ResourceShareAssociation

	for _, principal := range principals {
		assoc := ResourceShareAssociation{
			ResourceShareArn: arn,
			AssociatedEntity: principal,
			AssociationType:  "PRINCIPAL",
			Status:           "ASSOCIATED",
			CreationTime:     now,
			LastUpdatedTime:  now,
		}
		s.associations = append(s.associations, assoc)
		newAssocs = append(newAssocs, assoc)
	}

	for _, resourceArn := range resourceArns {
		assoc := ResourceShareAssociation{
			ResourceShareArn: arn,
			AssociatedEntity: resourceArn,
			AssociationType:  "RESOURCE",
			Status:           "ASSOCIATED",
			CreationTime:     now,
			LastUpdatedTime:  now,
		}
		s.associations = append(s.associations, assoc)
		newAssocs = append(newAssocs, assoc)
	}

	return newAssocs, nil
}

// DisassociateResourceShare removes associations.
func (s *Store) DisassociateResourceShare(arn string, principals, resourceArns []string) ([]ResourceShareAssociation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shares[arn]; !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			fmt.Sprintf("Resource share %s not found.", arn), http.StatusNotFound)
	}

	removeSet := make(map[string]bool)
	for _, p := range principals {
		removeSet[p] = true
	}
	for _, r := range resourceArns {
		removeSet[r] = true
	}

	var removed []ResourceShareAssociation
	remaining := make([]ResourceShareAssociation, 0, len(s.associations))
	for _, a := range s.associations {
		if a.ResourceShareArn == arn && removeSet[a.AssociatedEntity] {
			a.Status = "DISASSOCIATED"
			removed = append(removed, a)
		} else {
			remaining = append(remaining, a)
		}
	}
	s.associations = remaining
	return removed, nil
}

// GetResourceShareAssociations returns associations for a share or by type.
func (s *Store) GetResourceShareAssociations(associationType string, shareArns []string) []ResourceShareAssociation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	arnSet := make(map[string]bool, len(shareArns))
	for _, arn := range shareArns {
		arnSet[arn] = true
	}
	out := make([]ResourceShareAssociation, 0)
	for _, a := range s.associations {
		if associationType != "" && a.AssociationType != associationType {
			continue
		}
		if len(arnSet) > 0 && !arnSet[a.ResourceShareArn] {
			continue
		}
		out = append(out, a)
	}
	return out
}

// GetResourceShareInvitations returns all invitations.
func (s *Store) GetResourceShareInvitations() []*ResourceShareInvitation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ResourceShareInvitation, 0, len(s.invitations))
	for _, inv := range s.invitations {
		out = append(out, inv)
	}
	return out
}

// AcceptResourceShareInvitation accepts an invitation.
func (s *Store) AcceptResourceShareInvitation(invitationArn string) (*ResourceShareInvitation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invitations[invitationArn]
	if !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			"Invitation not found.", http.StatusNotFound)
	}
	if inv.Status != "PENDING" {
		return nil, service.NewAWSError("ResourceShareInvitationAlreadyAcceptedException",
			"Invitation already processed.", http.StatusBadRequest)
	}
	inv.Status = "ACCEPTED"
	return inv, nil
}

// RejectResourceShareInvitation rejects an invitation.
func (s *Store) RejectResourceShareInvitation(invitationArn string) (*ResourceShareInvitation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invitations[invitationArn]
	if !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			"Invitation not found.", http.StatusNotFound)
	}
	if inv.Status != "PENDING" {
		return nil, service.NewAWSError("ResourceShareInvitationAlreadyRejectedException",
			"Invitation already processed.", http.StatusBadRequest)
	}
	inv.Status = "REJECTED"
	return inv, nil
}

// TagResource adds tags to a resource share.
func (s *Store) TagResource(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	share, ok := s.shares[arn]
	if !ok {
		return service.NewAWSError("UnknownResourceException",
			"Resource not found.", http.StatusNotFound)
	}
	for _, nt := range tags {
		found := false
		for i, et := range share.Tags {
			if et.Key == nt.Key {
				share.Tags[i].Value = nt.Value
				found = true
				break
			}
		}
		if !found {
			share.Tags = append(share.Tags, nt)
		}
	}
	return nil
}

// UntagResource removes tags from a resource share.
func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	share, ok := s.shares[arn]
	if !ok {
		return service.NewAWSError("UnknownResourceException",
			"Resource not found.", http.StatusNotFound)
	}
	keySet := make(map[string]bool, len(tagKeys))
	for _, k := range tagKeys {
		keySet[k] = true
	}
	out := make([]Tag, 0, len(share.Tags))
	for _, t := range share.Tags {
		if !keySet[t.Key] {
			out = append(out, t)
		}
	}
	share.Tags = out
	return nil
}

// ListTagsForResource returns tags for a resource share.
func (s *Store) ListTagsForResource(arn string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	share, ok := s.shares[arn]
	if !ok {
		return nil, service.NewAWSError("UnknownResourceException",
			"Resource not found.", http.StatusNotFound)
	}
	return share.Tags, nil
}
