package cloudwatchlogs

import "encoding/json"

// cwlState is the serialised form of all CloudWatch Logs state.
type cwlState struct {
	LogGroups []cwlLogGroupState `json:"log_groups"`
}

type cwlLogGroupState struct {
	Name          string            `json:"name"`
	RetentionDays int               `json:"retention_days,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// ExportState returns a JSON snapshot of all CloudWatch Logs log groups.
func (s *CloudWatchLogsService) ExportState() (json.RawMessage, error) {
	state := cwlState{LogGroups: make([]cwlLogGroupState, 0)}

	groups := s.store.DescribeLogGroups("")
	for _, g := range groups {
		state.LogGroups = append(state.LogGroups, cwlLogGroupState{
			Name:          g.Name,
			RetentionDays: g.RetentionDays,
			Tags:          g.Tags,
		})
	}

	return json.Marshal(state)
}

// ImportState restores CloudWatch Logs state from a JSON snapshot.
func (s *CloudWatchLogsService) ImportState(data json.RawMessage) error {
	var state cwlState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, gs := range state.LogGroups {
		// CreateLogGroup is idempotent-ish; ignore error if already exists.
		s.store.CreateLogGroup(gs.Name)
		if gs.RetentionDays > 0 {
			s.store.PutRetentionPolicy(gs.Name, gs.RetentionDays)
		}
		if len(gs.Tags) > 0 {
			s.store.TagLogGroup(gs.Name, gs.Tags)
		}
	}
	return nil
}
