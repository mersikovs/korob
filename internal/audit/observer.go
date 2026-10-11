package audit

import (
	"context"

	models "github.com/mersikovs/korob.git/internal/model"
)

type Observer interface {
	Notify(ctx context.Context, event models.AuditEvent)
	GetID() string
}
