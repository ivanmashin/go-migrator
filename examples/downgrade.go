package examples

import (
	"github.com/MashinIvan/go-migrator"
	"gorm.io/gorm"
	"log"
)

func main() {
	db := &gorm.DB{}
	migrator, err := go_migrator.NewMigrationsManager(db, "1.0.0")
	if err != nil {
		log.Fatalln(err)
	}

	migrator.RegisterMigration(NewInitialMigration())
	migrator.RegisterMigration(NewRepeatableMigration())
	migrator.RegisterMigration(NewVersionedMigration())

	err = migrator.Downgrade()
	if err != nil {
		log.Fatalln(err)
	}
}
