package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mersikovs/korob.git/internal/config/db"
	models "github.com/mersikovs/korob.git/internal/model"
)

type PgStorage struct {
	pool *pgxpool.Pool
}

func NewPgStorage(ctx context.Context, dsn string) (*PgStorage, error) {
	curPool, err := db.NewPool(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &PgStorage{pool: curPool}, nil
}

func (s *PgStorage) GetNamesList(ctx context.Context) ([]string, error) {
	query := `SELECT id FROM metrics`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var id string

		if err := row.Scan(&id); err != nil {
			return "", fmt.Errorf("scan row: %w", err)
		}

		return id, nil
	})
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *PgStorage) Get(ctx context.Context, mType, name string) (models.Metrics, error) {
	query := `SELECT id, type, delta, value  FROM metrics WHERE id = $1 AND type  = $2`

	row := s.pool.QueryRow(ctx, query, name, mType)
	var m models.Metrics
	var delta pgtype.Int8
	var value pgtype.Float8

	err := row.Scan(&m.ID, &m.MType, &delta, &value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Metrics{}, nil
		}
		return models.Metrics{}, fmt.Errorf("scan metric: %w", err)
	}

	if delta.Valid {
		v := delta.Int64
		m.Delta = &v
	}
	if value.Valid {
		v := value.Float64
		m.Value = &v
	}

	return m, nil
}

func (s *PgStorage) Save(mType, name string, m models.Metrics) error {
	ctx := context.TODO()
	query := `
        INSERT INTO metrics (id, type, delta, value)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (id, type) DO UPDATE
        SET 
            delta = EXCLUDED.delta,
            value = EXCLUDED.value`

	_, err := s.pool.Exec(ctx, query, m.ID, m.MType, m.Delta, m.Value)
	if err != nil {
		return fmt.Errorf("save metric %q: %w", m.ID, err)
	}

	return nil
}

func (s *PgStorage) BatchSave(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO metrics (id, type, delta, value) VALUES ")

	args := make([]any, 0, len(metrics)*4)
	for i, m := range metrics {
		if i > 0 {
			sb.WriteString(", ")
		}

		base := i*4 + 1
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3)
		args = append(args, m.ID, m.MType, m.Delta, m.Value)
	}

	sb.WriteString(` ON CONFLICT (id, type) DO UPDATE SET
        value = CASE
            WHEN EXCLUDED.type = 'gauge'   THEN EXCLUDED.value
            ELSE metrics.value END,
		delta = CASE 
			WHEN EXCLUDED.type = 'counter' THEN metrics.delta + EXCLUDED.delta
			ELSE metrics.delta
        END`)

	delays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastError error

	for i := 0; i <= len(delays); i++ {
		ctx := context.TODO()

		_, lastError = s.pool.Exec(ctx, sb.String(), args...)

		if isPreExecutionError(lastError) && i < len(delays) {
			fmt.Printf("Pre-execution ошибка (попытка %d), retry через %v: %v\n",
				i+1, delays[i], lastError)
			time.Sleep(delays[i])
			continue
		}

		return lastError
	}
	return lastError
}

func (s *PgStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func isPreExecutionError(err error) bool {
	if err == nil {
		return false
	}

	var uerr *url.Error
	if errors.As(err, &uerr) {
		if uerr.Err == nil {
			return false
		}
		err = uerr.Err
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	msg := strings.ToLower(err.Error())

	if strings.Contains(msg, "connection refused") {
		return true
	}

	if strings.Contains(msg, "dial") && strings.Contains(msg, "timeout") {
		return true
	}

	return false
}

func (s *PgStorage) Close() error {
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}
