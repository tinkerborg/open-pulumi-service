package state

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
	"github.com/tinkerborg/open-pulumi-service/internal/util"
)

// TODO - clean up handling of version strings (parse to int / latest)

func (p *Service) CreateStack(stack *apitype.Stack) error {
	stackName, err := tokens.ParseStackName(stack.StackName.String())
	if err != nil {
		return err
	}

	record := model.StackRecord{
		Owner:   stack.OrgName,
		Project: stack.ProjectName,
		Name:    stackName.String(),
		Stack:   stack,
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
	return p.store.Delete(StackRecord(identifier))
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

	updateRecord := &model.UpdateRecord{
		Version: versionNumber,
		StackID: stackRecord.ID,
		DryRun:  util.Ptr(false),
	}

	if err := p.store.Read(updateRecord, store.Join(model.ServiceUserInfo{})); err != nil {
		return nil, err
	}

	return createStackUpdate(stackRecord, updateRecord), nil
}

func (p *Service) GetStackDeployment(identifier client.StackIdentifier) (*apitype.UntypedDeployment, error) {
	// TODO nested preloads
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

	checkpointRecord := &model.CheckpointRecord{
		UpdateID: stackRecord.Stack.ActiveUpdate,
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

func (p *Service) ListStackResources(identifier client.UpdateIdentifier) ([]apitype.ResourceV3, error) {
	checkpointRecord := &model.CheckpointRecord{
		UpdateID: identifier.UpdateID,
	}

	if err := p.store.Read(checkpointRecord); err != nil {
		return nil, err
	}

	// TODO support versioning
	deployment := &apitype.DeploymentV3{}
	if err := json.Unmarshal(checkpointRecord.Checkpoint.Checkpoint, &deployment); err != nil {
		return nil, err
	}

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

func readStackRecord(s *store.Postgres, identifier client.StackIdentifier, opts ...store.DBOption) (*model.StackRecord, error) {
	stackRecord := StackRecord(identifier)

	err := s.Read(stackRecord, opts...)
	if err != nil {
		return nil, err
	}

	return stackRecord, nil
}

func StackRecord(identifier client.StackIdentifier) *model.StackRecord {
	return &model.StackRecord{
		Owner:   identifier.Owner,
		Project: identifier.Project,
		Name:    identifier.Stack.String(),
	}
}
