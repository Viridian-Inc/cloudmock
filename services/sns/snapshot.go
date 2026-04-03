package sns

import "encoding/json"

// snsState is the serialised form of all SNS state.
type snsState struct {
	Topics []snsTopicState `json:"topics"`
}

type snsTopicState struct {
	Name          string               `json:"name"`
	Attributes    map[string]string    `json:"attributes,omitempty"`
	Tags          map[string]string    `json:"tags,omitempty"`
	Subscriptions []snsSubscriptionState `json:"subscriptions,omitempty"`
}

type snsSubscriptionState struct {
	Protocol string `json:"protocol"`
	Endpoint string `json:"endpoint"`
}

// ExportState returns a JSON snapshot of all SNS topics and subscriptions.
func (s *SNSService) ExportState() (json.RawMessage, error) {
	state := snsState{Topics: make([]snsTopicState, 0)}

	s.store.mu.RLock()
	for _, topic := range s.store.topics {
		ts := snsTopicState{
			Name:       topic.Name,
			Attributes: topic.Attributes,
			Tags:       topic.Tags,
		}
		for _, sub := range topic.Subscriptions {
			ts.Subscriptions = append(ts.Subscriptions, snsSubscriptionState{
				Protocol: sub.Protocol,
				Endpoint: sub.Endpoint,
			})
		}
		state.Topics = append(state.Topics, ts)
	}
	s.store.mu.RUnlock()

	return json.Marshal(state)
}

// ImportState restores SNS state from a JSON snapshot.
func (s *SNSService) ImportState(data json.RawMessage) error {
	var state snsState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, ts := range state.Topics {
		topic := s.store.CreateTopic(ts.Name, ts.Attributes, ts.Tags)
		for _, sub := range ts.Subscriptions {
			s.store.Subscribe(topic.ARN, sub.Protocol, sub.Endpoint, s.store.accountID)
		}
	}
	return nil
}
