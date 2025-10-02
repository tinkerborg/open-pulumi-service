package schema

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate/client"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

type StackRecord struct {
	ID        StackID `gorm:"primaryKey,type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Stack     *apitype.Stack       `gorm:"type:jsonb;serializer:json"`
	Updates   []UpdateRecord       `gorm:"foreignKey:StackID;constraint:OnDelete:CASCADE"`
	Versions  []StackVersionRecord `gorm:"foreignKey:StackID;constraint:OnDelete:CASCADE"`
}

type StackVersionRecord struct {
	StackID  StackID  `gorm:"primaryKey,type:text"`
	Version  int      `gorm:"primaryKey"`
	UpdateID UpdateID `gorm:"type:text"`
}

type StackID struct {
	client.StackIdentifier
}

func NewStackID(identifier client.StackIdentifier) StackID {
	return StackID{StackIdentifier: identifier}
}

func (s StackID) Value() (driver.Value, error) {
	imoo := s.StackIdentifier.String()
	return imoo, nil
	// return s.StackIdentifier.String(), nil
}

func (s *StackID) Scan(value interface{}) error {
	stackKey, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}

	parts := strings.Split(stackKey, "/")
	if len(parts) != 3 {
		return errors.New("invalid stack key")
	}

	stackName, err := tokens.ParseStackName(parts[2])
	if err != nil {
		return err
	}

	s.StackIdentifier = client.StackIdentifier{
		Owner:   parts[0],
		Project: parts[1],
		Stack:   stackName,
	}

	return nil
}
