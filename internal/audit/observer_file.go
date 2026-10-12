package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

// FileObserver writes audit events to a local file with thread-safe append operations.
type FileObserver struct {
	id       string
	filePath string
	file     *os.File
	mu       sync.Mutex
	logger   logger.Logger
}

// NewFileObserver creates a FileObserver and opens the target file in append mode.
// Returns an error if the file cannot be created or opened.
func NewFileObserver(id, filePath string, logger logger.Logger) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file '%s': %w", filePath, err)
	}

	return &FileObserver{
		id:       id,
		filePath: filePath,
		file:     file,
		logger:   logger,
	}, nil
}

// Notify marshals the audit event to JSON and appends it to the file,
// ensuring thread-safe writes via mutex.
func (f *FileObserver) Notify(_ context.Context, event models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		if f.logger != nil {
			f.logger.Info("file observer: failed to marshal event", "error", err)
		}
		return
	}

	data = append(data, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	_, err = f.file.Write(data)
	if err != nil {
		if f.logger != nil {
			f.logger.Info("file observer: failed to write to file", "error", err, "path", f.filePath)
		}
	}
}

// GetID returns the unique identifier of this observer.
func (f *FileObserver) GetID() string {
	return f.id
}

// Close safely closes the underlying file and releases system resources.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		return err
	}
	return nil
}
