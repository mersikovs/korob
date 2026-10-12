package repository

import (
	"context"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

// Storage defines the interface for metrics persistence and retrieval.
type Storage interface {
	Get(ctx context.Context, mType, name string) (models.Metrics, error)
	GetNamesList(ctx context.Context) ([]string, error)
	Save(ctx context.Context, m models.Metrics) error
	BatchSave(ctx context.Context, metrics []models.Metrics) error
}

// NewStorage returns a [Storage] implementation.
//
// It connects to PostgreSQL if [models.ServerConfig.DatabaseDSN] is provided,
// otherwise it falls back to an in-memory storage.
// Returns an error only on PostgreSQL connection failure.
func NewStorage(ctx context.Context, cnf *models.ServerConfig, logger logger.Logger) (Storage, error) {
	if cnf.DatabaseDSN != "" {
		return NewPgStorage(ctx, cnf.DatabaseDSN, logger)
	}

	return NewMemoryStorage(ctx, cnf, logger)
}
