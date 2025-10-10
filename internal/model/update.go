package model

import (
	"fmt"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

// TODO - requestedBy should be a join
type UpdateRecord struct {
	ID          string                         `gorm:"type:uuid;default:gen_random_uuid();unique"`
	StackID     string                         `gorm:"index;type:text"`
	Kind        apitype.UpdateKind             `gorm:"type:jsonb;serializer:json"`
	Version     int                            `gorm:"index"`
	Update      *apitype.UpdateProgram         `gorm:"type:jsonb;serializer:json"`
	Options     *apitype.UpdateOptions         `gorm:"type:jsonb;serializer:json"`
	Config      map[string]apitype.ConfigValue `gorm:"type:jsonb;serializer:json"`
	Metadata    *apitype.UpdateMetadata        `gorm:"type:jsonb;serializer:json"`
	Results     apitype.UpdateResults          `gorm:"type:jsonb;serializer:json"`
	RequestedBy ServiceUserInfo                `gorm:"type:jsonb;serializer:json"`
	Checkpoint  *CheckpointRecord              `gorm:"foreignKey:UpdateID;constraint:OnDelete:CASCADE"`
	Events      []EngineEventRecord            `gorm:"foreignKey:UpdateID;constraint:OnDelete:CASCADE"`
	DryRun      bool
	StartTime   time.Time
	EndTime     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CheckpointRecord struct {
	UpdateID   string                       `gorm:"primaryKey;type:text"`
	Checkpoint *apitype.VersionedCheckpoint `gorm:"type:jsonb;serializer:json"`
}

type EngineEventRecord struct {
	UpdateID    string               `gorm:"primaryKey;type:text"`
	Sequence    int                  `gorm:"primaryKey"`
	EngineEvent *apitype.EngineEvent `gorm:"type:jsonb;serializer:json"`
}

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
