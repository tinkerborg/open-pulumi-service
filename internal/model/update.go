package model

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

type StackUpdate struct {
	Info             apitype.UpdateInfo `json:"info"`
	RequestedBy      ServiceUserInfo    `json:"requestedBy"`
	RequestedByToken string             `json:"requestedByToken"`

	apitype.GetDeploymentUpdatesUpdateInfo
}

type CompleteUpdateResponse struct {
	Version int `json:"version" yaml:"version"`
}

func ParseUpdateKind(kind string) (apitype.UpdateKind, error) {
	switch apitype.UpdateKind(kind) {
	case apitype.UpdateUpdate,
		apitype.PreviewUpdate,
		apitype.RefreshUpdate,
		apitype.RenameUpdate,
		apitype.DestroyUpdate,
		apitype.StackImportUpdate,
		apitype.ResourceImportUpdate:
		return apitype.UpdateKind(kind), nil
	}
	return apitype.UpdateKind(""), fmt.Errorf("invalid update kind '%s'", kind)
}

func ConvertUpdateStatus(status apitype.UpdateStatus) apitype.UpdateResult {
	switch status {
	case apitype.StatusNotStarted, apitype.StatusRequested:
		return apitype.NotStartedResult
	case apitype.StatusRunning:
		return apitype.InProgressResult
	case apitype.StatusFailed:
		return apitype.FailedResult
	case apitype.StatusSucceeded:
		return apitype.SucceededResult
	}

	return apitype.UpdateResult("")
}

type ListPreviewsResponse struct {
	Updates      []*StackUpdate `json:"updates"`
	ItemsPerPage int            `json:"itemsPerPage"`
	Total        int            `json:"total"`
}
