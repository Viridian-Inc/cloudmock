package lexmodels

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// LatestVersion is the alias used for the working draft of a Lex resource.
const LatestVersion = "$LATEST"

// StoredBot is the persisted shape of a Lex bot version.
type StoredBot struct {
	Name                          string
	Version                       string
	Description                   string
	Locale                        string
	VoiceID                       string
	ProcessBehavior               string
	Status                        string
	FailureReason                 string
	Checksum                      string
	IdleSessionTTLInSeconds       int
	NluIntentConfidenceThreshold  float64
	ChildDirected                 bool
	DetectSentiment               bool
	EnableModelImprovements       bool
	CreateVersion                 bool
	Intents                       []map[string]any
	ClarificationPrompt           map[string]any
	AbortStatement                map[string]any
	CreatedAt                     time.Time
	LastUpdatedAt                 time.Time
}

// StoredIntent is the persisted shape of a Lex intent version.
type StoredIntent struct {
	Name                  string
	Version               string
	Description           string
	Checksum              string
	ParentIntentSignature string
	Slots                 []map[string]any
	SampleUtterances      []string
	ConfirmationPrompt    map[string]any
	RejectionStatement    map[string]any
	FollowUpPrompt        map[string]any
	ConclusionStatement   map[string]any
	DialogCodeHook        map[string]any
	FulfillmentActivity   map[string]any
	InputContexts         []map[string]any
	OutputContexts        []map[string]any
	KendraConfiguration   map[string]any
	CreatedAt             time.Time
	LastUpdatedAt         time.Time
}

// StoredSlotType is the persisted shape of a Lex slot type version.
type StoredSlotType struct {
	Name                    string
	Version                 string
	Description             string
	Checksum                string
	ValueSelectionStrategy  string
	ParentSlotTypeSignature string
	EnumerationValues       []map[string]any
	SlotTypeConfigurations  []map[string]any
	CreatedAt               time.Time
	LastUpdatedAt           time.Time
}

// StoredBotAlias is the persisted shape of a Lex bot alias.
type StoredBotAlias struct {
	Name             string
	BotName          string
	BotVersion       string
	Description      string
	Checksum         string
	ConversationLogs map[string]any
	CreatedAt        time.Time
	LastUpdatedAt    time.Time
}

// StoredChannelAssoc is the persisted shape of a bot channel association.
type StoredChannelAssoc struct {
	Name             string
	BotName          string
	BotAlias         string
	Description      string
	Type             string
	Status           string
	FailureReason    string
	BotConfiguration map[string]string
	CreatedAt        time.Time
}

// StoredImport models an asynchronous resource import job.
type StoredImport struct {
	ImportID      string
	Name          string
	ResourceType  string
	MergeStrategy string
	Status        string
	FailureReason []string
	Tags          map[string]string
	CreatedAt     time.Time
}

// StoredMigration models an async v1 → v2 bot migration job.
type StoredMigration struct {
	MigrationID       string
	MigrationStrategy string
	Status            string
	V1BotName         string
	V1BotVersion      string
	V1BotLocale       string
	V2BotID           string
	V2BotRole         string
	StartedAt         time.Time
	Alerts            []map[string]any
}

// Store is the in-memory data store for Lex Models V1.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	// bots: name → version → bot
	bots map[string]map[string]*StoredBot
	// intents: name → version → intent
	intents map[string]map[string]*StoredIntent
	// slotTypes: name → version → slotType
	slotTypes map[string]map[string]*StoredSlotType
	// botAliases: botName → aliasName → alias
	botAliases map[string]map[string]*StoredBotAlias
	// channelAssocs: botName → botAlias → channelName → assoc
	channelAssocs map[string]map[string]map[string]*StoredChannelAssoc
	// utterances: botName → list of utterance strings
	utterances map[string][]string

	imports    map[string]*StoredImport
	migrations map[string]*StoredMigration

	// tags: arn → key → value
	tags map[string]map[string]string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:     accountID,
		region:        region,
		bots:          make(map[string]map[string]*StoredBot),
		intents:       make(map[string]map[string]*StoredIntent),
		slotTypes:     make(map[string]map[string]*StoredSlotType),
		botAliases:    make(map[string]map[string]*StoredBotAlias),
		channelAssocs: make(map[string]map[string]map[string]*StoredChannelAssoc),
		utterances:    make(map[string][]string),
		imports:       make(map[string]*StoredImport),
		migrations:    make(map[string]*StoredMigration),
		tags:          make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state. Satisfies the Resettable interface
// used by the admin API.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bots = make(map[string]map[string]*StoredBot)
	s.intents = make(map[string]map[string]*StoredIntent)
	s.slotTypes = make(map[string]map[string]*StoredSlotType)
	s.botAliases = make(map[string]map[string]*StoredBotAlias)
	s.channelAssocs = make(map[string]map[string]map[string]*StoredChannelAssoc)
	s.utterances = make(map[string][]string)
	s.imports = make(map[string]*StoredImport)
	s.migrations = make(map[string]*StoredMigration)
	s.tags = make(map[string]map[string]string)
}

// ── ARN helpers ──────────────────────────────────────────────────────────────

func (s *Store) botArn(name string) string {
	return fmt.Sprintf("arn:aws:lex:%s:%s:bot:%s", s.region, s.accountID, name)
}

func (s *Store) intentArn(name string) string {
	return fmt.Sprintf("arn:aws:lex:%s:%s:intent:%s", s.region, s.accountID, name)
}

func (s *Store) slotTypeArn(name string) string {
	return fmt.Sprintf("arn:aws:lex:%s:%s:slottype:%s", s.region, s.accountID, name)
}

func (s *Store) botAliasArn(botName, aliasName string) string {
	return fmt.Sprintf("arn:aws:lex:%s:%s:bot:%s:%s", s.region, s.accountID, botName, aliasName)
}

// ── Bots ─────────────────────────────────────────────────────────────────────

// PutBot creates or updates the $LATEST version of a bot. If createVersion is
// set on the bot, a numeric version snapshot is also produced.
func (s *Store) PutBot(b *StoredBot) (*StoredBot, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.Name == "" {
		return nil, service.ErrValidation("name is required.")
	}
	if b.Locale == "" {
		b.Locale = "en-US"
	}
	if b.IdleSessionTTLInSeconds == 0 {
		b.IdleSessionTTLInSeconds = 300
	}
	if s.bots[b.Name] == nil {
		s.bots[b.Name] = make(map[string]*StoredBot)
	}
	now := time.Now().UTC()
	existing := s.bots[b.Name][LatestVersion]
	if existing == nil {
		b.CreatedAt = now
	} else {
		b.CreatedAt = existing.CreatedAt
	}
	b.LastUpdatedAt = now
	b.Version = LatestVersion
	if b.Status == "" {
		b.Status = "READY"
	}
	b.Checksum = newChecksum()
	s.bots[b.Name][LatestVersion] = b
	return b, nil
}

// CreateBotVersion snapshots $LATEST into a new numeric version.
func (s *Store) CreateBotVersion(name string) (*StoredBot, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions, ok := s.bots[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Bot not found: "+name, 404)
	}
	src, ok := versions[LatestVersion]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "$LATEST version not found for bot: "+name, 404)
	}
	next := nextNumericVersion(versions)
	snap := *src
	snap.Version = next
	snap.CreatedAt = time.Now().UTC()
	snap.LastUpdatedAt = snap.CreatedAt
	snap.Checksum = newChecksum()
	s.bots[name][next] = &snap
	return &snap, nil
}

// GetBot returns a stored bot version. Pass "$LATEST" or a numeric string.
func (s *Store) GetBot(name, version string) (*StoredBot, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.bots[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Bot not found: "+name, 404)
	}
	if version == "" {
		version = LatestVersion
	}
	bot, ok := versions[version]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Bot %s version %s not found", name, version), 404)
	}
	return bot, nil
}

// DeleteBot removes a bot and every version. Also cleans up aliases and
// channel associations rooted on that bot.
func (s *Store) DeleteBot(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bots[name]; !ok {
		return service.NewAWSError("NotFoundException", "Bot not found: "+name, 404)
	}
	delete(s.bots, name)
	delete(s.botAliases, name)
	delete(s.channelAssocs, name)
	delete(s.utterances, name)
	return nil
}

// DeleteBotVersion removes a single numeric version. $LATEST cannot be deleted.
func (s *Store) DeleteBotVersion(name, version string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == "" || version == LatestVersion {
		return service.ErrValidation("Cannot delete $LATEST. Use DeleteBot instead.")
	}
	versions, ok := s.bots[name]
	if !ok {
		return service.NewAWSError("NotFoundException", "Bot not found: "+name, 404)
	}
	if _, ok := versions[version]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Bot %s version %s not found", name, version), 404)
	}
	delete(versions, version)
	return nil
}

// ListBots returns one summary per bot (the highest available version).
func (s *Store) ListBots(nameContains string) []*StoredBot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredBot, 0, len(s.bots))
	for name, versions := range s.bots {
		if nameContains != "" && !contains(name, nameContains) {
			continue
		}
		if b := pickHighestBot(versions); b != nil {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ListBotVersions returns every version of a single bot.
func (s *Store) ListBotVersions(name string) ([]*StoredBot, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.bots[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Bot not found: "+name, 404)
	}
	out := make([]*StoredBot, 0, len(versions))
	for _, v := range versions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

// ── Intents ──────────────────────────────────────────────────────────────────

// PutIntent creates or updates the $LATEST version of an intent.
func (s *Store) PutIntent(in *StoredIntent) (*StoredIntent, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Name == "" {
		return nil, service.ErrValidation("name is required.")
	}
	if s.intents[in.Name] == nil {
		s.intents[in.Name] = make(map[string]*StoredIntent)
	}
	now := time.Now().UTC()
	existing := s.intents[in.Name][LatestVersion]
	if existing == nil {
		in.CreatedAt = now
	} else {
		in.CreatedAt = existing.CreatedAt
	}
	in.LastUpdatedAt = now
	in.Version = LatestVersion
	in.Checksum = newChecksum()
	s.intents[in.Name][LatestVersion] = in
	return in, nil
}

// CreateIntentVersion snapshots $LATEST.
func (s *Store) CreateIntentVersion(name string) (*StoredIntent, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions, ok := s.intents[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Intent not found: "+name, 404)
	}
	src, ok := versions[LatestVersion]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			"$LATEST version not found for intent: "+name, 404)
	}
	next := nextNumericVersionIntents(versions)
	snap := *src
	snap.Version = next
	snap.CreatedAt = time.Now().UTC()
	snap.LastUpdatedAt = snap.CreatedAt
	snap.Checksum = newChecksum()
	s.intents[name][next] = &snap
	return &snap, nil
}

func (s *Store) GetIntent(name, version string) (*StoredIntent, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.intents[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Intent not found: "+name, 404)
	}
	if version == "" {
		version = LatestVersion
	}
	in, ok := versions[version]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Intent %s version %s not found", name, version), 404)
	}
	return in, nil
}

func (s *Store) DeleteIntent(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.intents[name]; !ok {
		return service.NewAWSError("NotFoundException", "Intent not found: "+name, 404)
	}
	delete(s.intents, name)
	return nil
}

func (s *Store) DeleteIntentVersion(name, version string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == "" || version == LatestVersion {
		return service.ErrValidation("Cannot delete $LATEST. Use DeleteIntent instead.")
	}
	versions, ok := s.intents[name]
	if !ok {
		return service.NewAWSError("NotFoundException", "Intent not found: "+name, 404)
	}
	if _, ok := versions[version]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Intent %s version %s not found", name, version), 404)
	}
	delete(versions, version)
	return nil
}

func (s *Store) ListIntents(nameContains string) []*StoredIntent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredIntent, 0, len(s.intents))
	for name, versions := range s.intents {
		if nameContains != "" && !contains(name, nameContains) {
			continue
		}
		if in := pickHighestIntent(versions); in != nil {
			out = append(out, in)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) ListIntentVersions(name string) ([]*StoredIntent, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.intents[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Intent not found: "+name, 404)
	}
	out := make([]*StoredIntent, 0, len(versions))
	for _, v := range versions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

// ── Slot types ───────────────────────────────────────────────────────────────

func (s *Store) PutSlotType(st *StoredSlotType) (*StoredSlotType, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st.Name == "" {
		return nil, service.ErrValidation("name is required.")
	}
	if s.slotTypes[st.Name] == nil {
		s.slotTypes[st.Name] = make(map[string]*StoredSlotType)
	}
	now := time.Now().UTC()
	existing := s.slotTypes[st.Name][LatestVersion]
	if existing == nil {
		st.CreatedAt = now
	} else {
		st.CreatedAt = existing.CreatedAt
	}
	st.LastUpdatedAt = now
	st.Version = LatestVersion
	st.Checksum = newChecksum()
	s.slotTypes[st.Name][LatestVersion] = st
	return st, nil
}

func (s *Store) CreateSlotTypeVersion(name string) (*StoredSlotType, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions, ok := s.slotTypes[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Slot type not found: "+name, 404)
	}
	src, ok := versions[LatestVersion]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			"$LATEST version not found for slot type: "+name, 404)
	}
	next := nextNumericVersionSlotTypes(versions)
	snap := *src
	snap.Version = next
	snap.CreatedAt = time.Now().UTC()
	snap.LastUpdatedAt = snap.CreatedAt
	snap.Checksum = newChecksum()
	s.slotTypes[name][next] = &snap
	return &snap, nil
}

func (s *Store) GetSlotType(name, version string) (*StoredSlotType, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.slotTypes[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Slot type not found: "+name, 404)
	}
	if version == "" {
		version = LatestVersion
	}
	st, ok := versions[version]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Slot type %s version %s not found", name, version), 404)
	}
	return st, nil
}

func (s *Store) DeleteSlotType(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.slotTypes[name]; !ok {
		return service.NewAWSError("NotFoundException", "Slot type not found: "+name, 404)
	}
	delete(s.slotTypes, name)
	return nil
}

func (s *Store) DeleteSlotTypeVersion(name, version string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == "" || version == LatestVersion {
		return service.ErrValidation("Cannot delete $LATEST. Use DeleteSlotType instead.")
	}
	versions, ok := s.slotTypes[name]
	if !ok {
		return service.NewAWSError("NotFoundException", "Slot type not found: "+name, 404)
	}
	if _, ok := versions[version]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Slot type %s version %s not found", name, version), 404)
	}
	delete(versions, version)
	return nil
}

func (s *Store) ListSlotTypes(nameContains string) []*StoredSlotType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredSlotType, 0, len(s.slotTypes))
	for name, versions := range s.slotTypes {
		if nameContains != "" && !contains(name, nameContains) {
			continue
		}
		if st := pickHighestSlotType(versions); st != nil {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) ListSlotTypeVersions(name string) ([]*StoredSlotType, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.slotTypes[name]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Slot type not found: "+name, 404)
	}
	out := make([]*StoredSlotType, 0, len(versions))
	for _, v := range versions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

// ── Bot aliases ──────────────────────────────────────────────────────────────

func (s *Store) PutBotAlias(a *StoredBotAlias) (*StoredBotAlias, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.Name == "" {
		return nil, service.ErrValidation("name is required.")
	}
	if a.BotName == "" {
		return nil, service.ErrValidation("botName is required.")
	}
	if _, ok := s.bots[a.BotName]; !ok {
		return nil, service.NewAWSError("NotFoundException", "Bot not found: "+a.BotName, 404)
	}
	if s.botAliases[a.BotName] == nil {
		s.botAliases[a.BotName] = make(map[string]*StoredBotAlias)
	}
	now := time.Now().UTC()
	if existing, ok := s.botAliases[a.BotName][a.Name]; ok {
		a.CreatedAt = existing.CreatedAt
	} else {
		a.CreatedAt = now
	}
	a.LastUpdatedAt = now
	a.Checksum = newChecksum()
	s.botAliases[a.BotName][a.Name] = a
	return a, nil
}

func (s *Store) GetBotAlias(botName, aliasName string) (*StoredBotAlias, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	aliases, ok := s.botAliases[botName]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Alias %s not found for bot %s", aliasName, botName), 404)
	}
	a, ok := aliases[aliasName]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Alias %s not found for bot %s", aliasName, botName), 404)
	}
	return a, nil
}

func (s *Store) DeleteBotAlias(botName, aliasName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	aliases, ok := s.botAliases[botName]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Alias %s not found for bot %s", aliasName, botName), 404)
	}
	if _, ok := aliases[aliasName]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Alias %s not found for bot %s", aliasName, botName), 404)
	}
	delete(aliases, aliasName)
	if assocs, ok := s.channelAssocs[botName]; ok {
		delete(assocs, aliasName)
	}
	return nil
}

func (s *Store) ListBotAliases(botName, nameContains string) []*StoredBotAlias {
	s.mu.RLock()
	defer s.mu.RUnlock()
	aliases := s.botAliases[botName]
	out := make([]*StoredBotAlias, 0, len(aliases))
	for name, a := range aliases {
		if nameContains != "" && !contains(name, nameContains) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ── Channel associations ─────────────────────────────────────────────────────

func (s *Store) PutChannelAssoc(a *StoredChannelAssoc) (*StoredChannelAssoc, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.Name == "" || a.BotName == "" || a.BotAlias == "" {
		return nil, service.ErrValidation("name, botName, and botAlias are required.")
	}
	if s.channelAssocs[a.BotName] == nil {
		s.channelAssocs[a.BotName] = make(map[string]map[string]*StoredChannelAssoc)
	}
	if s.channelAssocs[a.BotName][a.BotAlias] == nil {
		s.channelAssocs[a.BotName][a.BotAlias] = make(map[string]*StoredChannelAssoc)
	}
	if a.Status == "" {
		a.Status = "CREATED"
	}
	a.CreatedAt = time.Now().UTC()
	s.channelAssocs[a.BotName][a.BotAlias][a.Name] = a
	return a, nil
}

func (s *Store) GetChannelAssoc(botName, alias, name string) (*StoredChannelAssoc, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if assocs, ok := s.channelAssocs[botName]; ok {
		if byAlias, ok := assocs[alias]; ok {
			if a, ok := byAlias[name]; ok {
				return a, nil
			}
		}
	}
	return nil, service.NewAWSError("NotFoundException",
		fmt.Sprintf("Channel association %s not found for %s/%s", name, botName, alias), 404)
}

func (s *Store) DeleteChannelAssoc(botName, alias, name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if assocs, ok := s.channelAssocs[botName]; ok {
		if byAlias, ok := assocs[alias]; ok {
			if _, ok := byAlias[name]; ok {
				delete(byAlias, name)
				return nil
			}
		}
	}
	return service.NewAWSError("NotFoundException",
		fmt.Sprintf("Channel association %s not found for %s/%s", name, botName, alias), 404)
}

func (s *Store) ListChannelAssocs(botName, alias, nameContains string) []*StoredChannelAssoc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*StoredChannelAssoc{}
	if assocs, ok := s.channelAssocs[botName]; ok {
		if byAlias, ok := assocs[alias]; ok {
			for name, a := range byAlias {
				if nameContains != "" && !contains(name, nameContains) {
					continue
				}
				out = append(out, a)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ── Utterances ───────────────────────────────────────────────────────────────

// AddUtterance appends a sample utterance for the given bot.
func (s *Store) AddUtterance(botName, utterance string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.utterances[botName] = append(s.utterances[botName], utterance)
}

// GetUtterances returns recorded utterances for a bot.
func (s *Store) GetUtterances(botName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.utterances[botName]))
	copy(out, s.utterances[botName])
	return out
}

// DeleteUtterances drops recorded utterances for a bot.
func (s *Store) DeleteUtterances(botName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.utterances, botName)
}

// ── Imports / migrations ─────────────────────────────────────────────────────

func (s *Store) StartImport(name, resourceType, mergeStrategy string, tags map[string]string) *StoredImport {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	imp := &StoredImport{
		ImportID:      id,
		Name:          name,
		ResourceType:  resourceType,
		MergeStrategy: mergeStrategy,
		Status:        "COMPLETE",
		Tags:          copyStringMap(tags),
		CreatedAt:     time.Now().UTC(),
	}
	s.imports[id] = imp
	return imp
}

func (s *Store) GetImport(id string) (*StoredImport, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	imp, ok := s.imports[id]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Import not found: "+id, 404)
	}
	return imp, nil
}

func (s *Store) StartMigration(strategy, v1Name, v1Version, v2Name, v2Role string) *StoredMigration {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	v2Id := newID()
	locale := "en-US"
	if versions, ok := s.bots[v1Name]; ok {
		if b, ok := versions[v1Version]; ok && b.Locale != "" {
			locale = b.Locale
		}
	}
	m := &StoredMigration{
		MigrationID:       id,
		MigrationStrategy: strategy,
		Status:            "COMPLETED",
		V1BotName:         v1Name,
		V1BotVersion:      v1Version,
		V1BotLocale:       locale,
		V2BotID:           v2Id,
		V2BotRole:         v2Role,
		StartedAt:         time.Now().UTC(),
		Alerts:            []map[string]any{},
	}
	s.migrations[id] = m
	return m
}

func (s *Store) GetMigration(id string) (*StoredMigration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.migrations[id]
	if !ok {
		return nil, service.NewAWSError("NotFoundException", "Migration not found: "+id, 404)
	}
	return m, nil
}

func (s *Store) ListMigrations() []*StoredMigration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMigration, 0, len(s.migrations))
	for _, m := range s.migrations {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.tags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newChecksum() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func contains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// nextNumericVersion returns the next "1", "2", … snapshot identifier that
// does not already exist in versions.
func nextNumericVersion(versions map[string]*StoredBot) string {
	max := 0
	for v := range versions {
		if v == LatestVersion {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

func nextNumericVersionIntents(versions map[string]*StoredIntent) string {
	max := 0
	for v := range versions {
		if v == LatestVersion {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

func nextNumericVersionSlotTypes(versions map[string]*StoredSlotType) string {
	max := 0
	for v := range versions {
		if v == LatestVersion {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

// pickHighestBot returns the highest numeric version, or $LATEST if no
// snapshots exist.
func pickHighestBot(versions map[string]*StoredBot) *StoredBot {
	var bestN int
	var best *StoredBot
	for v, b := range versions {
		if v == LatestVersion {
			if best == nil {
				best = b
			}
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && (best == nil || best.Version == LatestVersion || n > bestN) {
			best = b
			bestN = n
		}
	}
	return best
}

func pickHighestIntent(versions map[string]*StoredIntent) *StoredIntent {
	var bestN int
	var best *StoredIntent
	for v, b := range versions {
		if v == LatestVersion {
			if best == nil {
				best = b
			}
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && (best == nil || best.Version == LatestVersion || n > bestN) {
			best = b
			bestN = n
		}
	}
	return best
}

func pickHighestSlotType(versions map[string]*StoredSlotType) *StoredSlotType {
	var bestN int
	var best *StoredSlotType
	for v, b := range versions {
		if v == LatestVersion {
			if best == nil {
				best = b
			}
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && (best == nil || best.Version == LatestVersion || n > bestN) {
			best = b
			bestN = n
		}
	}
	return best
}

func versionLess(a, b string) bool {
	if a == LatestVersion {
		return true
	}
	if b == LatestVersion {
		return false
	}
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return an < bn
	}
	return a < b
}
