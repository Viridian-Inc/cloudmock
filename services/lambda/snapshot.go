package lambda

import "encoding/json"

// lambdaState is the serialised form of all Lambda state.
type lambdaState struct {
	Functions []lambdaFunctionState `json:"functions"`
}

type lambdaFunctionState struct {
	FunctionName string            `json:"function_name"`
	Runtime      string            `json:"runtime"`
	Role         string            `json:"role"`
	Handler      string            `json:"handler"`
	Description  string            `json:"description,omitempty"`
	Timeout      int               `json:"timeout"`
	MemorySize   int               `json:"memory_size"`
	Environment  map[string]string `json:"environment,omitempty"`
}

// ExportState returns a JSON snapshot of all Lambda functions.
func (s *LambdaService) ExportState() (json.RawMessage, error) {
	state := lambdaState{Functions: make([]lambdaFunctionState, 0)}

	for _, fn := range s.store.List() {
		fs := lambdaFunctionState{
			FunctionName: fn.FunctionName,
			Runtime:      fn.Runtime,
			Role:         fn.Role,
			Handler:      fn.Handler,
			Description:  fn.Description,
			Timeout:      fn.Timeout,
			MemorySize:   fn.MemorySize,
		}
		if fn.Environment != nil {
			fs.Environment = fn.Environment.Variables
		}
		state.Functions = append(state.Functions, fs)
	}

	return json.Marshal(state)
}

// ImportState restores Lambda state from a JSON snapshot.
func (s *LambdaService) ImportState(data json.RawMessage) error {
	var state lambdaState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, fs := range state.Functions {
		fn := &Function{
			FunctionName: fs.FunctionName,
			Runtime:      fs.Runtime,
			Role:         fs.Role,
			Handler:      fs.Handler,
			Description:  fs.Description,
			Timeout:      fs.Timeout,
			MemorySize:   fs.MemorySize,
		}
		if fs.Environment != nil {
			fn.Environment = &Environment{Variables: fs.Environment}
		}
		// Ignore error if function already exists.
		s.store.Create(fn)
	}
	return nil
}
