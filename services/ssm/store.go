package ssm

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Parameter holds all data for a single SSM parameter.
type Parameter struct {
	Name             string
	Value            string
	Type             string // String | SecureString | StringList
	Version          int
	ARN              string
	Description      string
	LastModifiedDate time.Time
	DataType         string
	Tags             map[string]string
}

// Store is the in-memory store for SSM parameters.
type Store struct {
	mu        sync.RWMutex
	params    map[string]*Parameter // keyed by name
	accountID string
	region    string
}

// NewStore creates an empty SSM parameter Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		params:    make(map[string]*Parameter),
		accountID: accountID,
		region:    region,
	}
}

// buildARN constructs an ARN for a parameter.
// ARN format: arn:aws:ssm:{region}:{accountId}:parameter{name}
// name includes leading slash (e.g. /my/param → arn:...:parameter/my/param)
func (s *Store) buildARN(name string) string {
	// Ensure name starts with / for the ARN
	paramPath := name
	if !strings.HasPrefix(paramPath, "/") {
		paramPath = "/" + paramPath
	}
	return fmt.Sprintf("arn:aws:ssm:%s:%s:parameter%s", s.region, s.accountID, paramPath)
}

// PutParameter creates or updates a parameter. Returns the new version number.
func (s *Store) PutParameter(name, value, paramType, description string, overwrite bool, tags map[string]string) (int, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return 0, service.NewAWSError("ValidationError", "Name is required.", http.StatusBadRequest)
	}
	if paramType == "" {
		paramType = "String"
	}

	existing, exists := s.params[name]
	if exists && !overwrite {
		return 0, service.NewAWSError("ParameterAlreadyExists",
			fmt.Sprintf("The parameter already exists. To overwrite this value, set the overwrite option in the request to true. Parameter: %s", name),
			http.StatusBadRequest)
	}

	now := time.Now().UTC()
	version := 1
	arn := s.buildARN(name)
	existingTags := make(map[string]string)

	if exists {
		version = existing.Version + 1
		arn = existing.ARN
		// Preserve existing tags if none provided
		if len(tags) == 0 {
			existingTags = existing.Tags
		} else {
			existingTags = tags
		}
	} else {
		if len(tags) > 0 {
			existingTags = tags
		}
	}

	p := &Parameter{
		Name:             name,
		Value:            value,
		Type:             paramType,
		Version:          version,
		ARN:              arn,
		Description:      description,
		LastModifiedDate: now,
		DataType:         "text",
		Tags:             existingTags,
	}
	s.params[name] = p
	return version, nil
}

// GetParameter retrieves a single parameter by name.
func (s *Store) GetParameter(name string) (*Parameter, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.params[name]
	if !ok {
		return nil, service.NewAWSError("ParameterNotFound",
			fmt.Sprintf("Parameter %s not found.", name),
			http.StatusBadRequest)
	}
	return p, nil
}

// GetParameters retrieves multiple parameters by name.
// Returns (found parameters, invalid parameter names).
func (s *Store) GetParameters(names []string) ([]*Parameter, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found []*Parameter
	var invalid []string
	for _, name := range names {
		if p, ok := s.params[name]; ok {
			found = append(found, p)
		} else {
			invalid = append(invalid, name)
		}
	}
	return found, invalid
}

// GetParametersByPath returns parameters matching a path prefix.
// If recursive is false, only immediate children (no further slash after path prefix).
func (s *Store) GetParametersByPath(path string, recursive bool) []*Parameter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Normalize: path must end with / for prefix matching of children
	prefix := path
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	var results []*Parameter
	for name, p := range s.params {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		// The part after the prefix
		rest := name[len(prefix):]
		if !recursive {
			// Only immediate children: rest must not contain /
			if strings.Contains(rest, "/") {
				continue
			}
		}
		results = append(results, p)
	}
	return results
}

// DeleteParameter removes a single parameter by name.
func (s *Store) DeleteParameter(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.params[name]; !ok {
		return service.NewAWSError("ParameterNotFound",
			fmt.Sprintf("Parameter %s not found.", name),
			http.StatusBadRequest)
	}
	delete(s.params, name)
	return nil
}

// DeleteParameters removes multiple parameters by name.
// Returns (deleted names, invalid names that were not found).
func (s *Store) DeleteParameters(names []string) ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted []string
	var invalid []string
	for _, name := range names {
		if _, ok := s.params[name]; ok {
			delete(s.params, name)
			deleted = append(deleted, name)
		} else {
			invalid = append(invalid, name)
		}
	}
	return deleted, invalid
}

// DescribeParameters returns metadata for all parameters (without values).
func (s *Store) DescribeParameters() []*Parameter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Parameter, 0, len(s.params))
	for _, p := range s.params {
		out = append(out, p)
	}
	return out
}
