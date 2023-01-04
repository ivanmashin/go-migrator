package models

import "time"

type MigrationState string

const (
	StateSuccess    MigrationState = "success"
	StateFailure    MigrationState = "failure"
	StateUndone     MigrationState = "undone"
	StateRegistered MigrationState = "registered"
	StateSkipped    MigrationState = "skipped"
	StateNotFound   MigrationState = "not found"
)

type MigrationModel struct {
	Id           uint32 `gorm:"primaryKey"`
	Rank         int
	Type         string
	Version      string
	Description  string
	RegisteredOn time.Time
	ExecutedOn   *time.Time
	Checksum     string
	State        MigrationState
}

func (v MigrationModel) TableName() string {
	return "migrations"
}
