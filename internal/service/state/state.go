package state

import (
	"errors"

	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
)

type Service struct {
	store *store.Postgres
}

func New(store *store.Postgres) *Service {
	store.RegisterModels(
		&schema.StackRecord{},
		&schema.UpdateRecord{},
		&schema.CheckpointRecord{},
		&schema.EngineEventRecord{},
		&schema.StackVersionRecord{},
		&model.ServiceUser{},
	)

	return &Service{store}
}

// TODO - these should be used to abstract the db error types
var (
	ErrExist    = errors.New("resource already exists")
	ErrNotExist = errors.New("resource does not exist")
)
