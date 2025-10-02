package model

import (
	"crypto/rsa"
)

type AuthToken struct {
	ID       int
	UserID   string   `gorm:"index"`
	Value    string   `gorm:"index"`
	Purposes []string `gorm:"type:jsonb"`
}

type RSAKey struct {
	ID    int             `gorm:"primaryKey"`
	Name  string          `gorm:"primaryKey"`
	Value *rsa.PrivateKey `gorm:"type:jsonb;serializer:json"`
}
