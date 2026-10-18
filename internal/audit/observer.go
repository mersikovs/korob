package audit

import (
	"context"

	models "github.com/mersikovs/korob.git/internal/model"
)

// Observer defines the contract for processing audit events
type Observer interface {
	Notify(ctx context.Context, event models.AuditEvent)
	GetID() string
	Close() error
}
