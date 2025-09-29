package state

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
)

func (p *Service) CreateStack(stack *apitype.Stack) error {
	stack.ID = uuid.New().String()

	stackName, err := tokens.ParseStackName(stack.StackName.String())
	if err != nil {
		return err
	}

	record := schema.StackRecord{
		ID: schema.NewStackID(client.StackIdentifier{
			Owner:   stack.OrgName,
			Project: stack.ProjectName,
			Stack:   stackName,
		}),
		Stack: *stack,
	}

	if err := p.store.Create(&record); err != nil {
		return err
	}

	return nil
}

func (p *Service) GetStack(identifier client.StackIdentifier) (*apitype.Stack, error) {
	stackRecord, err := readStackRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	return &stackRecord.Stack, nil
}

func (p *Service) DeleteStack(identifier client.StackIdentifier) error {
	stackRecord := &schema.StackRecord{
		ID: schema.NewStackID(identifier),
	}

	return p.store.Delete(&stackRecord)
}

// TODO support latest
func (p *Service) GetStackUpdate(identifier client.StackIdentifier, version int) (*model.StackUpdate, error) {
	stackRecord, err := readStackRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	versionRecord := &schema.StackVersionRecord{
		ID:      schema.NewStackID(identifier),
		Version: version,
	}

	if err := p.store.Read(versionRecord); err != nil {
		return nil, err
	}

	updateRecord, err := readUpdateRecord(p.store, versionRecord.UpdateID.UpdateIdentifier)
	if err != nil {
		return nil, err
	}

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
	}, nil
}

func (p *Service) GetStackDeployment(identifier client.StackIdentifier) (*apitype.UntypedDeployment, error) {
	stackRecord, err := readStackRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	if stackRecord.Stack.Version == 0 {
		deployment, _ := json.Marshal(&apitype.DeploymentV1{
			Manifest: apitype.ManifestV1{
				Time:    time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC),
				Magic:   "",
				Version: "",
			},
		})

		return &apitype.UntypedDeployment{
			Version:    1,
			Deployment: deployment,
		}, nil
	}

	checkpointRecord := &schema.CheckpointRecord{
		ID: schema.NewUpdateID(client.UpdateIdentifier{UpdateID: stackRecord.Stack.ActiveUpdate}),
	}

	if err := p.store.Read(&checkpointRecord); err != nil {
		return nil, err
	}

	checkpoint := checkpointRecord.Checkpoint

	return &apitype.UntypedDeployment{
		Version:    checkpoint.Version,
		Features:   checkpoint.Features,
		Deployment: checkpoint.Checkpoint,
	}, nil

}

func readStackRecord(s *store.Postgres, identifier client.StackIdentifier) (*schema.StackRecord, error) {
	stackRecord := schema.StackRecord{
		ID: schema.NewStackID(identifier),
	}

	err := s.Read(&stackRecord)
	if err != nil {
		return nil, err
	}

	return &stackRecord, nil
}
