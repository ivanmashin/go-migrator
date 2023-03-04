package db_migrator

import "database/sql"

type MigrationLite struct {
	migrationType MigrationType
	version       string
	description   string

	isTransactional bool
	isAllowFailure  bool

	up   string
	down string

	upF   func(db *sql.DB) error
	downF func(db *sql.DB) error

	checkSum            func() string
	identifier          uint32
	repeatUnconditional bool
}
