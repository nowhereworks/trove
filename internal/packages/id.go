package packages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func NewUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return ""
	}
	return id.String()
}

func newUUID() (pgtype.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("generate uuid v7: %w", err)
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}
