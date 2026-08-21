package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func Test_isPreExecutionError(t *testing.T) {

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-PgError",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "wrapped non-PgError",
			err:  errors.New("wrap: " + errors.New("other").Error()),
			want: false,
		},
		// Положительные случаи для каждого кода из списка
		{
			name: "ConnectionException",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionException},
			want: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			err:  &pgconn.PgError{Code: pgerrcode.SQLClientUnableToEstablishSQLConnection},
			want: true,
		},
		{
			name: "ConnectionDoesNotExist",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist},
			want: true,
		},
		{
			name: "ConnectionFailure",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			want: true,
		},
		{
			name: "TransactionResolutionUnknown",
			err:  &pgconn.PgError{Code: pgerrcode.TransactionResolutionUnknown},
			want: true,
		},
		{
			name: "ProtocolViolation",
			err:  &pgconn.PgError{Code: pgerrcode.ProtocolViolation},
			want: true,
		},
		// Отрицательный случай с кодом, не входящим в список
		{
			name: "DuplicateKey (not pre-execution)",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation}, // 23505
			want: false,
		},
		{
			name: "another random PgError code",
			err:  &pgconn.PgError{Code: "12345"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPreExecutionError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
