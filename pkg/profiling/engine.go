package profiling

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"time"
)

// Engine manages capture, storage, and retrieval of runtime profiles.
type Engine struct {
	profileDir  string
	mu          sync.RWMutex
	profiles    []Profile
	maxProfiles int
}

// New creates a profiling engine that stores profiles in profileDir.
// maxProfiles controls the circular buffer size; oldest profiles are evicted when exceeded.
func New(profileDir string, maxProfiles int) *Engine {
	return &Engine{
		profileDir:  profileDir,
		maxProfiles: maxProfiles,
	}
}

// Capture takes a runtime profile of the given type for the named service.
// Supported types: "cpu", "heap", "goroutine".
func (e *Engine) Capture(service, profileType string, duration time.Duration) (*Profile, error) {
	id := fmt.Sprintf("%s-%s-%d", service, profileType, time.Now().UnixNano())
	filePath := filepath.Join(e.profileDir, id+".pprof")

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create profile file: %w", err)
	}

	switch profileType {
	case "cpu":
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			os.Remove(filePath)
			return nil, fmt.Errorf("start CPU profile: %w", err)
		}
		time.Sleep(duration)
		pprof.StopCPUProfile()
	case "heap":
		if err := pprof.WriteHeapProfile(f); err != nil {
			f.Close()
			os.Remove(filePath)
			return nil, fmt.Errorf("write heap profile: %w", err)
		}
	case "goroutine":
		p := pprof.Lookup("goroutine")
		if p == nil {
			f.Close()
			os.Remove(filePath)
			return nil, fmt.Errorf("goroutine profile not found")
		}
		if err := p.WriteTo(f, 0); err != nil {
			f.Close()
			os.Remove(filePath)
			return nil, fmt.Errorf("write goroutine profile: %w", err)
		}
	default:
		f.Close()
		os.Remove(filePath)
		return nil, fmt.Errorf("unsupported profile type: %s", profileType)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close profile file: %w", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat profile file: %w", err)
	}

	prof := Profile{
		ID:         id,
		Service:    service,
		Type:       profileType,
		FilePath:   filePath,
		CapturedAt: time.Now(),
		Duration:   duration,
		Size:       info.Size(),
	}

	e.mu.Lock()
	e.profiles = append(e.profiles, prof)
	if len(e.profiles) > e.maxProfiles {
		evicted := e.profiles[0]
		e.profiles = e.profiles[1:]
		os.Remove(evicted.FilePath)
	}
	e.mu.Unlock()

	return &prof, nil
}

// Get returns a profile by ID.
func (e *Engine) Get(id string) (*Profile, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for i := range e.profiles {
		if e.profiles[i].ID == id {
			p := e.profiles[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", id)
}

// List returns all profiles for a service. If service is empty, all profiles are returned.
func (e *Engine) List(service string) ([]Profile, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []Profile
	for _, p := range e.profiles {
		if service == "" || p.Service == service {
			result = append(result, p)
		}
	}
	return result, nil
}

// FilePath returns the file system path for a profile by ID.
func (e *Engine) FilePath(id string) (string, error) {
	p, err := e.Get(id)
	if err != nil {
		return "", err
	}
	return p.FilePath, nil
}

// FindRelevant returns profiles for a service captured within 5 minutes of the given time.
func (e *Engine) FindRelevant(service string, around time.Time) []Profile {
	const window = 5 * time.Minute

	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []Profile
	for _, p := range e.profiles {
		if p.Service != service {
			continue
		}
		diff := p.CapturedAt.Sub(around)
		if diff < 0 {
			diff = -diff
		}
		if diff <= window {
			result = append(result, p)
		}
	}
	return result
}
