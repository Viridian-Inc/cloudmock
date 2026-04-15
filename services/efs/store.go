package efs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredFileSystem is the persisted shape of an EFS file system.
type StoredFileSystem struct {
	FileSystemID                   string
	Arn                            string
	Name                           string
	CreationToken                  string
	CreationTime                   time.Time
	LifeCycleState                 string
	NumberOfMountTargets           int
	OwnerID                        string
	PerformanceMode                string
	ThroughputMode                 string
	ProvisionedThroughputInMibps   float64
	Encrypted                      bool
	KmsKeyID                       string
	AvailabilityZoneName           string
	AvailabilityZoneID             string
	Tags                           map[string]string
	ReplicationOverwriteProtection string
	SizeValue                      int64
	SizeValueInIA                  int64
	SizeValueInArchive             int64
	SizeValueInStandard            int64
	// Associated configurations
	Policy            string
	BackupStatus      string
	LifecyclePolicies []map[string]any
}

// StoredMountTarget is the persisted shape of an EFS mount target.
type StoredMountTarget struct {
	MountTargetID         string
	FileSystemID          string
	SubnetID              string
	LifeCycleState        string
	IPAddress             string
	IPv6Address           string
	NetworkInterfaceID    string
	AvailabilityZoneName  string
	AvailabilityZoneID    string
	VpcID                 string
	OwnerID               string
	SecurityGroups        []string
}

// StoredAccessPoint is the persisted shape of an EFS access point.
type StoredAccessPoint struct {
	AccessPointID  string
	AccessPointArn string
	ClientToken    string
	FileSystemID   string
	Name           string
	OwnerID        string
	LifeCycleState string
	PosixUser      map[string]any
	RootDirectory  map[string]any
	Tags           map[string]string
}

// StoredReplication is the persisted shape of an EFS replication configuration.
type StoredReplication struct {
	SourceFileSystemID      string
	SourceFileSystemArn     string
	SourceFileSystemRegion  string
	SourceFileSystemOwnerID string
	OriginalSourceArn       string
	CreationTime            time.Time
	Destinations            []map[string]any
}

// StoredAccountPreferences is the per-region setting for ResourceIdType.
type StoredAccountPreferences struct {
	ResourceIDType string
	Resources      []string
}

// ── Store ────────────────────────────────────────────────────────────────────

// Store is the in-memory data store for efs resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	fileSystems  map[string]*StoredFileSystem  // FileSystemId -> file system
	mountTargets map[string]*StoredMountTarget // MountTargetId -> mount target
	accessPoints map[string]*StoredAccessPoint // AccessPointId -> access point
	replications map[string]*StoredReplication // SourceFileSystemId -> replication

	preferences *StoredAccountPreferences
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:    accountID,
		region:       region,
		fileSystems:  make(map[string]*StoredFileSystem),
		mountTargets: make(map[string]*StoredMountTarget),
		accessPoints: make(map[string]*StoredAccessPoint),
		replications: make(map[string]*StoredReplication),
		preferences:  &StoredAccountPreferences{ResourceIDType: "LONG_ID"},
	}
}

// Reset clears all in-memory state. Satisfies the Resettable interface used by
// the admin API.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileSystems = make(map[string]*StoredFileSystem)
	s.mountTargets = make(map[string]*StoredMountTarget)
	s.accessPoints = make(map[string]*StoredAccessPoint)
	s.replications = make(map[string]*StoredReplication)
	s.preferences = &StoredAccountPreferences{ResourceIDType: "LONG_ID"}
}

// ── File systems ────────────────────────────────────────────────────────────

// CreateFileSystem persists a new file system. CreationToken acts as an
// idempotency key — re-using a token returns the existing file system instead
// of creating a duplicate.
func (s *Store) CreateFileSystem(fs *StoredFileSystem) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fs.CreationToken != "" {
		for _, existing := range s.fileSystems {
			if existing.CreationToken == fs.CreationToken {
				return nil, service.NewAWSError("FileSystemAlreadyExists",
					"File system with CreationToken "+fs.CreationToken+" already exists.", 409)
			}
		}
	}

	if fs.FileSystemID == "" {
		fs.FileSystemID = newFileSystemID()
	}
	fs.Arn = fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:file-system/%s", s.region, s.accountID, fs.FileSystemID)
	if fs.CreationTime.IsZero() {
		fs.CreationTime = time.Now().UTC()
	}
	if fs.LifeCycleState == "" {
		fs.LifeCycleState = "available"
	}
	if fs.PerformanceMode == "" {
		fs.PerformanceMode = "generalPurpose"
	}
	if fs.ThroughputMode == "" {
		fs.ThroughputMode = "bursting"
	}
	if fs.OwnerID == "" {
		fs.OwnerID = s.accountID
	}
	if fs.Tags == nil {
		fs.Tags = make(map[string]string)
	}
	if fs.BackupStatus == "" {
		fs.BackupStatus = "DISABLED"
	}
	if fs.ReplicationOverwriteProtection == "" {
		fs.ReplicationOverwriteProtection = "ENABLED"
	}

	s.fileSystems[fs.FileSystemID] = fs
	return fs, nil
}

// GetFileSystem returns a file system by ID.
func (s *Store) GetFileSystem(id string) (*StoredFileSystem, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	return fs, nil
}

// DeleteFileSystem removes a file system by ID and any associated state.
func (s *Store) DeleteFileSystem(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.fileSystems[id]; !ok {
		return service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	// Mount targets must be deleted first per real EFS semantics.
	for _, mt := range s.mountTargets {
		if mt.FileSystemID == id {
			return service.NewAWSError("FileSystemInUse",
				"File system "+id+" has mount targets and cannot be deleted.", 409)
		}
	}
	delete(s.fileSystems, id)
	// Drop dependent state.
	for apID, ap := range s.accessPoints {
		if ap.FileSystemID == id {
			delete(s.accessPoints, apID)
		}
	}
	delete(s.replications, id)
	return nil
}

// ListFileSystems returns all file systems, optionally filtered by id or
// creation token.
func (s *Store) ListFileSystems(filterID, filterToken string) []*StoredFileSystem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFileSystem, 0, len(s.fileSystems))
	for _, fs := range s.fileSystems {
		if filterID != "" && fs.FileSystemID != filterID {
			continue
		}
		if filterToken != "" && fs.CreationToken != filterToken {
			continue
		}
		out = append(out, fs)
	}
	return out
}

// UpdateFileSystem changes the throughput configuration on an existing file
// system.
func (s *Store) UpdateFileSystem(id, throughputMode string, provisionedThroughput float64) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	if throughputMode != "" {
		fs.ThroughputMode = throughputMode
	}
	if provisionedThroughput > 0 {
		fs.ProvisionedThroughputInMibps = provisionedThroughput
	}
	return fs, nil
}

// UpdateFileSystemProtection sets ReplicationOverwriteProtection on a file
// system.
func (s *Store) UpdateFileSystemProtection(id, protection string) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	if protection != "" {
		fs.ReplicationOverwriteProtection = protection
	}
	return fs, nil
}

// SetFileSystemPolicy attaches a JSON policy doc to a file system.
func (s *Store) SetFileSystemPolicy(id, policy string) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	fs.Policy = policy
	return fs, nil
}

// DeleteFileSystemPolicy removes the policy from a file system.
func (s *Store) DeleteFileSystemPolicy(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	fs.Policy = ""
	return nil
}

// SetBackupStatus updates the backup status of a file system.
func (s *Store) SetBackupStatus(id, status string) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	if status != "" {
		fs.BackupStatus = status
	}
	return fs, nil
}

// SetLifecyclePolicies replaces the lifecycle policies on a file system.
func (s *Store) SetLifecyclePolicies(id string, policies []map[string]any) (*StoredFileSystem, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	fs.LifecyclePolicies = policies
	return fs, nil
}

// MergeTags adds the given tags to an existing file system.
func (s *Store) MergeTags(id string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	if fs.Tags == nil {
		fs.Tags = make(map[string]string)
	}
	for k, v := range tags {
		fs.Tags[k] = v
	}
	return nil
}

// RemoveTags drops the given tag keys from a file system.
func (s *Store) RemoveTags(id string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[id]
	if !ok {
		return service.NewAWSError("FileSystemNotFound",
			"File system not found: "+id, 404)
	}
	for _, k := range keys {
		delete(fs.Tags, k)
	}
	return nil
}

// ── Tags by ARN/ResourceId ──────────────────────────────────────────────────

// resolveResource locates the tag map for a given resource id or arn.
func (s *Store) resolveResource(resourceID string) (map[string]string, *service.AWSError) {
	if resourceID == "" {
		return nil, service.ErrValidation("ResourceId is required.")
	}
	if fs, ok := s.fileSystems[resourceID]; ok {
		if fs.Tags == nil {
			fs.Tags = make(map[string]string)
		}
		return fs.Tags, nil
	}
	if ap, ok := s.accessPoints[resourceID]; ok {
		if ap.Tags == nil {
			ap.Tags = make(map[string]string)
		}
		return ap.Tags, nil
	}
	// Try matching by ARN.
	for _, fs := range s.fileSystems {
		if fs.Arn == resourceID {
			if fs.Tags == nil {
				fs.Tags = make(map[string]string)
			}
			return fs.Tags, nil
		}
	}
	for _, ap := range s.accessPoints {
		if ap.AccessPointArn == resourceID {
			if ap.Tags == nil {
				ap.Tags = make(map[string]string)
			}
			return ap.Tags, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		"Resource not found: "+resourceID, 404)
}

// TagResource merges tags onto any tagged resource (file system or access point).
func (s *Store) TagResource(resourceID string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, err := s.resolveResource(resourceID)
	if err != nil {
		return err
	}
	for k, v := range tags {
		target[k] = v
	}
	return nil
}

// UntagResource removes tag keys from a tagged resource.
func (s *Store) UntagResource(resourceID string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, err := s.resolveResource(resourceID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		delete(target, k)
	}
	return nil
}

// ListTags returns the tags for a file system or access point.
func (s *Store) ListTags(resourceID string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, err := s.resolveResource(resourceID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(target))
	for k, v := range target {
		out[k] = v
	}
	return out, nil
}

// ── Mount targets ───────────────────────────────────────────────────────────

// CreateMountTarget creates a new mount target attached to a file system.
func (s *Store) CreateMountTarget(mt *StoredMountTarget) (*StoredMountTarget, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, ok := s.fileSystems[mt.FileSystemID]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+mt.FileSystemID, 404)
	}
	if mt.MountTargetID == "" {
		mt.MountTargetID = newMountTargetID()
	}
	if mt.LifeCycleState == "" {
		mt.LifeCycleState = "available"
	}
	if mt.IPAddress == "" {
		mt.IPAddress = "10.0.0." + hex.EncodeToString([]byte{byte(len(s.mountTargets) + 10)})[:2]
	}
	if mt.NetworkInterfaceID == "" {
		mt.NetworkInterfaceID = "eni-" + newShortID()
	}
	if mt.AvailabilityZoneName == "" {
		mt.AvailabilityZoneName = s.region + "a"
	}
	if mt.AvailabilityZoneID == "" {
		mt.AvailabilityZoneID = "use1-az1"
	}
	if mt.VpcID == "" {
		mt.VpcID = "vpc-" + newShortID()
	}
	if mt.OwnerID == "" {
		mt.OwnerID = s.accountID
	}
	s.mountTargets[mt.MountTargetID] = mt
	fs.NumberOfMountTargets++
	return mt, nil
}

// GetMountTarget returns a mount target by ID.
func (s *Store) GetMountTarget(id string) (*StoredMountTarget, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mt, ok := s.mountTargets[id]
	if !ok {
		return nil, service.NewAWSError("MountTargetNotFound",
			"Mount target not found: "+id, 404)
	}
	return mt, nil
}

// DeleteMountTarget removes a mount target by ID.
func (s *Store) DeleteMountTarget(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	mt, ok := s.mountTargets[id]
	if !ok {
		return service.NewAWSError("MountTargetNotFound",
			"Mount target not found: "+id, 404)
	}
	delete(s.mountTargets, id)
	if fs, ok := s.fileSystems[mt.FileSystemID]; ok {
		if fs.NumberOfMountTargets > 0 {
			fs.NumberOfMountTargets--
		}
	}
	return nil
}

// ListMountTargets returns mount targets filtered by file system or by id.
func (s *Store) ListMountTargets(fileSystemID, mountTargetID string) []*StoredMountTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMountTarget, 0, len(s.mountTargets))
	for _, mt := range s.mountTargets {
		if fileSystemID != "" && mt.FileSystemID != fileSystemID {
			continue
		}
		if mountTargetID != "" && mt.MountTargetID != mountTargetID {
			continue
		}
		out = append(out, mt)
	}
	return out
}

// SetMountTargetSecurityGroups overrides the security groups on a mount target.
func (s *Store) SetMountTargetSecurityGroups(id string, sgs []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	mt, ok := s.mountTargets[id]
	if !ok {
		return service.NewAWSError("MountTargetNotFound",
			"Mount target not found: "+id, 404)
	}
	mt.SecurityGroups = append([]string(nil), sgs...)
	return nil
}

// ── Access points ───────────────────────────────────────────────────────────

// CreateAccessPoint creates an access point bound to a file system. ClientToken
// behaves as an idempotency key.
func (s *Store) CreateAccessPoint(ap *StoredAccessPoint) (*StoredAccessPoint, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.fileSystems[ap.FileSystemID]; !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+ap.FileSystemID, 404)
	}
	if ap.ClientToken != "" {
		for _, existing := range s.accessPoints {
			if existing.ClientToken == ap.ClientToken {
				return nil, service.NewAWSError("AccessPointAlreadyExists",
					"Access point with ClientToken "+ap.ClientToken+" already exists.", 409)
			}
		}
	}
	if ap.AccessPointID == "" {
		ap.AccessPointID = newAccessPointID()
	}
	ap.AccessPointArn = fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:access-point/%s", s.region, s.accountID, ap.AccessPointID)
	if ap.LifeCycleState == "" {
		ap.LifeCycleState = "available"
	}
	if ap.OwnerID == "" {
		ap.OwnerID = s.accountID
	}
	if ap.Tags == nil {
		ap.Tags = make(map[string]string)
	}
	s.accessPoints[ap.AccessPointID] = ap
	return ap, nil
}

// DeleteAccessPoint removes an access point by id.
func (s *Store) DeleteAccessPoint(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accessPoints[id]; !ok {
		return service.NewAWSError("AccessPointNotFound",
			"Access point not found: "+id, 404)
	}
	delete(s.accessPoints, id)
	return nil
}

// ListAccessPoints returns access points filtered by file system or access
// point id.
func (s *Store) ListAccessPoints(fileSystemID, accessPointID string) []*StoredAccessPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAccessPoint, 0, len(s.accessPoints))
	for _, ap := range s.accessPoints {
		if fileSystemID != "" && ap.FileSystemID != fileSystemID {
			continue
		}
		if accessPointID != "" && ap.AccessPointID != accessPointID {
			continue
		}
		out = append(out, ap)
	}
	return out
}

// ── Replication configurations ──────────────────────────────────────────────

// CreateReplication creates or replaces a replication configuration for a
// source file system.
func (s *Store) CreateReplication(rep *StoredReplication) (*StoredReplication, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.fileSystems[rep.SourceFileSystemID]
	if !ok {
		return nil, service.NewAWSError("FileSystemNotFound",
			"File system not found: "+rep.SourceFileSystemID, 404)
	}
	if _, exists := s.replications[rep.SourceFileSystemID]; exists {
		return nil, service.NewAWSError("ReplicationAlreadyExists",
			"Replication configuration already exists for "+rep.SourceFileSystemID, 409)
	}
	rep.SourceFileSystemArn = src.Arn
	if rep.OriginalSourceArn == "" {
		rep.OriginalSourceArn = src.Arn
	}
	if rep.SourceFileSystemRegion == "" {
		rep.SourceFileSystemRegion = s.region
	}
	if rep.SourceFileSystemOwnerID == "" {
		rep.SourceFileSystemOwnerID = s.accountID
	}
	if rep.CreationTime.IsZero() {
		rep.CreationTime = time.Now().UTC()
	}
	// Materialise destinations into stable map shape.
	for i, dest := range rep.Destinations {
		if dest == nil {
			continue
		}
		if _, ok := dest["FileSystemId"]; !ok {
			dest["FileSystemId"] = newFileSystemID()
		}
		if _, ok := dest["Region"]; !ok {
			dest["Region"] = s.region
		}
		if _, ok := dest["Status"]; !ok {
			dest["Status"] = "ENABLED"
		}
		if _, ok := dest["LastReplicatedTimestamp"]; !ok {
			dest["LastReplicatedTimestamp"] = rep.CreationTime.Format(time.RFC3339)
		}
		rep.Destinations[i] = dest
	}
	s.replications[rep.SourceFileSystemID] = rep
	return rep, nil
}

// DeleteReplication removes a replication configuration by source file system id.
func (s *Store) DeleteReplication(sourceID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.replications[sourceID]; !ok {
		return service.NewAWSError("ReplicationNotFound",
			"Replication configuration not found for "+sourceID, 404)
	}
	delete(s.replications, sourceID)
	return nil
}

// ListReplications returns replication configurations, optionally filtered by
// source file system id.
func (s *Store) ListReplications(sourceID string) []*StoredReplication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredReplication, 0, len(s.replications))
	for _, r := range s.replications {
		if sourceID != "" && r.SourceFileSystemID != sourceID {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ── Account preferences ─────────────────────────────────────────────────────

// GetAccountPreferences returns the per-region account preferences.
func (s *Store) GetAccountPreferences() *StoredAccountPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.preferences
}

// PutAccountPreferences sets the account preferences for the region.
func (s *Store) PutAccountPreferences(resourceIDType string) *StoredAccountPreferences {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preferences = &StoredAccountPreferences{
		ResourceIDType: resourceIDType,
		Resources:      []string{"FILE_SYSTEM", "MOUNT_TARGET", "ACCESS_POINT"},
	}
	return s.preferences
}

// ── ID helpers ──────────────────────────────────────────────────────────────

func newFileSystemID() string  { return "fs-" + newShortID() }
func newMountTargetID() string { return "fsmt-" + newShortID() }
func newAccessPointID() string { return "fsap-" + newShortID() }

func newShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to monotonic fill if /dev/urandom fails; not security
		// critical for a mock.
		now := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(now >> (i * 8))
		}
	}
	return hex.EncodeToString(b)
}
