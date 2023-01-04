package examples

import (
	"github.com/MashinIvan/go-migrator"
	"gorm.io/gorm"
	"log"
)

func migrate() {
	db := &gorm.DB{}
	migrator, err := go_migrator.NewMigrationsManager(db, "1.1.0")
	if err != nil {
		log.Fatalln(err)
	}

	migrator.RegisterMigration(NewInitialMigration())
	migrator.RegisterMigration(NewRepeatableMigration())
	migrator.RegisterMigration(NewVersionedMigration())

	err = migrator.Migrate()
	if err != nil {
		log.Fatalln(err)
	}
}
