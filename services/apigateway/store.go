package apigateway

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- random ID helpers ----

// newID returns a random alphanumeric ID of the given length.
func newID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]byte, n)
	for i, v := range b {
		out[i] = chars[int(v)%len(chars)]
	}
	return string(out)
}

// ---- data types ----

// Integration holds the integration configuration for an API method.
type Integration struct {
	Type                string `json:"type"`
	Uri                 string `json:"uri"`
	HttpMethod          string `json:"httpMethod"`
	IntegrationHttpMethod string `json:"integrationHttpMethod,omitempty"`
}

// Method holds metadata for an API Gateway resource method.
type Method struct {
	HttpMethod        string       `json:"httpMethod"`
	AuthorizationType string       `json:"authorizationType"`
	Integration       *Integration `json:"methodIntegration,omitempty"`
}

// Resource holds an API Gateway resource node.
type Resource struct {
	Id       string             `json:"id"`
	ParentId string             `json:"parentId"`
	PathPart string             `json:"pathPart"`
	Path     string             `json:"path"`
	Methods  map[string]*Method `json:"-"`
}

// Deployment holds a REST API deployment.
type Deployment struct {
	Id          string    `json:"id"`
	CreatedDate time.Time `json:"createdDate"`
	Description string    `json:"description"`
}

// Stage holds a REST API stage.
type Stage struct {
	StageName    string    `json:"stageName"`
	DeploymentId string    `json:"deploymentId"`
	Description  string    `json:"description"`
	CreatedDate  time.Time `json:"createdDate"`
}

// RestApi is the top-level API Gateway REST API.
type RestApi struct {
	Id          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	CreatedDate time.Time             `json:"createdDate"`
	Resources   map[string]*Resource  `json:"-"`
	Deployments map[string]*Deployment `json:"-"`
	Stages      map[string]*Stage     `json:"-"`
}

// ---- store ----

// Store is the in-memory store for API Gateway resources.
type Store struct {
	mu        sync.RWMutex
	apis      map[string]*RestApi // keyed by API id
	accountID string
	region    string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		apis:      make(map[string]*RestApi),
		accountID: accountID,
		region:    region,
	}
}

// ---- RestApi operations ----

// CreateRestApi creates a new REST API and returns it.
func (s *Store) CreateRestApi(name, description string) *RestApi {
	id := newID(10)
	rootID := newID(6)
	api := &RestApi{
		Id:          id,
		Name:        name,
		Description: description,
		CreatedDate: time.Now().UTC(),
		Resources: map[string]*Resource{
			rootID: {
				Id:       rootID,
				ParentId: "",
				PathPart: "",
				Path:     "/",
				Methods:  make(map[string]*Method),
			},
		},
		Deployments: make(map[string]*Deployment),
		Stages:      make(map[string]*Stage),
	}

	s.mu.Lock()
	s.apis[id] = api
	s.mu.Unlock()

	return api
}

// GetRestApi returns the API with the given id, or an AWSError if not found.
func (s *Store) GetRestApi(id string) (*RestApi, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[id]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", id), http.StatusNotFound)
	}
	return api, nil
}

// ListRestApis returns all REST APIs as a snapshot.
func (s *Store) ListRestApis() []*RestApi {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RestApi, 0, len(s.apis))
	for _, api := range s.apis {
		out = append(out, api)
	}
	return out
}

// DeleteRestApi removes an API and all its sub-resources.
func (s *Store) DeleteRestApi(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apis[id]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", id), http.StatusNotFound)
	}
	delete(s.apis, id)
	return nil
}

// ---- Resource operations ----

// computePath walks up the parent chain to build the full path for a new resource.
// The caller must hold at least a read lock on s.mu, or the API's Resources map.
func computePath(resources map[string]*Resource, parentID, pathPart string) string {
	if parentID == "" {
		return "/" + pathPart
	}
	parent, ok := resources[parentID]
	if !ok {
		return "/" + pathPart
	}
	if parent.Path == "/" {
		return "/" + pathPart
	}
	return parent.Path + "/" + pathPart
}

// CreateResource adds a new resource to the given API.
func (s *Store) CreateResource(apiID, parentID, pathPart string) (*Resource, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	if _, ok := api.Resources[parentID]; !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid resource identifier specified %s", parentID), http.StatusNotFound)
	}

	id := newID(6)
	path := computePath(api.Resources, parentID, pathPart)
	r := &Resource{
		Id:       id,
		ParentId: parentID,
		PathPart: pathPart,
		Path:     path,
		Methods:  make(map[string]*Method),
	}
	api.Resources[id] = r
	return r, nil
}

// GetResources returns all resources for a given API.
func (s *Store) GetResources(apiID string) ([]*Resource, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	out := make([]*Resource, 0, len(api.Resources))
	for _, r := range api.Resources {
		out = append(out, r)
	}
	return out, nil
}

// DeleteResource removes a resource from an API.
func (s *Store) DeleteResource(apiID, resourceID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	if _, ok := api.Resources[resourceID]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid resource identifier specified %s", resourceID), http.StatusNotFound)
	}
	delete(api.Resources, resourceID)
	return nil
}

// ---- Method operations ----

// PutMethod creates or replaces a method on a resource.
func (s *Store) PutMethod(apiID, resourceID, httpMethod, authorizationType string) (*Method, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	res, ok := api.Resources[resourceID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid resource identifier specified %s", resourceID), http.StatusNotFound)
	}
	m := &Method{
		HttpMethod:        strings.ToUpper(httpMethod),
		AuthorizationType: authorizationType,
	}
	res.Methods[strings.ToUpper(httpMethod)] = m
	return m, nil
}

// GetMethod retrieves a method from a resource.
func (s *Store) GetMethod(apiID, resourceID, httpMethod string) (*Method, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	res, ok := api.Resources[resourceID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid resource identifier specified %s", resourceID), http.StatusNotFound)
	}
	m, ok := res.Methods[strings.ToUpper(httpMethod)]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid method %s specified for resource %s", httpMethod, resourceID), http.StatusNotFound)
	}
	return m, nil
}

// PutIntegration sets the integration on a method.
func (s *Store) PutIntegration(apiID, resourceID, httpMethod string, integration *Integration) (*Integration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	res, ok := api.Resources[resourceID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid resource identifier specified %s", resourceID), http.StatusNotFound)
	}
	m, ok := res.Methods[strings.ToUpper(httpMethod)]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid method %s specified for resource %s", httpMethod, resourceID), http.StatusNotFound)
	}
	m.Integration = integration
	return integration, nil
}

// ---- Deployment operations ----

// CreateDeployment adds a deployment to an API.
func (s *Store) CreateDeployment(apiID, description string) (*Deployment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	id := newID(10)
	d := &Deployment{
		Id:          id,
		CreatedDate: time.Now().UTC(),
		Description: description,
	}
	api.Deployments[id] = d
	return d, nil
}

// GetDeployments returns all deployments for an API.
func (s *Store) GetDeployments(apiID string) ([]*Deployment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	out := make([]*Deployment, 0, len(api.Deployments))
	for _, d := range api.Deployments {
		out = append(out, d)
	}
	return out, nil
}

// ---- Stage operations ----

// CreateStage creates a stage for an API deployment.
func (s *Store) CreateStage(apiID, stageName, deploymentID, description string) (*Stage, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	if _, ok := api.Deployments[deploymentID]; !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid deployment identifier specified %s", deploymentID), http.StatusNotFound)
	}
	st := &Stage{
		StageName:    stageName,
		DeploymentId: deploymentID,
		Description:  description,
		CreatedDate:  time.Now().UTC(),
	}
	api.Stages[stageName] = st
	return st, nil
}

// GetStages returns all stages for an API.
func (s *Store) GetStages(apiID string) ([]*Stage, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid API identifier specified %s", apiID), http.StatusNotFound)
	}
	out := make([]*Stage, 0, len(api.Stages))
	for _, st := range api.Stages {
		out = append(out, st)
	}
	return out, nil
}
