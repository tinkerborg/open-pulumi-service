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
	user *model.ServiceUser,
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

	updateRecord := schema.UpdateRecord{
		ID:        schema.NewUpdateID(identifier),
		StackID:   schema.NewStackID(identifier.StackIdentifier),
		Kind:      identifier.UpdateKind,
		Update:    update,
		Version:   stackRecord.Stack.Version + 1,
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(0, 0),
		Options:   options,
		Config:    config,
		Metadata:  metadata,
		DryRun:    options.DryRun,
		Results: apitype.UpdateResults{
			Status: apitype.StatusNotStarted,
			Events: []apitype.UpdateEvent{},
			// TODO
			// ContinuationToken:
		},
		// TODO should accept serviceuserinfo directly
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
				StackID:  schema.NewStackID(identifier.StackIdentifier),
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
		UpdateID:   schema.NewUpdateID(identifier),
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
				UpdateID:    updateID,
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
	// TODO - get user
	updateID, err := p.CreateUpdate(identifier, nil, nil, nil, nil, nil)
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

func (p *Service) ListPreviews(identifier client.StackIdentifier, version string) ([]*model.StackUpdate, error) {
	// version :=
	update, err := p.GetStackUpdate(identifier, version)
	if err != nil {
		return nil, err
	}

	stackRecord, err := readStackRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	condition := &schema.UpdateRecord{
		StackID: schema.NewStackID(identifier),
		Kind:    "preview",
		Version: update.Version + 1,
		DryRun:  true,
	}

	updateRecords := &[]*schema.UpdateRecord{}
	if err := p.store.List(updateRecords, condition); err != nil {
		return nil, err
	}

	updates := []*model.StackUpdate{}
	for _, updateRecord := range *updateRecords {
		switch updateRecord.Kind {
		case "destroy":
			updateRecord.Kind = "Pdestroy"
		default:
			updateRecord.Kind = "Pupdate"
		}
		updates = append(updates, createStackUpdate(stackRecord, updateRecord))

	}
	return updates, nil

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

func createStackUpdate(stackRecord *schema.StackRecord, updateRecord *schema.UpdateRecord) *model.StackUpdate {
	return &model.StackUpdate{
		Info: apitype.UpdateInfo{
			Kind:        updateRecord.Kind,
			Message:     "",
			Environment: updateRecord.Metadata.Environment,
			Config:      updateRecord.Config,
			StartTime:   updateRecord.StartTime.Unix(),
			EndTime:     updateRecord.EndTime.Unix(),
			Result:      model.ConvertUpdateStatus(updateRecord.Results.Status),
			Version:     updateRecord.Version,
		},
		RequestedBy: updateRecord.RequestedBy,
		GetDeploymentUpdatesUpdateInfo: apitype.GetDeploymentUpdatesUpdateInfo{
			UpdateID:      updateRecord.ID.UpdateID,
			Version:       updateRecord.Version,
			LatestVersion: stackRecord.Stack.Version,
		},
	}
}
