package eventgrid

// Topic is the ARM resource shape for Microsoft.EventGrid/topics.
type Topic struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// EventSubscription is the ARM child resource shape for Microsoft.EventGrid/topics/eventSubscriptions.
type EventSubscription struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

type topicSharedAccessKeys struct {
	Key1 string `json:"key1"`
	Key2 string `json:"key2"`
}
