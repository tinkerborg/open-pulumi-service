package model

import (
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

type StackRecord struct {
	ID        string               `gorm:"type:uuid;default:gen_random_uuid();unique"`
	Owner     string               `gorm:"primaryKey"`
	Project   string               `gorm:"primaryKey"`
	Name      string               `gorm:"primaryKey"`
	Stack     *apitype.Stack       `gorm:"type:jsonb;serializer:json"`
	Updates   []UpdateRecord       `gorm:"foreignKey:StackID;references:ID;constraint:OnDelete:CASCADE"`
	Versions  []StackVersionRecord `gorm:"foreignKey:StackID;references:ID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StackVersionRecord struct {
	ID       string `gorm:"type:uuid;default:gen_random_uuid()"`
	StackID  string
	Version  int
	UpdateID string `gorm:"type:text"`
}
