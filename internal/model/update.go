package model

import (
	"fmt"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

type UpdateRecord struct {
	ID          string                         `gorm:"index;type:uuid;default:gen_random_uuid()"`
	StackID     string                         `gorm:"type:uuid;uniqueIndex:idx_stack_id_version_dry_run,where:dry_run=false;index:idx_stack_id_version,where:dry_run=true"`
	Version     int                            `gorm:"uniqueIndex:idx_stack_id_version_dry_run,where:dry_run=false;index:idx_stack_id_version,where:dry_run=true"`
	DryRun      *bool                          `gorm:"uniqueIndex:idx_stack_id_version_dry_run,where:dry_run=false;index:idx_stack_id_version,where:dry_run=true"`
	Update      *apitype.UpdateProgram         `gorm:"type:jsonb;serializer:json"`
	Options     *apitype.UpdateOptions         `gorm:"type:jsonb;serializer:json"`
	Config      map[string]apitype.ConfigValue `gorm:"type:jsonb;serializer:json"`
	Metadata    *apitype.UpdateMetadata        `gorm:"type:jsonb;serializer:json"`
	Results     apitype.UpdateResults          `gorm:"type:jsonb;serializer:json"`
	UserID      string                         `gorm:"type:uuid;index"`
	RequestedBy *ServiceUserInfo               `gorm:"foreignKey:UserID"`
	Checkpoint  CheckpointRecord               `gorm:"foreignKey:UpdateID;constraint:OnDelete:CASCADE"`
	Events      []EngineEventRecord            `gorm:"foreignKey:UpdateID;constraint:OnDelete:CASCADE"`
	Kind        apitype.UpdateKind
	StartTime   time.Time
	EndTime     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CheckpointRecord struct {
	UpdateID   string                       `gorm:"primaryKey;type:text"`
	Checkpoint *apitype.VersionedCheckpoint `gorm:"type:jsonb;serializer:json"`
}

// type CheckpointRecord struct {
// 	ID         string          `gorm:"index;type:uuid;default:gen_random_uuid()"`
// 	Features   []string        `json:"features,omitempty" gorm:"type:jsonb;serializer:json"`
// 	Checkpoint json.RawMessage `json:"checkpoint" gorm:"type:jsonb;serializer:json"`
// 	Version    int
// }

type EngineEventRecord struct {
	UpdateID    string               `gorm:"primaryKey;type:text"`
	Sequence    int                  `gorm:"primaryKey"`
	EngineEvent *apitype.EngineEvent `gorm:"type:jsonb;serializer:json"`
}

type StackUpdate struct {
	Info             apitype.UpdateInfo `json:"info"`
	RequestedBy      *ServiceUserInfo   `json:"requestedBy"`
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
