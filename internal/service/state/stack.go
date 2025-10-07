package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/store/schema"
)

// TODO - clean up handling of version strings (parse to int / latest)

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
		Stack: stack,
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

	return stackRecord.Stack, nil
}

func (p *Service) DeleteStack(identifier client.StackIdentifier) error {
	stackRecord := &schema.StackRecord{
		ID: schema.NewStackID(identifier),
	}

	return p.store.Delete(&stackRecord)
}

// TODO support latest
func (p *Service) GetStackUpdate(identifier client.StackIdentifier, version string) (*model.StackUpdate, error) {
	stackRecord, err := readStackRecord(p.store, identifier)
	if err != nil {
		return nil, err
	}

	versionNumber, err := p.ParseStackVersion(stackRecord.Stack, version)
	if err != nil {
		return nil, err
	}

	versionRecord := &schema.StackVersionRecord{
		StackID: schema.NewStackID(identifier),
		Version: versionNumber,
	}

	if err := p.store.Read(versionRecord); err != nil {
		return nil, err
	}

	updateRecord, err := readUpdateRecord(p.store, versionRecord.UpdateID.UpdateIdentifier)
	if err != nil {
		return nil, err
	}

	return createStackUpdate(stackRecord, updateRecord), nil
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
		UpdateID: schema.NewUpdateID(client.UpdateIdentifier{UpdateID: stackRecord.Stack.ActiveUpdate}),
	}

	if err := p.store.Read(checkpointRecord); err != nil {
		return nil, err
	}

	checkpoint := checkpointRecord.Checkpoint

	return &apitype.UntypedDeployment{
		Version:    checkpoint.Version,
		Features:   checkpoint.Features,
		Deployment: checkpoint.Checkpoint,
	}, nil

}

func (p *Service) ListStackResources(stackIdentifier client.StackIdentifier, version string) ([]apitype.ResourceV3, error) {
	update, err := p.GetStackUpdate(stackIdentifier, version)
	if err != nil {
		return nil, err
	}

	updateIdentifier := client.UpdateIdentifier{
		StackIdentifier: stackIdentifier,
		UpdateID:        update.UpdateID,
	}

	checkpointRecord := &schema.CheckpointRecord{
		UpdateID: schema.NewUpdateID(updateIdentifier),
	}

	if err := p.store.Read(checkpointRecord); err != nil {
		return nil, err
	}

	// TODO support versioning
	deployment := &apitype.DeploymentV3{}
	if err := json.Unmarshal(checkpointRecord.Checkpoint.Checkpoint, &deployment); err != nil {
		return nil, err
	}

	fmt.Printf("u %+v\n", checkpointRecord)

	return deployment.Resources, nil
}

func (p *Service) ParseStackVersion(stack *apitype.Stack, version string) (int, error) {
	var versionNumber int
	switch version {
	case "latest":
		versionNumber = stack.Version

	default:
		v, err := strconv.Atoi(version)
		if err != nil {
			return 0, err
		}
		versionNumber = v
	}

	return versionNumber, nil
}

func readStackRecord(s *store.Postgres, identifier client.StackIdentifier) (*schema.StackRecord, error) {
	stackRecord := &schema.StackRecord{
		ID: schema.NewStackID(identifier),
	}

	err := s.Read(stackRecord)
	if err != nil {
		return nil, err
	}

	return stackRecord, nil
}
