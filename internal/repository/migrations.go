package repository

import (
	"github.com/Maksumys/db-migrator/internal/models"
	"gorm.io/gorm"
	"hash/fnv"
	"time"
)

type Order string

const (
	OrderASC  Order = "ASC"
	OrderDESC Order = "DESC"
)

func GetMigrationsSorted(db *gorm.DB, order Order) ([]models.MigrationModel, error) {
	var migrations []models.MigrationModel
	err := db.Order("rank " + string(order)).Find(&migrations).Error
	return migrations, err
}

func UpdateMigrationState(db *gorm.DB, model *models.MigrationModel, state models.MigrationState) error {
	return db.Model(model).Update("state", state).Error
}

func UpdateMigrationStateExecuted(db *gorm.DB, model *models.MigrationModel, state models.MigrationState, checksum string) error {
	now := time.Now().UTC()
	return db.Model(model).Updates(models.MigrationModel{
		ExecutedOn: &now,
		State:      state,
		Checksum:   checksum,
	}).Error
}

type SaveMigrationRequest struct {
	Rank        int
	Type        string
	Version     string
	Description string
	State       models.MigrationState
}

func SaveMigration(db *gorm.DB, request SaveMigrationRequest) (models.MigrationModel, error) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(request.Type + request.Version))
	migration := models.MigrationModel{
		Id:           h.Sum32(),
		Rank:         request.Rank,
		Type:         request.Type,
		Version:      request.Version,
		Description:  request.Description,
		RegisteredOn: time.Now().UTC(),
		State:        request.State,
	}

	return migration, db.Save(&migration).Error
}

func HasMigrationsTable(db *gorm.DB) bool {
	return db.Migrator().HasTable(models.MigrationModel{}.TableName())
}

func CreateMigrationsTable(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id NUMERIC PRIMARY KEY,
			rank BIGINT,
			type TEXT,
			version TEXT,
			description TEXT,
			registered_on TIMESTAMPTZ,
			executed_on TIMESTAMPTZ,
			checksum TEXT,
			state TEXT
		)
	`).Error
}
