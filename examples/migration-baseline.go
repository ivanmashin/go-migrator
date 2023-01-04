package examples

import (
	"github.com/MashinIvan/go-migrator"
	"gorm.io/gorm"
)

func NewInitialMigration() *go_migrator.Migration {
	return go_migrator.NewBaselineMigration(&initialMigration{})
}

type initialMigration struct{}

func (m *initialMigration) Migrate(db *gorm.DB) error {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users
		(
			id            BIGSERIAL PRIMARY KEY,
			create_date   TIMESTAMP WITH TIME ZONE,
			update_date   TIMESTAMP WITH TIME ZONE,
			username      TEXT,
			email         TEXT,
		);
	`).Error
	return err
}

func (m *initialMigration) Downgrade(db *gorm.DB) error {
	return db.Exec(`
		DROP TABLE IF EXISTS users;
	`).Error
}

func (m *initialMigration) Description() string {
	return "initial migration"
}

func (m *initialMigration) Version() go_migrator.Version {
	return go_migrator.Version{
		Major:      1,
		Minor:      0,
		Patch:      0,
		PreRelease: 0,
	}
}

func (m *initialMigration) Checksum() uint32 {
	return 0
}
