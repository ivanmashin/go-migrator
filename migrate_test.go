package db_migrator

import (
	"database/sql"
	"log"
	"testing"
)

const dsn = "postgres://admin:admin@127.0.0.1:5432/test"

func TestMigrate(t *testing.T) {
	{
		migrator, err := NewMigrationsManager(dsn, "1.1.0")
		if err != nil {
			log.Fatalln(err)
		}

		migrator.RegisterLite(
			MigrationLite{
				migrationType:   TypeBaseline,
				version:         "1.0.0.0",
				description:     "initial migration with connections",
				isAllowFailure:  false,
				isTransactional: true,
				up:              "",
				down:            "",
				upF: func(db *sql.DB) error {
					_, err := db.Exec("create table connections( id bigserial, one text, two numeric )")
					return err
				},
			},
			MigrationLite{
				migrationType: TypeVersioned,
				version:       "1.0.0.1",
				description:   "up connections",
				up:            "",
				down:          "",
				upF: func(db *sql.DB) error {
					_, err := db.Exec("drop table connections")
					return err
				},
			},
		)

		err = migrator.Migrate()
		if err != nil {
			log.Fatalln(err)
		}
	}
}
