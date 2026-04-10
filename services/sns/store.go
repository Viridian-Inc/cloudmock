package sns

import (
	"fmt"
	"strconv"
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

// GetTopicAttributes returns the full set of attributes AWS reports for
// a topic: the explicit attributes stored on the topic plus synthesized
// defaults (Policy, Owner, EffectiveDeliveryPolicy, subscription
// counters). Callers should use this rather than reading t.Attributes
// directly, because SDK clients (notably Terraform's aws provider)
// unconditionally parse Policy as JSON and crash on an empty string.
func (s *Store) GetTopicAttributes(arn string) (map[string]string, bool) {
	s.mu.RLock()
	t, ok := s.topics[arn]
	if !ok {
		s.mu.RUnlock()
		return nil, false
	}
	// Count subscriptions by state for the SubscriptionsPending /
	// SubscriptionsConfirmed counters. We only track confirmation state,
	// so deleted is always reported as 0.
	var confirmed, pending int
	for _, sub := range t.Subscriptions {
		if sub.ConfirmationWasAuthenticated {
			confirmed++
		} else {
			pending++
		}
	}
	name := t.Name
	region := s.region
	accountID := s.accountID
	existing := make(map[string]string, len(t.Attributes))
	for k, v := range t.Attributes {
		existing[k] = v
	}
	s.mu.RUnlock()

	out := make(map[string]string, len(existing)+8)
	out["TopicArn"] = arn
	out["Owner"] = accountID
	out["SubscriptionsConfirmed"] = strconv.Itoa(confirmed)
	out["SubscriptionsPending"] = strconv.Itoa(pending)
	out["SubscriptionsDeleted"] = "0"
	out["Policy"] = defaultTopicPolicy(region, accountID, name)
	out["EffectiveDeliveryPolicy"] = defaultDeliveryPolicy()
	if _, has := existing["DisplayName"]; !has {
		out["DisplayName"] = name
	}
	// Explicit attributes (set via CreateTopic or SetTopicAttributes)
	// override the synthesized defaults.
	for k, v := range existing {
		out[k] = v
	}
	return out, true
}

// defaultTopicPolicy returns the default resource-based policy AWS
// attaches to a newly created SNS topic. It is a well-formed JSON
// document so Terraform's aws provider can parse it successfully.
func defaultTopicPolicy(region, accountID, topicName string) string {
	return fmt.Sprintf(`{"Version":"2008-10-17","Id":"__default_policy_ID","Statement":[{"Sid":"__default_statement_ID","Effect":"Allow","Principal":{"AWS":"*"},"Action":["SNS:GetTopicAttributes","SNS:SetTopicAttributes","SNS:AddPermission","SNS:RemovePermission","SNS:DeleteTopic","SNS:Subscribe","SNS:ListSubscriptionsByTopic","SNS:Publish"],"Resource":"arn:aws:sns:%s:%s:%s","Condition":{"StringEquals":{"AWS:SourceOwner":"%s"}}}]}`,
		region, accountID, topicName, accountID)
}

// defaultDeliveryPolicy returns the default SNS delivery policy JSON.
func defaultDeliveryPolicy() string {
	return `{"http":{"defaultHealthyRetryPolicy":{"minDelayTarget":20,"maxDelayTarget":20,"numRetries":3,"numMaxDelayRetries":0,"numNoDelayRetries":0,"numMinDelayRetries":0,"backoffFunction":"linear"},"disableSubscriptionOverrides":false}}`
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

// ListTagsForResource returns a copy of the tag map for a topic.
// Returns false if the topic does not exist.
func (s *Store) ListTagsForResource(topicArn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[topicArn]
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(t.Tags))
	for k, v := range t.Tags {
		out[k] = v
	}
	return out, true
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
