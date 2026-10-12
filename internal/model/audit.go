package models

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	// Ts is the Unix timestamp of the event.
	TS int64 `json:"ts"`
	// Metrics contains the names of the metrics affected or accessed.
	Metrics []string `json:"metrics"`
	// IPAddress is the client IP address that triggered the request.
	IPAddress string `json:"ip_address"`
}
