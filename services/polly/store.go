package polly

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StoredLexicon is the persisted shape of a pronunciation lexicon.
type StoredLexicon struct {
	Name         string
	Content      string
	Alphabet     string
	LanguageCode string
	LexemesCount int
	Size         int
	LastModified time.Time
	Arn          string
}

// StoredTask is the persisted shape of a SpeechSynthesisTask.
type StoredTask struct {
	TaskID             string
	TaskStatus         string
	TaskStatusReason   string
	CreationTime       time.Time
	RequestCharacters  int
	Engine             string
	LanguageCode       string
	LexiconNames       []string
	OutputFormat       string
	OutputURI          string
	SampleRate         string
	SnsTopicArn        string
	SpeechMarkTypes    []string
	TextType           string
	VoiceID            string
}

// Store is the in-memory data store for polly resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string
	lexicons  map[string]*StoredLexicon
	tasks     map[string]*StoredTask
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID: accountID,
		region:    region,
		lexicons:  make(map[string]*StoredLexicon),
		tasks:     make(map[string]*StoredTask),
	}
}

// PutLexicon stores or replaces a lexicon.
func (s *Store) PutLexicon(name, content string) *StoredLexicon {
	s.mu.Lock()
	defer s.mu.Unlock()

	alphabet, language, lexemes := parseLexicon(content)
	lex := &StoredLexicon{
		Name:         name,
		Content:      content,
		Alphabet:     alphabet,
		LanguageCode: language,
		LexemesCount: lexemes,
		Size:         len(content),
		LastModified: time.Now().UTC(),
		Arn:          fmt.Sprintf("arn:aws:polly:%s:%s:lexicon/%s", s.region, s.accountID, name),
	}
	s.lexicons[name] = lex
	return lex
}

// GetLexicon returns a lexicon by name.
func (s *Store) GetLexicon(name string) (*StoredLexicon, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lex, ok := s.lexicons[name]
	if !ok {
		return nil, service.NewAWSError("LexiconNotFoundException",
			"Lexicon not found: "+name, 404)
	}
	return lex, nil
}

// DeleteLexicon removes a lexicon by name.
func (s *Store) DeleteLexicon(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lexicons[name]; !ok {
		return service.NewAWSError("LexiconNotFoundException",
			"Lexicon not found: "+name, 404)
	}
	delete(s.lexicons, name)
	return nil
}

// ListLexicons returns all stored lexicons.
func (s *Store) ListLexicons() []*StoredLexicon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredLexicon, 0, len(s.lexicons))
	for _, lex := range s.lexicons {
		out = append(out, lex)
	}
	return out
}

// CreateTask creates a new synthesis task.
func (s *Store) CreateTask(task *StoredTask) *StoredTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task.TaskID == "" {
		task.TaskID = newTaskID()
	}
	if task.TaskStatus == "" {
		task.TaskStatus = "completed"
	}
	if task.CreationTime.IsZero() {
		task.CreationTime = time.Now().UTC()
	}
	s.tasks[task.TaskID] = task
	return task
}

// GetTask returns a task by ID.
func (s *Store) GetTask(id string) (*StoredTask, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, service.NewAWSError("SynthesisTaskNotFoundException",
			"Synthesis task not found: "+id, 404)
	}
	return task, nil
}

// ListTasks returns all tasks, optionally filtered by status.
func (s *Store) ListTasks(status string) []*StoredTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		if status != "" && t.TaskStatus != status {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Reset clears all in-memory state. Satisfies the Resettable interface used by the admin API.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lexicons = make(map[string]*StoredLexicon)
	s.tasks = make(map[string]*StoredTask)
}

// parseLexicon does a best-effort scan of PLS content for alphabet, language
// and lexeme count without bringing in an XML parser.
func parseLexicon(content string) (alphabet, language string, lexemes int) {
	alphabet = "ipa"
	language = "en-US"
	for i := 0; i+8 < len(content); i++ {
		if content[i] == '<' && content[i+1] == 'l' && content[i+2] == 'e' &&
			content[i+3] == 'x' && content[i+4] == 'e' && content[i+5] == 'm' &&
			content[i+6] == 'e' {
			lexemes++
		}
	}
	return
}

func newTaskID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
