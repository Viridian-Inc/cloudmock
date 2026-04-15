package rekognition

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredCollection is a face collection.
type StoredCollection struct {
	CollectionID     string
	CollectionArn    string
	FaceModelVersion string
	CreationTime     time.Time
	Tags             map[string]string
}

// StoredFace is a face indexed inside a collection.
type StoredFace struct {
	FaceID                 string
	ImageID                string
	ExternalImageID        string
	BoundingBox            map[string]any
	Confidence             float64
	IndexFacesModelVersion string
	UserID                 string // associated user, if any
}

// StoredUser is a user inside a collection.
type StoredUser struct {
	UserID     string
	UserStatus string
	FaceIDs    []string
}

// StoredProject is an Amazon Rekognition Custom Labels project.
type StoredProject struct {
	ProjectName  string
	ProjectArn   string
	Status       string
	CreationTime time.Time
	Feature      string
	AutoUpdate   string
	Tags         map[string]string
	Policies     map[string]*StoredProjectPolicy
}

// StoredProjectPolicy is a project resource policy.
type StoredProjectPolicy struct {
	PolicyName     string
	PolicyDocument string
	PolicyRevID    string
	CreationTime   time.Time
	LastUpdated    time.Time
}

// StoredProjectVersion is a model version inside a project.
type StoredProjectVersion struct {
	ProjectArn        string
	ProjectVersionArn string
	VersionName       string
	Status            string
	StatusMessage     string
	CreationTime      time.Time
	MinInferenceUnits int
	MaxInferenceUnits int
	KmsKeyID          string
	Tags              map[string]string
	Feature           string
}

// StoredDataset is a Custom Labels dataset.
type StoredDataset struct {
	DatasetArn          string
	DatasetType         string
	ProjectArn          string
	Status              string
	StatusMessage       string
	CreationTime        time.Time
	LastUpdatedTime     time.Time
	Tags                map[string]string
	Stats               map[string]any
	Entries             []map[string]any
	Labels              []string
}

// StoredStreamProcessor is a Kinesis Video stream processor.
type StoredStreamProcessor struct {
	Name               string
	StreamProcessorArn string
	Status             string
	StatusMessage      string
	Input              map[string]any
	Output             map[string]any
	Settings           map[string]any
	RoleArn            string
	KmsKeyID           string
	Notification       map[string]any
	RegionsOfInterest  []map[string]any
	DataSharing        map[string]any
	CreationTime       time.Time
	LastUpdateTime     time.Time
	Tags               map[string]string
}

// StoredFaceLivenessSession is a Face Liveness session.
type StoredFaceLivenessSession struct {
	SessionID    string
	Status       string
	Confidence   float64
	CreationTime time.Time
	KmsKeyID     string
	Settings     map[string]any
}

// StoredMediaAnalysisJob is a media-analysis job (image-based).
type StoredMediaAnalysisJob struct {
	JobID            string
	JobName          string
	JobArn           string
	Status           string
	OperationsConfig map[string]any
	Input            map[string]any
	Output           map[string]any
	Results          map[string]any
	CreationTime     time.Time
	CompletionTime   time.Time
	KmsKeyID         string
	Tags             map[string]string
}

// StoredVideoJob is an async video analysis job covering label/text/etc.
type StoredVideoJob struct {
	JobID          string
	JobType        string // label / text / moderation / face / facesearch / person / celebrity / segment
	Status         string
	StatusMessage  string
	Video          map[string]any
	StartTime      time.Time
	CompletionTime time.Time
	Extra          map[string]any
}

// Store is the in-memory data store for Rekognition.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	collections      map[string]*StoredCollection             // collectionId -> collection
	faces            map[string]map[string]*StoredFace        // collectionId -> faceId -> face
	users            map[string]map[string]*StoredUser        // collectionId -> userId -> user
	projects         map[string]*StoredProject                // arn -> project
	projectVersions  map[string]*StoredProjectVersion         // versionArn -> version
	datasets         map[string]*StoredDataset                // arn -> dataset
	streamProcessors map[string]*StoredStreamProcessor        // name -> processor
	livenessSessions map[string]*StoredFaceLivenessSession    // id -> session
	mediaJobs        map[string]*StoredMediaAnalysisJob       // id -> job
	videoJobs        map[string]*StoredVideoJob               // id -> job
	tags             map[string]map[string]string             // arn -> tags
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:        accountID,
		region:           region,
		collections:      make(map[string]*StoredCollection),
		faces:            make(map[string]map[string]*StoredFace),
		users:            make(map[string]map[string]*StoredUser),
		projects:         make(map[string]*StoredProject),
		projectVersions:  make(map[string]*StoredProjectVersion),
		datasets:         make(map[string]*StoredDataset),
		streamProcessors: make(map[string]*StoredStreamProcessor),
		livenessSessions: make(map[string]*StoredFaceLivenessSession),
		mediaJobs:        make(map[string]*StoredMediaAnalysisJob),
		videoJobs:        make(map[string]*StoredVideoJob),
		tags:             make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state. Satisfies the Resettable interface.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collections = make(map[string]*StoredCollection)
	s.faces = make(map[string]map[string]*StoredFace)
	s.users = make(map[string]map[string]*StoredUser)
	s.projects = make(map[string]*StoredProject)
	s.projectVersions = make(map[string]*StoredProjectVersion)
	s.datasets = make(map[string]*StoredDataset)
	s.streamProcessors = make(map[string]*StoredStreamProcessor)
	s.livenessSessions = make(map[string]*StoredFaceLivenessSession)
	s.mediaJobs = make(map[string]*StoredMediaAnalysisJob)
	s.videoJobs = make(map[string]*StoredVideoJob)
	s.tags = make(map[string]map[string]string)
}

// AccountID returns the account ID for this store.
func (s *Store) AccountID() string { return s.accountID }

// Region returns the region for this store.
func (s *Store) Region() string { return s.region }

// ── Collections ──────────────────────────────────────────────────────────────

func (s *Store) collectionArn(id string) string {
	return fmt.Sprintf("arn:aws:rekognition:%s:%s:collection/%s", s.region, s.accountID, id)
}

// CreateCollection creates a new face collection.
func (s *Store) CreateCollection(id string, tags map[string]string) (*StoredCollection, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[id]; ok {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			"Collection already exists: "+id, http.StatusBadRequest)
	}
	c := &StoredCollection{
		CollectionID:     id,
		CollectionArn:    s.collectionArn(id),
		FaceModelVersion: "6.0",
		CreationTime:     time.Now().UTC(),
		Tags:             copyStringMap(tags),
	}
	s.collections[id] = c
	s.faces[id] = make(map[string]*StoredFace)
	s.users[id] = make(map[string]*StoredUser)
	return c, nil
}

// GetCollection looks up a collection by id.
func (s *Store) GetCollection(id string) (*StoredCollection, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.collections[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+id, http.StatusBadRequest)
	}
	return c, nil
}

// DeleteCollection removes a collection, its faces, and its users.
func (s *Store) DeleteCollection(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+id, http.StatusBadRequest)
	}
	delete(s.collections, id)
	delete(s.faces, id)
	delete(s.users, id)
	return nil
}

// ListCollections returns all collections, sorted by id.
func (s *Store) ListCollections() []*StoredCollection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCollection, 0, len(s.collections))
	for _, c := range s.collections {
		out = append(out, c)
	}
	return out
}

// CountFaces returns the number of faces in a collection.
func (s *Store) CountFaces(collectionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.faces[collectionID])
}

// CountUsers returns the number of users in a collection.
func (s *Store) CountUsers(collectionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users[collectionID])
}

// ── Faces ────────────────────────────────────────────────────────────────────

// IndexFace adds a face to a collection.
func (s *Store) IndexFace(collectionID, externalImageID string) (*StoredFace, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	face := &StoredFace{
		FaceID:                 generateUUID(),
		ImageID:                generateUUID(),
		ExternalImageID:        externalImageID,
		BoundingBox:            sampleBoundingBox(),
		Confidence:             99.85,
		IndexFacesModelVersion: "6.0",
	}
	if s.faces[collectionID] == nil {
		s.faces[collectionID] = make(map[string]*StoredFace)
	}
	s.faces[collectionID][face.FaceID] = face
	return face, nil
}

// ListFaces returns all faces in a collection (optionally filtered by user).
func (s *Store) ListFaces(collectionID, userID string) ([]*StoredFace, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	out := make([]*StoredFace, 0, len(s.faces[collectionID]))
	for _, f := range s.faces[collectionID] {
		if userID != "" && f.UserID != userID {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}

// DeleteFaces removes the given faces from a collection.
func (s *Store) DeleteFaces(collectionID string, faceIDs []string) ([]string, []string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	deleted := make([]string, 0, len(faceIDs))
	failed := make([]string, 0)
	for _, id := range faceIDs {
		if _, ok := s.faces[collectionID][id]; ok {
			delete(s.faces[collectionID], id)
			deleted = append(deleted, id)
		} else {
			failed = append(failed, id)
		}
	}
	return deleted, failed, nil
}

// GetFace returns a face by ID.
func (s *Store) GetFace(collectionID, faceID string) (*StoredFace, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	f, ok := s.faces[collectionID][faceID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Face not found: "+faceID, http.StatusBadRequest)
	}
	return f, nil
}

// AssociateFaces associates faces with a user.
func (s *Store) AssociateFaces(collectionID, userID string, faceIDs []string) ([]map[string]any, []map[string]any, string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, nil, "", service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	user, ok := s.users[collectionID][userID]
	if !ok {
		return nil, nil, "", service.NewAWSError("ResourceNotFoundException",
			"User not found: "+userID, http.StatusBadRequest)
	}
	associated := make([]map[string]any, 0, len(faceIDs))
	unsuccessful := make([]map[string]any, 0)
	for _, fid := range faceIDs {
		f, ok := s.faces[collectionID][fid]
		if !ok {
			unsuccessful = append(unsuccessful, map[string]any{
				"FaceId":  fid,
				"UserId":  userID,
				"Reasons": []string{"FACE_NOT_FOUND"},
			})
			continue
		}
		f.UserID = userID
		user.FaceIDs = appendUnique(user.FaceIDs, fid)
		associated = append(associated, map[string]any{
			"FaceId": fid,
		})
	}
	user.UserStatus = "ACTIVE"
	return associated, unsuccessful, user.UserStatus, nil
}

// DisassociateFaces removes face associations with a user.
func (s *Store) DisassociateFaces(collectionID, userID string, faceIDs []string) ([]map[string]any, []map[string]any, string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, nil, "", service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	user, ok := s.users[collectionID][userID]
	if !ok {
		return nil, nil, "", service.NewAWSError("ResourceNotFoundException",
			"User not found: "+userID, http.StatusBadRequest)
	}
	disassociated := make([]map[string]any, 0, len(faceIDs))
	unsuccessful := make([]map[string]any, 0)
	for _, fid := range faceIDs {
		f, ok := s.faces[collectionID][fid]
		if !ok {
			unsuccessful = append(unsuccessful, map[string]any{
				"FaceId":  fid,
				"UserId":  userID,
				"Reasons": []string{"FACE_NOT_FOUND"},
			})
			continue
		}
		if f.UserID == userID {
			f.UserID = ""
		}
		user.FaceIDs = removeString(user.FaceIDs, fid)
		disassociated = append(disassociated, map[string]any{
			"FaceId": fid,
		})
	}
	return disassociated, unsuccessful, user.UserStatus, nil
}

// ── Users ────────────────────────────────────────────────────────────────────

// CreateUser creates a user inside a collection.
func (s *Store) CreateUser(collectionID, userID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	if _, ok := s.users[collectionID][userID]; ok {
		return service.NewAWSError("ResourceAlreadyExistsException",
			"User already exists: "+userID, http.StatusBadRequest)
	}
	if s.users[collectionID] == nil {
		s.users[collectionID] = make(map[string]*StoredUser)
	}
	s.users[collectionID][userID] = &StoredUser{
		UserID:     userID,
		UserStatus: "CREATED",
		FaceIDs:    make([]string, 0),
	}
	return nil
}

// DeleteUser removes a user.
func (s *Store) DeleteUser(collectionID, userID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collections[collectionID]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	if _, ok := s.users[collectionID][userID]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"User not found: "+userID, http.StatusBadRequest)
	}
	for _, f := range s.faces[collectionID] {
		if f.UserID == userID {
			f.UserID = ""
		}
	}
	delete(s.users[collectionID], userID)
	return nil
}

// ListUsers returns the users in a collection.
func (s *Store) ListUsers(collectionID string) ([]*StoredUser, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.collections[collectionID]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Collection not found: "+collectionID, http.StatusBadRequest)
	}
	out := make([]*StoredUser, 0, len(s.users[collectionID]))
	for _, u := range s.users[collectionID] {
		out = append(out, u)
	}
	return out, nil
}

// ── Projects ─────────────────────────────────────────────────────────────────

func (s *Store) projectArn(name string) string {
	return fmt.Sprintf("arn:aws:rekognition:%s:%s:project/%s/%d",
		s.region, s.accountID, name, time.Now().UnixNano())
}

// CreateProject creates a Custom Labels project.
func (s *Store) CreateProject(name, feature, autoUpdate string, tags map[string]string) (*StoredProject, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.projects {
		if p.ProjectName == name {
			return nil, service.NewAWSError("ResourceAlreadyExistsException",
				"Project already exists: "+name, http.StatusBadRequest)
		}
	}
	if feature == "" {
		feature = "CUSTOM_LABELS"
	}
	if autoUpdate == "" {
		autoUpdate = "DISABLED"
	}
	arn := s.projectArn(name)
	p := &StoredProject{
		ProjectName:  name,
		ProjectArn:   arn,
		Status:       "CREATED",
		CreationTime: time.Now().UTC(),
		Feature:      feature,
		AutoUpdate:   autoUpdate,
		Tags:         copyStringMap(tags),
		Policies:     make(map[string]*StoredProjectPolicy),
	}
	s.projects[arn] = p
	return p, nil
}

// GetProject returns a project by ARN.
func (s *Store) GetProject(arn string) (*StoredProject, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+arn, http.StatusBadRequest)
	}
	return p, nil
}

// DeleteProject removes a project and cascades to versions.
func (s *Store) DeleteProject(arn string) (*StoredProject, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+arn, http.StatusBadRequest)
	}
	for varn, v := range s.projectVersions {
		if v.ProjectArn == arn {
			delete(s.projectVersions, varn)
		}
	}
	for darn, d := range s.datasets {
		if d.ProjectArn == arn {
			delete(s.datasets, darn)
		}
	}
	p.Status = "DELETING"
	delete(s.projects, arn)
	return p, nil
}

// ListProjects returns all projects, optionally filtered by feature and names.
func (s *Store) ListProjects(features, names []string) []*StoredProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProject, 0, len(s.projects))
	for _, p := range s.projects {
		if len(features) > 0 && !containsString(features, p.Feature) {
			continue
		}
		if len(names) > 0 && !containsString(names, p.ProjectName) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// PutProjectPolicy adds or updates a project policy.
func (s *Store) PutProjectPolicy(projectArn, policyName, policyDoc, revID string) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectArn]
	if !ok {
		return "", service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	if p.Policies == nil {
		p.Policies = make(map[string]*StoredProjectPolicy)
	}
	existing, exists := p.Policies[policyName]
	if exists && revID != "" && existing.PolicyRevID != revID {
		return "", service.NewAWSError("InvalidPolicyRevisionIdException",
			"Policy revision id mismatch", http.StatusBadRequest)
	}
	newRev := generateUUID()
	p.Policies[policyName] = &StoredProjectPolicy{
		PolicyName:     policyName,
		PolicyDocument: policyDoc,
		PolicyRevID:    newRev,
		CreationTime:   ifZero(existing, time.Now().UTC()),
		LastUpdated:    time.Now().UTC(),
	}
	return newRev, nil
}

// DeleteProjectPolicy removes a policy by name.
func (s *Store) DeleteProjectPolicy(projectArn, policyName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectArn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	if _, ok := p.Policies[policyName]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Policy not found: "+policyName, http.StatusBadRequest)
	}
	delete(p.Policies, policyName)
	return nil
}

// ListProjectPolicies returns the policies attached to a project.
func (s *Store) ListProjectPolicies(projectArn string) ([]*StoredProjectPolicy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[projectArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	out := make([]*StoredProjectPolicy, 0, len(p.Policies))
	for _, pol := range p.Policies {
		out = append(out, pol)
	}
	return out, nil
}

// ── Project versions ────────────────────────────────────────────────────────

func (s *Store) projectVersionArn(projectArn, version string) string {
	return fmt.Sprintf("%s/version/%s/%d", projectArn, version, time.Now().UnixNano())
}

// CreateProjectVersion adds a new version to a project.
func (s *Store) CreateProjectVersion(projectArn, versionName, kmsKeyID string, tags map[string]string) (*StoredProjectVersion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	for _, v := range s.projectVersions {
		if v.ProjectArn == projectArn && v.VersionName == versionName {
			return nil, service.NewAWSError("ResourceAlreadyExistsException",
				"Project version already exists: "+versionName, http.StatusBadRequest)
		}
	}
	v := &StoredProjectVersion{
		ProjectArn:        projectArn,
		ProjectVersionArn: s.projectVersionArn(projectArn, versionName),
		VersionName:       versionName,
		Status:            "TRAINING_COMPLETED",
		CreationTime:      time.Now().UTC(),
		KmsKeyID:          kmsKeyID,
		Tags:              copyStringMap(tags),
		Feature:           p.Feature,
	}
	s.projectVersions[v.ProjectVersionArn] = v
	return v, nil
}

// CopyProjectVersion copies a version into a destination project.
func (s *Store) CopyProjectVersion(srcArn, destProjectArn, versionName, kmsKeyID string, tags map[string]string) (*StoredProjectVersion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.projectVersions[srcArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Source project version not found: "+srcArn, http.StatusBadRequest)
	}
	if _, ok := s.projects[destProjectArn]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Destination project not found: "+destProjectArn, http.StatusBadRequest)
	}
	v := &StoredProjectVersion{
		ProjectArn:        destProjectArn,
		ProjectVersionArn: s.projectVersionArn(destProjectArn, versionName),
		VersionName:       versionName,
		Status:            "COPYING_COMPLETED",
		CreationTime:      time.Now().UTC(),
		KmsKeyID:          kmsKeyID,
		Tags:              copyStringMap(tags),
		Feature:           src.Feature,
	}
	s.projectVersions[v.ProjectVersionArn] = v
	return v, nil
}

// DeleteProjectVersion removes a project version.
func (s *Store) DeleteProjectVersion(versionArn string) (*StoredProjectVersion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.projectVersions[versionArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project version not found: "+versionArn, http.StatusBadRequest)
	}
	v.Status = "DELETING"
	delete(s.projectVersions, versionArn)
	return v, nil
}

// DescribeProjectVersions returns versions of a project, optionally filtered by name.
func (s *Store) DescribeProjectVersions(projectArn string, versionNames []string) ([]*StoredProjectVersion, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[projectArn]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	out := make([]*StoredProjectVersion, 0)
	for _, v := range s.projectVersions {
		if v.ProjectArn != projectArn {
			continue
		}
		if len(versionNames) > 0 && !containsString(versionNames, v.VersionName) {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// StartProjectVersion runs a version with N inference units.
func (s *Store) StartProjectVersion(versionArn string, minInf, maxInf int) (*StoredProjectVersion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.projectVersions[versionArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project version not found: "+versionArn, http.StatusBadRequest)
	}
	v.Status = "RUNNING"
	v.MinInferenceUnits = minInf
	v.MaxInferenceUnits = maxInf
	return v, nil
}

// StopProjectVersion stops a running version.
func (s *Store) StopProjectVersion(versionArn string) (*StoredProjectVersion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.projectVersions[versionArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project version not found: "+versionArn, http.StatusBadRequest)
	}
	v.Status = "STOPPED"
	return v, nil
}

// ── Datasets ─────────────────────────────────────────────────────────────────

func (s *Store) datasetArn(projectArn, datasetType string) string {
	return fmt.Sprintf("%s/dataset/%s/%d", projectArn, strings.ToLower(datasetType), time.Now().UnixNano())
}

// CreateDataset creates a dataset under a project.
func (s *Store) CreateDataset(projectArn, datasetType string, tags map[string]string) (*StoredDataset, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[projectArn]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Project not found: "+projectArn, http.StatusBadRequest)
	}
	if datasetType != "TRAIN" && datasetType != "TEST" {
		return nil, service.NewAWSError("InvalidParameterException",
			"DatasetType must be TRAIN or TEST", http.StatusBadRequest)
	}
	d := &StoredDataset{
		DatasetArn:      s.datasetArn(projectArn, datasetType),
		DatasetType:     datasetType,
		ProjectArn:      projectArn,
		Status:          "CREATE_COMPLETE",
		StatusMessage:   "Dataset created",
		CreationTime:    time.Now().UTC(),
		LastUpdatedTime: time.Now().UTC(),
		Tags:            copyStringMap(tags),
		Stats: map[string]any{
			"LabeledEntries": 0,
			"TotalEntries":   0,
			"TotalLabels":    0,
			"ErrorEntries":   0,
		},
		Entries: make([]map[string]any, 0),
		Labels:  make([]string, 0),
	}
	s.datasets[d.DatasetArn] = d
	return d, nil
}

// GetDataset returns a dataset by ARN.
func (s *Store) GetDataset(arn string) (*StoredDataset, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.datasets[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, http.StatusBadRequest)
	}
	return d, nil
}

// DeleteDataset removes a dataset.
func (s *Store) DeleteDataset(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, http.StatusBadRequest)
	}
	delete(s.datasets, arn)
	return nil
}

// UpdateDatasetEntries appends ground-truth entries to a dataset.
func (s *Store) UpdateDatasetEntries(arn string, entries []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.datasets[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, http.StatusBadRequest)
	}
	d.Entries = append(d.Entries, entries...)
	d.LastUpdatedTime = time.Now().UTC()
	if stats, ok := d.Stats["TotalEntries"].(int); ok {
		d.Stats["TotalEntries"] = stats + len(entries)
	} else {
		d.Stats["TotalEntries"] = len(entries)
	}
	return nil
}

// DistributeDatasetEntries marks a distribute-to-test as complete.
func (s *Store) DistributeDatasetEntries(arns []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range arns {
		if d, ok := s.datasets[a]; ok {
			d.Status = "UPDATE_COMPLETE"
			d.LastUpdatedTime = time.Now().UTC()
		}
	}
	return nil
}

// ListDatasetEntries returns entries for a dataset.
func (s *Store) ListDatasetEntries(arn string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.datasets[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, http.StatusBadRequest)
	}
	return d.Entries, nil
}

// ListDatasetLabels returns labels for a dataset.
func (s *Store) ListDatasetLabels(arn string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.datasets[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, http.StatusBadRequest)
	}
	return d.Labels, nil
}

// ── Stream processors ────────────────────────────────────────────────────────

func (s *Store) streamProcessorArn(name string) string {
	return fmt.Sprintf("arn:aws:rekognition:%s:%s:streamprocessor/%s", s.region, s.accountID, name)
}

// CreateStreamProcessor creates a stream processor.
func (s *Store) CreateStreamProcessor(name string, input, output, settings, notification map[string]any, regions []map[string]any, dataSharing map[string]any, roleArn, kmsKeyID string, tags map[string]string) (*StoredStreamProcessor, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.streamProcessors[name]; ok {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			"Stream processor already exists: "+name, http.StatusBadRequest)
	}
	sp := &StoredStreamProcessor{
		Name:               name,
		StreamProcessorArn: s.streamProcessorArn(name),
		Status:             "STOPPED",
		StatusMessage:      "",
		Input:              input,
		Output:             output,
		Settings:           settings,
		Notification:       notification,
		RegionsOfInterest:  regions,
		DataSharing:        dataSharing,
		RoleArn:            roleArn,
		KmsKeyID:           kmsKeyID,
		CreationTime:       time.Now().UTC(),
		LastUpdateTime:     time.Now().UTC(),
		Tags:               copyStringMap(tags),
	}
	s.streamProcessors[name] = sp
	return sp, nil
}

// GetStreamProcessor returns a stream processor by name.
func (s *Store) GetStreamProcessor(name string) (*StoredStreamProcessor, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sp, ok := s.streamProcessors[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Stream processor not found: "+name, http.StatusBadRequest)
	}
	return sp, nil
}

// DeleteStreamProcessor removes a stream processor.
func (s *Store) DeleteStreamProcessor(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.streamProcessors[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Stream processor not found: "+name, http.StatusBadRequest)
	}
	if sp.Status == "RUNNING" {
		return service.NewAWSError("ResourceInUseException",
			"Stream processor must be stopped before deletion", http.StatusBadRequest)
	}
	delete(s.streamProcessors, name)
	return nil
}

// ListStreamProcessors returns all stream processors.
func (s *Store) ListStreamProcessors() []*StoredStreamProcessor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredStreamProcessor, 0, len(s.streamProcessors))
	for _, sp := range s.streamProcessors {
		out = append(out, sp)
	}
	return out
}

// StartStreamProcessor transitions to RUNNING.
func (s *Store) StartStreamProcessor(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.streamProcessors[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Stream processor not found: "+name, http.StatusBadRequest)
	}
	sp.Status = "RUNNING"
	sp.LastUpdateTime = time.Now().UTC()
	return nil
}

// StopStreamProcessor transitions to STOPPED.
func (s *Store) StopStreamProcessor(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.streamProcessors[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Stream processor not found: "+name, http.StatusBadRequest)
	}
	sp.Status = "STOPPED"
	sp.LastUpdateTime = time.Now().UTC()
	return nil
}

// UpdateStreamProcessor patches in update fields.
func (s *Store) UpdateStreamProcessor(name string, settings map[string]any, regions []map[string]any, dataSharing map[string]any, paramsToDelete []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.streamProcessors[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Stream processor not found: "+name, http.StatusBadRequest)
	}
	if settings != nil {
		sp.Settings = settings
	}
	if regions != nil {
		sp.RegionsOfInterest = regions
	}
	if dataSharing != nil {
		sp.DataSharing = dataSharing
	}
	for _, p := range paramsToDelete {
		switch p {
		case "ConnectedHomeMinConfidence":
			if sp.Settings != nil {
				if ch, ok := sp.Settings["ConnectedHome"].(map[string]any); ok {
					delete(ch, "MinConfidence")
				}
			}
		}
	}
	sp.LastUpdateTime = time.Now().UTC()
	return nil
}

// ── Face liveness sessions ───────────────────────────────────────────────────

// CreateFaceLivenessSession returns a new session ID.
func (s *Store) CreateFaceLivenessSession(kmsKeyID string, settings map[string]any) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateUUID()
	s.livenessSessions[id] = &StoredFaceLivenessSession{
		SessionID:    id,
		Status:       "SUCCEEDED",
		Confidence:   99.5,
		CreationTime: time.Now().UTC(),
		KmsKeyID:     kmsKeyID,
		Settings:     settings,
	}
	return id
}

// GetFaceLivenessSession returns a session by ID.
func (s *Store) GetFaceLivenessSession(id string) (*StoredFaceLivenessSession, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.livenessSessions[id]
	if !ok {
		return nil, service.NewAWSError("SessionNotFoundException",
			"Face liveness session not found: "+id, http.StatusBadRequest)
	}
	return sess, nil
}

// ── Media analysis jobs ──────────────────────────────────────────────────────

func (s *Store) mediaJobArn(id string) string {
	return fmt.Sprintf("arn:aws:rekognition:%s:%s:mediaanalysisjob/%s", s.region, s.accountID, id)
}

// CreateMediaAnalysisJob records a new job.
func (s *Store) CreateMediaAnalysisJob(name, kmsKeyID string, opsConfig, input, output map[string]any, tags map[string]string) *StoredMediaAnalysisJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateUUID()
	job := &StoredMediaAnalysisJob{
		JobID:            id,
		JobName:          name,
		JobArn:           s.mediaJobArn(id),
		Status:           "SUCCEEDED",
		OperationsConfig: opsConfig,
		Input:            input,
		Output:           output,
		KmsKeyID:         kmsKeyID,
		CreationTime:     time.Now().UTC(),
		CompletionTime:   time.Now().UTC(),
		Tags:             copyStringMap(tags),
		Results: map[string]any{
			"S3Object": map[string]any{
				"Bucket": "cloudmock-results",
				"Name":   id + "/results.json",
			},
		},
	}
	s.mediaJobs[id] = job
	return job
}

// GetMediaAnalysisJob returns a media job by ID.
func (s *Store) GetMediaAnalysisJob(id string) (*StoredMediaAnalysisJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.mediaJobs[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Media analysis job not found: "+id, http.StatusBadRequest)
	}
	return job, nil
}

// ListMediaAnalysisJobs returns all media jobs.
func (s *Store) ListMediaAnalysisJobs() []*StoredMediaAnalysisJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMediaAnalysisJob, 0, len(s.mediaJobs))
	for _, j := range s.mediaJobs {
		out = append(out, j)
	}
	return out
}

// ── Async video jobs ─────────────────────────────────────────────────────────

// StartVideoJob creates an async video job that immediately succeeds.
func (s *Store) StartVideoJob(jobType string, video map[string]any, extra map[string]any) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateUUID()
	now := time.Now().UTC()
	s.videoJobs[id] = &StoredVideoJob{
		JobID:          id,
		JobType:        jobType,
		Status:         "SUCCEEDED",
		Video:          video,
		StartTime:      now,
		CompletionTime: now,
		Extra:          extra,
	}
	return id
}

// GetVideoJob returns a video job by ID.
func (s *Store) GetVideoJob(id string) (*StoredVideoJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.videoJobs[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Video job not found: "+id, http.StatusBadRequest)
	}
	return job, nil
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// TagResource attaches tags to an ARN.
func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

// UntagResource removes tag keys from an ARN.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns all tags for an ARN.
func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.tags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func removeString(list []string, v string) []string {
	out := make([]string, 0, len(list))
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func sampleBoundingBox() map[string]any {
	return map[string]any{
		"Width":  0.5,
		"Height": 0.5,
		"Left":   0.25,
		"Top":    0.25,
	}
}

func ifZero(existing *StoredProjectPolicy, fallback time.Time) time.Time {
	if existing == nil {
		return fallback
	}
	return existing.CreationTime
}
