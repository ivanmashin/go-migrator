package examples

import (
	"github.com/MashinIvan/go-migrator"
	"gorm.io/gorm"
)

func NewVersionedMigration() *go_migrator.Migration {
	return go_migrator.NewVersionedMigration(&versionedMigration{})
}

type versionedMigration struct{}

func (m *versionedMigration) Migrate(db *gorm.DB) error {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS groups
		(
			id            BIGSERIAL PRIMARY KEY,
			create_date   TIMESTAMP WITH TIME ZONE,
			update_date   TIMESTAMP WITH TIME ZONE,
			name          TEXT,
			owner_id      TEXT
				CONSTRAINT fk_group_owner
            		REFERENCES users
            		ON DELETE SET NULL,
		);
	`).Error
	return err
}

func (m *versionedMigration) Downgrade(db *gorm.DB) error {
	return db.Exec(`
		DROP TABLE IF EXISTS groups;
	`).Error
}

func (m *versionedMigration) Description() string {
	return "versioned migration"
}

func (m *versionedMigration) Version() go_migrator.Version {
	return go_migrator.Version{
		Major:      1,
		Minor:      1,
		Patch:      0,
		PreRelease: 0,
	}
}
