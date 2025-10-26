package state

import (
	"errors"

	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
)

type Service struct {
	store *store.Postgres
}

func New(store *store.Postgres) *Service {
	store.RegisterModels(
		model.StackRecord{},
		model.UpdateRecord{},
		model.CheckpointRecord{},
		model.EngineEventRecord{},
		model.StackVersionRecord{},
		model.ServiceUser{},
	)

	return &Service{store}
}

// TODO - these should be used to abstract the db error types
var (
	ErrExist    = errors.New("resource already exists")
	ErrNotExist = errors.New("resource does not exist")
)
