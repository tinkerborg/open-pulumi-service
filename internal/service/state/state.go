package state

import (
	"errors"

	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
)

type Service struct {
	store *store.Postgres
}

func New(store *store.Postgres) *Service {
	store.RegisterSchemas(
		schema.CheckpointRecord{},
		schema.EngineEventRecord{},
		schema.StackRecord{},
		schema.StackVersionRecord{},
		schema.UpdateRecord{},
	)

	return &Service{
		store: store,
	}
}

// TODO - these should be used to abstract the db error types
var (
	ErrExist    = errors.New("resource already exists")
	ErrNotExist = errors.New("resource does not exist")
)
