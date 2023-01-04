package repository

import (
	"github.com/MashinIvan/go-migrator/internal/models"
	"gorm.io/gorm"
)

func GetVersion(db *gorm.DB) (string, error) {
	var row models.VersionModel
	err := db.First(&row).Error

	switch err {
	case gorm.ErrRecordNotFound:
		return row.Version, ErrNotFound
	default:
		return row.Version, err
	}
}

func SaveVersion(db *gorm.DB, version string) error {
	var row models.VersionModel
	count := db.First(&row).RowsAffected

	if count == 0 {
		db.Create(&models.VersionModel{Version: version})
		return nil
	}

	return db.Model(&models.VersionModel{}).Where("version = ?", row.Version).Update("version", version).Error
}

func HasVersionTable(db *gorm.DB) bool {
	return db.Migrator().HasTable(models.VersionModel{}.TableName())
}

func CreateVersionTable(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS version (
			version TEXT
		)
	`).Error
}
