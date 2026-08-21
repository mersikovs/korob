package repository

import (
	"context"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

type Storage interface {
	Get(ctx context.Context, mType, name string) (models.Metrics, error)
	GetNamesList(ctx context.Context) ([]string, error)
	Save(ctx context.Context, mType, name string, m models.Metrics) error
	BatchSave(ctx context.Context, metrics []models.Metrics) error
}

func NewStorage(ctx context.Context, cnf *models.ServerConfig, logger logger.Logger) (Storage, error) {

	if cnf.DatabaseDSN != "" {
		return NewPgStorage(ctx, cnf.DatabaseDSN, logger)
	}

	return NewMemoryStorage(ctx, cnf, logger)
}
