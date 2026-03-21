package sns

import (
	"fmt"
	"sync"
	"time"
)

// Topic represents an SNS topic.
type Topic struct {
	ARN           string
	Name          string
	Attributes    map[string]string
	Tags          map[string]string
	Subscriptions []*Subscription
}

// Subscription represents an SNS subscription.
type Subscription struct {
	ARN                          string
	Protocol                     string
	Endpoint                     string
	TopicArn                     string
	Owner                        string
	ConfirmationWasAuthenticated bool
}

// PublishedMessage is a record of a message that was published to a topic.
type PublishedMessage struct {
	MessageId         string
	TopicArn          string
	Message           string
	Subject           string
	Timestamp         time.Time
	MessageAttributes map[string]string
}

// Store manages all SNS topics, subscriptions, and the published-message log.
type Store struct {
	mu            sync.RWMutex
	topics        map[string]*Topic        // keyed by ARN
	subscriptions map[string]*Subscription // keyed by ARN
	messages      []*PublishedMessage
	accountID     string
	region        string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		topics:        make(map[string]*Topic),
		subscriptions: make(map[string]*Subscription),
		messages:      make([]*PublishedMessage, 0),
		accountID:     accountID,
		region:        region,
	}
}

// topicARN builds the ARN for a topic name.
func (s *Store) topicARN(name string) string {
	return fmt.Sprintf("arn:aws:sns:%s:%s:%s", s.region, s.accountID, name)
}

// subscriptionARN builds the ARN for a subscription.
func (s *Store) subscriptionARN(topicName, id string) string {
	return fmt.Sprintf("arn:aws:sns:%s:%s:%s:%s", s.region, s.accountID, topicName, id)
}

// CreateTopic creates a topic idempotently. Returns the existing topic if one already exists.
func (s *Store) CreateTopic(name string, attrs, tags map[string]string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := s.topicARN(name)
	if t, ok := s.topics[arn]; ok {
		return t
	}

	if attrs == nil {
		attrs = make(map[string]string)
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	t := &Topic{
		ARN:           arn,
		Name:          name,
		Attributes:    attrs,
		Tags:          tags,
		Subscriptions: make([]*Subscription, 0),
	}
	s.topics[arn] = t
	return t
}

// DeleteTopic removes a topic and all its subscriptions. Returns false if not found.
func (s *Store) DeleteTopic(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[arn]
	if !ok {
		return false
	}

	// Remove all subscriptions belonging to this topic.
	for _, sub := range t.Subscriptions {
		delete(s.subscriptions, sub.ARN)
	}

	delete(s.topics, arn)
	return true
}

// GetTopic returns a topic by ARN.
func (s *Store) GetTopic(arn string) (*Topic, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[arn]
	return t, ok
}

// ListTopics returns all topic ARNs.
func (s *Store) ListTopics() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	arns := make([]string, 0, len(s.topics))
	for arn := range s.topics {
		arns = append(arns, arn)
	}
	return arns
}

// Subscribe adds a subscription to a topic. Returns the new Subscription.
func (s *Store) Subscribe(topicArn, protocol, endpoint, owner string) (*Subscription, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[topicArn]
	if !ok {
		return nil, false
	}

	id := newUUID()
	subARN := s.subscriptionARN(t.Name, id)

	sub := &Subscription{
		ARN:      subARN,
		Protocol: protocol,
		Endpoint: endpoint,
		TopicArn: topicArn,
		Owner:    owner,
	}

	s.subscriptions[subARN] = sub
	t.Subscriptions = append(t.Subscriptions, sub)
	return sub, true
}

// Unsubscribe removes a subscription by ARN. Returns false if not found.
func (s *Store) Unsubscribe(subARN string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subARN]
	if !ok {
		return false
	}

	// Remove from the topic's subscription list.
	if t, topicOK := s.topics[sub.TopicArn]; topicOK {
		updated := make([]*Subscription, 0, len(t.Subscriptions))
		for _, ts := range t.Subscriptions {
			if ts.ARN != subARN {
				updated = append(updated, ts)
			}
		}
		t.Subscriptions = updated
	}

	delete(s.subscriptions, subARN)
	return true
}

// ListSubscriptions returns all subscriptions.
func (s *Store) ListSubscriptions() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs := make([]*Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// ListSubscriptionsByTopic returns subscriptions for a given topic ARN.
func (s *Store) ListSubscriptionsByTopic(topicArn string) ([]*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.topics[topicArn]
	if !ok {
		return nil, false
	}

	// Return a copy so callers don't hold the lock.
	result := make([]*Subscription, len(t.Subscriptions))
	copy(result, t.Subscriptions)
	return result, true
}

// Publish records a published message and returns its MessageId.
func (s *Store) Publish(topicArn, message, subject string, msgAttrs map[string]string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.topics[topicArn]; !ok {
		return "", false
	}

	if msgAttrs == nil {
		msgAttrs = make(map[string]string)
	}

	msgID := newUUID()
	pm := &PublishedMessage{
		MessageId:         msgID,
		TopicArn:          topicArn,
		Message:           message,
		Subject:           subject,
		Timestamp:         time.Now().UTC(),
		MessageAttributes: msgAttrs,
	}
	s.messages = append(s.messages, pm)
	return msgID, true
}

// TagResource sets tags on a topic.
func (s *Store) TagResource(topicArn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[topicArn]
	if !ok {
		return false
	}
	for k, v := range tags {
		t.Tags[k] = v
	}
	return true
}

// UntagResource removes tags from a topic.
func (s *Store) UntagResource(topicArn string, tagKeys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[topicArn]
	if !ok {
		return false
	}
	for _, k := range tagKeys {
		delete(t.Tags, k)
	}
	return true
}

// SetTopicAttribute sets a single attribute on a topic.
func (s *Store) SetTopicAttribute(topicArn, name, value string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[topicArn]
	if !ok {
		return false
	}
	t.Attributes[name] = value
	return true
}
