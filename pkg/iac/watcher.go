package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors an IaC project directory for file changes and triggers
// re-scans. It debounces rapid changes (e.g. editor save + format) into a
// single callback invocation.
type Watcher struct {
	dir         string
	env         string
	logger      *slog.Logger
	onChange    func(*IaCImportResult)
	debounce    time.Duration
	watcher     *fsnotify.Watcher
	stopCh      chan struct{}
	stoppedOnce sync.Once
}

// WatcherOption configures a Watcher.
type WatcherOption func(*Watcher)

// WithDebounce sets the debounce interval (default 500ms).
func WithDebounce(d time.Duration) WatcherOption {
	return func(w *Watcher) { w.debounce = d }
}

// NewWatcher creates a file watcher that re-scans the IaC directory on changes.
// The onChange callback receives the new IaCImportResult each time files change.
func NewWatcher(dir, env string, logger *slog.Logger, onChange func(*IaCImportResult), opts ...WatcherOption) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		dir:      dir,
		env:      env,
		logger:   logger,
		onChange: onChange,
		debounce: 500 * time.Millisecond,
		watcher:  fsw,
		stopCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}

	// Add all directories under dir (fsnotify is not recursive by default).
	if err := w.addDirs(); err != nil {
		fsw.Close()
		return nil, err
	}

	go w.loop()
	return w, nil
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	w.stoppedOnce.Do(func() {
		close(w.stopCh)
		w.watcher.Close()
	})
}

// addDirs walks the directory tree and adds each directory to the fsnotify watcher.
func (w *Watcher) addDirs() error {
	return filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		// Skip node_modules and hidden dirs
		name := info.Name()
		if !info.IsDir() {
			return nil
		}
		if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		return w.watcher.Add(path)
	})
}

// loop is the main event loop — debounces file events and triggers re-scans.
func (w *Watcher) loop() {
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-w.stopCh:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Only react to IaC-relevant files
			if !isIaCFile(event.Name) {
				continue
			}
			// Debounce: reset timer on each event
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			timerC = timer.C

		case <-timerC:
			timerC = nil
			w.rescan()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("iac watcher error", "error", err)
		}
	}
}

// rescan re-imports the IaC directory and calls the onChange callback.
func (w *Watcher) rescan() {
	w.logger.Info("iac files changed, re-scanning", "dir", w.dir)
	result, err := ImportPulumiDir(w.dir, w.env, w.logger)
	if err != nil {
		w.logger.Error("iac re-scan failed", "error", err)
		return
	}
	w.logger.Info("iac re-scan complete",
		"tables", len(result.Tables), "lambdas", len(result.Lambdas),
		"sqs_queues", len(result.SQSQueues), "sns_topics", len(result.SNSTopics),
		"s3_buckets", len(result.S3Buckets), "microservices", len(result.Microservices))
	w.onChange(result)
}

// isIaCFile returns true if the file is relevant to IaC scanning.
func isIaCFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".ts", ".tsx", ".js", ".tf", ".hcl", ".json", ".yaml", ".yml":
		return true
	}
	return false
}
