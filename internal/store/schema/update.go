package schema

import (
	"database/sql/driver"
	"errors"
	"time"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
)

// TODO - requestedBy should be a join
type UpdateRecord struct {
	ID          UpdateID `gorm:"primaryKey;type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Kind        apitype.UpdateKind `gorm:"type:jsonb;serializer:json"`
	Version     int
	Update      *apitype.UpdateProgram         `gorm:"type:jsonb;serializer:json"`
	Options     *apitype.UpdateOptions         `gorm:"type:jsonb;serializer:json"`
	Config      map[string]apitype.ConfigValue `gorm:"type:jsonb;serializer:json"`
	Metadata    *apitype.UpdateMetadata        `gorm:"type:jsonb;serializer:json"`
	Results     apitype.UpdateResults          `gorm:"type:jsonb;serializer:json"`
	RequestedBy model.ServiceUserInfo          `gorm:"type:jsonb;serializer:json"`
	StartTime   time.Time
	EndTime     time.Time
}

type CheckpointRecord struct {
	ID         UpdateID                     `gorm:"primaryKey;type:text"`
	Checkpoint *apitype.VersionedCheckpoint `gorm:"type:jsonb;serializer:json"`
}

type EngineEventRecord struct {
	ID          UpdateID             `gorm:"primaryKey;type:text"`
	Sequence    int                  `gorm:"primaryKey"`
	EngineEvent *apitype.EngineEvent `gorm:"type:jsonb;serializer:json"`
}

type UpdateID struct {
	client.UpdateIdentifier
}

func NewUpdateID(identifier client.UpdateIdentifier) UpdateID {
	return UpdateID{UpdateIdentifier: identifier}
}

func (s UpdateID) Value() (driver.Value, error) {
	updateID := s.UpdateIdentifier.UpdateID
	if updateID == "" {
		return nil, errors.New("missing UpdateID")
	}

	return updateID, nil
}

func (s *UpdateID) Scan(value interface{}) error {
	updateID, ok := value.(string)
	if !ok {
		return errors.New("updateID value is not a string")
	}

	s.UpdateIdentifier.UpdateID = updateID

	return nil
}
