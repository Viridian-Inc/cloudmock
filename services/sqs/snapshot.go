package sqs

import "encoding/json"

// sqsState is the serialised form of all SQS state.
type sqsState struct {
	Queues []sqsQueueState `json:"queues"`
}

type sqsQueueState struct {
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// ExportState returns a JSON snapshot of all SQS queues.
func (s *SQSService) ExportState() (json.RawMessage, error) {
	state := sqsState{Queues: make([]sqsQueueState, 0)}

	s.store.mu.RLock()
	for _, q := range s.store.byName {
		state.Queues = append(state.Queues, sqsQueueState{
			Name:       q.QueueName(),
			Attributes: q.GetAttributes(),
		})
	}
	s.store.mu.RUnlock()

	return json.Marshal(state)
}

// ImportState restores SQS state from a JSON snapshot.
func (s *SQSService) ImportState(data json.RawMessage) error {
	var state sqsState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, qs := range state.Queues {
		// CreateQueue is idempotent.
		s.store.CreateQueue(qs.Name, qs.Attributes)
	}
	return nil
}
