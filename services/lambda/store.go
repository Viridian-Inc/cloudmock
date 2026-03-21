package lambda

import (
	"fmt"
	"sync"
	"time"
)

// Function represents an AWS Lambda function.
type Function struct {
	FunctionName string
	FunctionArn  string
	Runtime      string
	Role         string
	Handler      string
	Description  string
	Timeout      int
	MemorySize   int
	CodeSha256   string
	CodeSize     int64
	Version      string
	LastModified string
	Environment  *Environment
	Code         []byte // raw zip bytes
}

// Environment holds Lambda environment variables.
type Environment struct {
	Variables map[string]string `json:"Variables,omitempty"`
}

// FunctionStore manages Lambda functions in memory.
type FunctionStore struct {
	mu        sync.RWMutex
	functions map[string]*Function // key is FunctionName
	accountID string
	region    string
}

// NewStore returns a new empty FunctionStore.
func NewStore(accountID, region string) *FunctionStore {
	return &FunctionStore{
		functions: make(map[string]*Function),
		accountID: accountID,
		region:    region,
	}
}

// Create stores a new Lambda function. Returns an error if the function already exists.
func (s *FunctionStore) Create(f *Function) (*Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.functions[f.FunctionName]; ok {
		return nil, fmt.Errorf("function already exists: %s", f.FunctionName)
	}

	f.FunctionArn = fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", s.region, s.accountID, f.FunctionName)
	f.Version = "$LATEST"
	f.LastModified = time.Now().UTC().Format(time.RFC3339Nano)

	s.functions[f.FunctionName] = f
	return f, nil
}

// Get retrieves a function by name.
func (s *FunctionStore) Get(name string) (*Function, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.functions[name]
	return f, ok
}

// List returns all functions.
func (s *FunctionStore) List() []*Function {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Function, 0, len(s.functions))
	for _, f := range s.functions {
		out = append(out, f)
	}
	return out
}

// Delete removes a function by name. Returns false if not found.
func (s *FunctionStore) Delete(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.functions[name]; !ok {
		return false
	}
	delete(s.functions, name)
	return true
}

// UpdateCode replaces the code for a function.
func (s *FunctionStore) UpdateCode(name string, code []byte, codeSha256 string, codeSize int64) (*Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.functions[name]
	if !ok {
		return nil, fmt.Errorf("function not found: %s", name)
	}

	f.Code = code
	f.CodeSha256 = codeSha256
	f.CodeSize = codeSize
	f.LastModified = time.Now().UTC().Format(time.RFC3339Nano)

	return f, nil
}

// UpdateConfiguration updates the non-code configuration of a function.
func (s *FunctionStore) UpdateConfiguration(name string, updates map[string]interface{}) (*Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.functions[name]
	if !ok {
		return nil, fmt.Errorf("function not found: %s", name)
	}

	if v, ok := updates["Runtime"].(string); ok && v != "" {
		f.Runtime = v
	}
	if v, ok := updates["Handler"].(string); ok && v != "" {
		f.Handler = v
	}
	if v, ok := updates["Description"].(string); ok && v != "" {
		f.Description = v
	}
	if v, ok := updates["Role"].(string); ok && v != "" {
		f.Role = v
	}
	if v, ok := updates["Timeout"].(float64); ok && v > 0 {
		f.Timeout = int(v)
	}
	if v, ok := updates["MemorySize"].(float64); ok && v > 0 {
		f.MemorySize = int(v)
	}
	if env, ok := updates["Environment"].(map[string]interface{}); ok {
		if vars, ok := env["Variables"].(map[string]interface{}); ok {
			envVars := make(map[string]string, len(vars))
			for k, v := range vars {
				if sv, ok := v.(string); ok {
					envVars[k] = sv
				}
			}
			f.Environment = &Environment{Variables: envVars}
		}
	}

	f.LastModified = time.Now().UTC().Format(time.RFC3339Nano)
	return f, nil
}
