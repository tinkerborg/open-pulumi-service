package state

import (
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
)

// TODO constrain update kind
// TODO - not use UpdateProgramRequest
// func (p *PulumiStateService) CreateUpdate(owner, project, name, kind string, update *apitype.UpdateProgram, options *apitype.UpdateOptions) (*string, error) {
func (p *Service) CreateUpdate(
	identifier client.UpdateIdentifier,
	update *apitype.UpdateProgram,
	options *apitype.UpdateOptions,
	config map[string]apitype.ConfigValue,
	metadata *apitype.UpdateMetadata,
) (*string, error) {
	stackRecord, err := readStackRecord(p.store, identifier.StackIdentifier)
	if err != nil {
		return nil, err
	}

	if update == nil {
		update = &apitype.UpdateProgram{}
	}

	if options == nil {
		options = &apitype.UpdateOptions{}
	}

	if config == nil {
		config = map[string]apitype.ConfigValue{}
	}

	if metadata == nil {
		metadata = &apitype.UpdateMetadata{}
	}

	updateID := uuid.New().String()
	identifier.UpdateID = updateID

	user, err := p.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	updateRecord := schema.UpdateRecord{
		ID:        schema.NewUpdateID(identifier),
		Kind:      identifier.UpdateKind,
		Update:    update,
		Version:   stackRecord.Stack.Version + 1,
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(0, 0),
		Options:   options,
		Config:    config,
		Metadata:  metadata,
		Results: apitype.UpdateResults{
			Status: apitype.StatusNotStarted,
			Events: []apitype.UpdateEvent{},
			// TODO
			// ContinuationToken:
		},
		RequestedBy: model.ServiceUserInfo{
			Name:        user.Name,
			GitHubLogin: user.GitHubLogin,
			AvatarURL:   user.AvatarURL,
		},
	}

	if err := p.store.Create(&updateRecord); err != nil {
		return nil, err
	}

	return &updateID, nil
}

func (p *Service) GetUpdateResults(identifier client.UpdateIdentifier) (*apitype.UpdateResults, error) {
	updateRecord, err := readUpdateRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	return &apitype.UpdateResults{
		Status: updateRecord.Results.Status,
		Events: []apitype.UpdateEvent{},
	}, nil
}

func (p *Service) StartUpdate(identifier client.UpdateIdentifier) (int, error) {
	var version int

	if err := p.store.Transaction(func(s *store.Postgres) error {
		updateRecord, err := readUpdateRecord(s, identifier)
		if err != nil {
			return err
		}

		updateRecord.StartTime = time.Now()
		updateRecord.Results.Status = apitype.StatusRunning

		if err := s.Update(updateRecord); err != nil {
			return err
		}

		version = updateRecord.Version

		return nil
	}); err != nil {
		return version, err
	}

	return version, nil
}

func (p *Service) CompleteUpdate(identifier client.UpdateIdentifier, status apitype.UpdateStatus) (*int, error) {
	var version int

	if err := p.store.Transaction(func(s *store.Postgres) error {
		// TODO - transaction
		updateRecord, err := readUpdateRecord(s, identifier)
		if err != nil {
			return err
		}

		updateRecord.Results.Status = status
		updateRecord.EndTime = time.Now()

		if err := s.Update(updateRecord); err != nil {
			return err
		}

		if updateRecord.Options.DryRun {
			// TODO
		} else {

			stackRecord, err := readStackRecord(s, identifier.StackIdentifier)
			if err != nil {
				return err
			}

			stackRecord.Stack.Version = updateRecord.Version
			stackRecord.Stack.ActiveUpdate = identifier.UpdateID

			versionRecord := &schema.StackVersionRecord{
				ID:       schema.NewStackID(identifier.StackIdentifier),
				Version:  updateRecord.Version,
				UpdateID: updateRecord.ID,
			}

			if err := s.Update(versionRecord); err != nil {
				return err
			}

			if err := s.Update(stackRecord); err != nil {
				return err
			}
		}

		version = updateRecord.Version

		return nil
	}); err != nil {
		return nil, err
	}

	return &version, nil
}

func (p *Service) CheckpointUpdate(identifier client.UpdateIdentifier, checkpoint *apitype.VersionedCheckpoint) error {
	checkpointRecord := schema.CheckpointRecord{
		ID:         schema.NewUpdateID(identifier),
		Checkpoint: checkpoint,
	}

	err := p.store.Update(&checkpointRecord)
	if err != nil {
		return err
	}

	return nil
}

func (p *Service) AddEngineEvents(identifier client.UpdateIdentifier, events []apitype.EngineEvent) error {
	updateID := schema.NewUpdateID(identifier)

	if err := p.store.Transaction(func(s *store.Postgres) error {
		for _, event := range events {
			eventRecord := schema.EngineEventRecord{
				ID:          updateID,
				Sequence:    event.Sequence,
				EngineEvent: &event,
			}

			if err := s.Create(&eventRecord); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (p *Service) ListEngineEvents(identifier client.UpdateIdentifier) ([]apitype.EngineEvent, error) {
	eventRecords := []schema.EngineEventRecord{}

	if err := p.store.List(&eventRecords, schema.NewUpdateID(identifier)); err != nil {
		return nil, err
	}

	// TODO - do this on db level
	events := []apitype.EngineEvent{}

	for _, eventRecord := range eventRecords {
		events = append(events, *eventRecord.EngineEvent)
	}

	return events, nil
}

func (p *Service) CreateImport(identifier client.UpdateIdentifier, deployment *apitype.UntypedDeployment) (string, error) {
	// TODO - fail update on errors
	updateID, err := p.CreateUpdate(identifier, nil, nil, nil, nil)
	if err != nil {
		return "", err
	}

	identifier.UpdateID = *updateID

	if _, err := p.StartUpdate(identifier); err != nil {
		return "", err
	}

	checkpoint := &apitype.VersionedCheckpoint{
		Version:    deployment.Version,
		Checkpoint: deployment.Deployment,
	}

	if err := p.CheckpointUpdate(identifier, checkpoint); err != nil {
		return "", err
	}

	if _, err := p.CompleteUpdate(identifier, apitype.StatusSucceeded); err != nil {
		return "", err
	}

	return *updateID, nil
}

func readUpdateRecord(s *store.Postgres, identifier client.UpdateIdentifier) (*schema.UpdateRecord, error) {
	updateRecord := schema.UpdateRecord{
		ID: schema.NewUpdateID(identifier),
	}

	err := s.Read(&updateRecord)
	if err != nil {
		return nil, err
	}

	return &updateRecord, nil
}
