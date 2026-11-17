package models

import (
	"time"
)

type RefreshToken struct {
	ID        uint       `gorm:"column:id_token;primaryKey"`
	UserID    uint       `gorm:"column:id_user"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiredAt time.Time  `gorm:"column:expired_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	Base
}

func (RefreshToken) TableName() string { return "refresh_token" }
