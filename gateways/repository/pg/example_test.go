package pg

import (
	"context"
	"go-template/domain/entities"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestExampleRepository_CreateExample(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewExampleRepository(pool)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   entities.Example
		wantErr bool
	}{
		{
			name: "success",
			input: entities.Example{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Title: "Test Title",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.CreateExample(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify the record was created
				got, err := repo.GetExampleByID(ctx, id)
				assert.NoError(t, err)
				assert.Equal(t, tt.input.Title, got.Title)
			}
		})
	}
}
