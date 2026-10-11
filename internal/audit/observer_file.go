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

type FileObserver struct {
	ID       string
	filePath string
	file     *os.File
	mu       sync.Mutex
	logger   logger.Logger
}

func NewFileObserver(id, filePath string, logger logger.Logger) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file '%s': %w", filePath, err)
	}

	return &FileObserver{
		ID:       id,
		filePath: filePath,
		file:     file,
		logger:   logger,
	}, nil
}

func (f *FileObserver) Notify(ctx context.Context, event models.AuditEvent) {
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

func (f *FileObserver) GetID() string {
	return f.ID
}

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
