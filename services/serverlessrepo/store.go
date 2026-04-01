package serverlessrepo

import (
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9\.\-]+)?(\+[a-zA-Z0-9\.\-]+)?$`)

// Application represents a Serverless Application Repository application.
type Application struct {
	ApplicationID string
	Arn           string
	Name          string
	Description   string
	Author        string
	SpdxLicenseID string
	Labels        []string
	HomePageURL   string
	SemanticVersion string
	CreationTime  time.Time
}

// ApplicationVersion represents a version of an application.
type ApplicationVersion struct {
	ApplicationID   string
	SemanticVersion string
	TemplateURL     string
	CreationTime    time.Time
	SourceCodeURL   string
}

// ChangeSet represents a change set for deploying an application.
type ChangeSet struct {
	ChangeSetID   string
	ApplicationID string
	SemanticVersion string
	StackID       string
	CreationTime  time.Time
}

// Store manages Serverless Application Repository resources in memory.
type Store struct {
	mu        sync.RWMutex
	apps      map[string]*Application
	versions  map[string][]*ApplicationVersion // appID -> versions
	changeSets map[string]*ChangeSet
	accountID string
	region    string
	appSeq    int
	csSeq     int
}

// NewStore returns a new empty Serverless Application Repository Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		apps:       make(map[string]*Application),
		versions:   make(map[string][]*ApplicationVersion),
		changeSets: make(map[string]*ChangeSet),
		accountID:  accountID,
		region:     region,
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:serverlessrepo:%s:%s:", s.region, s.accountID)
}

// CreateApplication creates a new application.
func (s *Store) CreateApplication(name, description, author, spdxLicense, homePageURL, semanticVersion string, labels []string) (*Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appSeq++
	id := fmt.Sprintf("app-%012d", s.appSeq)

	app := &Application{
		ApplicationID:   id,
		Arn:              s.arnPrefix() + "applications/" + id,
		Name:             name,
		Description:      description,
		Author:           author,
		SpdxLicenseID:    spdxLicense,
		Labels:           labels,
		HomePageURL:      homePageURL,
		SemanticVersion:  semanticVersion,
		CreationTime:     time.Now().UTC(),
	}
	s.apps[id] = app

	// Create initial version
	if semanticVersion != "" {
		s.versions[id] = []*ApplicationVersion{
			{
				ApplicationID:   id,
				SemanticVersion: semanticVersion,
				TemplateURL:     fmt.Sprintf("https://s3.amazonaws.com/serverlessrepo/%s/%s/template.yaml", id, semanticVersion),
				CreationTime:    time.Now().UTC(),
			},
		}
	} else {
		s.versions[id] = []*ApplicationVersion{}
	}

	return app, nil
}

// GetApplication retrieves an application.
func (s *Store) GetApplication(id string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[id]
	return app, ok
}

// ListApplications returns all applications.
func (s *Store) ListApplications() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Application, 0, len(s.apps))
	for _, app := range s.apps {
		out = append(out, app)
	}
	return out
}

// DeleteApplication removes an application.
func (s *Store) DeleteApplication(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[id]; !ok {
		return false
	}
	delete(s.apps, id)
	delete(s.versions, id)
	return true
}

// CreateApplicationVersion creates a new version.
func (s *Store) CreateApplicationVersion(appID, semanticVersion, sourceCodeURL string) (*ApplicationVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}
	if semanticVersion != "" && !semverRegex.MatchString(semanticVersion) {
		return nil, fmt.Errorf("invalid semantic version: %s (must match X.Y.Z format)", semanticVersion)
	}
	// Check for duplicate version
	for _, v := range s.versions[appID] {
		if v.SemanticVersion == semanticVersion {
			return nil, fmt.Errorf("version already exists: %s", semanticVersion)
		}
	}

	ver := &ApplicationVersion{
		ApplicationID:   appID,
		SemanticVersion: semanticVersion,
		TemplateURL:     fmt.Sprintf("https://s3.amazonaws.com/serverlessrepo/%s/%s/template.yaml", appID, semanticVersion),
		SourceCodeURL:   sourceCodeURL,
		CreationTime:    time.Now().UTC(),
	}
	s.versions[appID] = append(s.versions[appID], ver)
	s.apps[appID].SemanticVersion = semanticVersion
	return ver, nil
}

// ListApplicationVersions returns all versions for an application, sorted by version.
func (s *Store) ListApplicationVersions(appID string) []*ApplicationVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := make([]*ApplicationVersion, len(s.versions[appID]))
	copy(versions, s.versions[appID])
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].SemanticVersion < versions[j].SemanticVersion
	})
	return versions
}

// CreateCloudFormationChangeSet creates a change set.
func (s *Store) CreateCloudFormationChangeSet(appID, semanticVersion, stackName string) (*ChangeSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	s.csSeq++
	csID := fmt.Sprintf("cs-%012d", s.csSeq)
	stackID := fmt.Sprintf("arn:aws:cloudformation:%s:%s:stack/%s/%s", s.region, s.accountID, stackName, csID)

	cs := &ChangeSet{
		ChangeSetID:     csID,
		ApplicationID:   appID,
		SemanticVersion: semanticVersion,
		StackID:         stackID,
		CreationTime:    time.Now().UTC(),
	}
	s.changeSets[csID] = cs
	return cs, nil
}
