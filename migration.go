package go_migrator

import (
	"gorm.io/gorm"
)

type MigrationType string

const (
	TypeBaseline   MigrationType = "baseline"
	TypeVersioned  MigrationType = "versioned"
	TypeRepeatable MigrationType = "repeatable"
)

type Migrator interface {
	Migrate(db *gorm.DB) error
	Description() string
	Version() Version
}

type VersionedMigrator interface {
	Migrator
	Downgrade(db *gorm.DB) error
}

type RepeatableMigrator interface {
	Migrator
	Checksum() string
}

func NewBaselineMigration(migrator Migrator) *Migration {
	return &Migration{
		transaction:   true,
		migrationType: TypeBaseline,
		migrator:      migrator,
		version:       migrator.Version().String(),
	}
}

func NewVersionedMigration(migrator VersionedMigrator) *Migration {
	return &Migration{
		transaction:   true,
		migrationType: TypeVersioned,
		migrator:      migrator,
		version:       migrator.Version().String(),
	}
}

func NewRepeatableMigration(migrator RepeatableMigrator, opts ...RepeatableMigratorOption) *Migration {
	migration := Migration{
		transaction:         true,
		repeatUnconditional: false,
		migrationType:       TypeRepeatable,
		migrator:            migrator,
		version:             migrator.Version().String(),
		checksum:            migrator.Checksum(),
	}

	for _, opt := range opts {
		opt(&migration)
	}
	return &migration
}

type Migration struct {
	// настраиваемые параметры миграции
	transaction         bool
	repeatUnconditional bool
	allowFailure        bool

	// свойства миграции
	identifier    uint32
	version       string
	checksum      string
	migrationType MigrationType
	migrator      Migrator
}
