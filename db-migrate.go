package go_migrator

import (
	"fmt"
	"github.com/MashinIvan/go-migrator/internal/models"
	"github.com/MashinIvan/go-migrator/internal/repository"
	"gorm.io/gorm"
	"sort"
)

// Migrate сохраняет и выполняет миграции в нужном порядке. Для этого на первом шаге создаются системные таблицы version
// и migrations, затем определяется необходимость проведения миграции типа TypeBaseline, после чего выполняются миграции
// типов TypeVersioned. Миграции типа TypeRepeatable выполняются в последнюю очередь.
// Все зарегистрированные миграции сохраняются в таблицу migrations. Миграции считаются новыми по инедтификатору
// f(версия, тип миграции).
//
// Паникует при попытке сохранить миграцию с версией меньшей, чем уже сохраненные.
// Паникует в случае, если какая-либо из необходимых в рамках выполнения операции миграций не была найдена.
func (m *MigrationManager) Migrate() error {
	m.logger.Println("Preparing migrations execution")

	err := m.initSystemTables()
	if err != nil {
		return err
	}

	savedMigrations, err := m.saveNewMigrations()
	if err != nil {
		return err
	}

	plan := m.planMigrate(savedMigrations)

	for !plan.IsEmpty() {
		migrationModel := plan.PopFirst()

		migration, ok := m.findMigration(migrationModel)
		if !ok {
			if !m.allowBypassNotFound(migrationModel) {
				panic(fmt.Sprintf(
					"migration (type: %s, version: %s) not found\n",
					migrationModel.Type, migrationModel.Version,
				))
			}

			m.logger.Printf(
				"migration (type: %s, version: %s) not found, skipping",
				migrationModel.Type, migrationModel.Version,
			)
			err = repository.UpdateMigrationState(m.db, &migrationModel, models.StateNotFound)
			if err != nil {
				return err
			}

			continue
		}

		err = m.executeMigration(migrationModel, migration)
		if err != nil && !migration.allowFailure {
			err = repository.UpdateMigrationState(m.db, &migrationModel, models.StateFailure)
			if err != nil {
				return err
			}

			return err
		}

		err = m.saveStateOnSuccessfulMigration(savedMigrations, migrationModel, migration)
		if err != nil {
			return err
		}
	}

	m.logger.Println("Migrations completed, current repository version is up to date")
	return nil
}

func (m *MigrationManager) planMigrate(savedMigrations []models.MigrationModel) migrationsPlan {
	planner := migratePlanner{
		manager:         m,
		savedMigrations: savedMigrations,
	}
	return planner.MakePlan()
}

func (m *MigrationManager) initSystemTables() error {
	hasVersionTable := repository.HasVersionTable(m.db)
	hasMigrationsTable := repository.HasMigrationsTable(m.db)

	if !hasVersionTable {
		m.logger.Println("Table versions not found, creating")
		err := repository.CreateVersionTable(m.db)
		if err != nil {
			return err
		}
	}

	if !hasMigrationsTable {
		m.logger.Println("Table migrations not found, creating")
		err := repository.CreateMigrationsTable(m.db)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MigrationManager) saveNewMigrations() ([]models.MigrationModel, error) {
	savedMigrations, err := repository.GetMigrationsSorted(m.db, repository.OrderASC)
	if err != nil {
		return nil, err
	}

	maxRank := 0
	for i, _ := range savedMigrations {
		if rank := savedMigrations[i].Rank; rank > maxRank {
			maxRank = rank
		}
	}

	newMigrations := make([]*Migration, 0, len(m.registeredMigrations))
	for i, _ := range m.registeredMigrations {
		if migrationIsNew(m.registeredMigrations[i], savedMigrations) {
			newMigrations = append(newMigrations, m.registeredMigrations[i])
		}
	}

	// запрет на сохранение миграций с версией, которая ниже максимальной версии из уже загерисрированных миграций
	for i, _ := range newMigrations {
		versionIncorrect := false
		for j, _ := range savedMigrations {
			versionSaved := mustParseVersion(savedMigrations[j].Version)
			versionToSave := mustParseVersion(newMigrations[i].version)

			if versionSaved.MoreThan(versionToSave) {
				versionIncorrect = true
			}
		}
		if versionIncorrect {
			panic(fmt.Sprintf(
				"Attempting to register migration with lower version than existing one. Type: %s. Identifier: %d",
				newMigrations[i].migrationType, newMigrations[i].identifier,
			))
		}
	}

	sort.SliceStable(newMigrations, func(i, j int) bool {
		leftVersioned := m.registeredMigrations[i].migrator.Version()
		rightVersioned := m.registeredMigrations[j].migrator.Version()

		return leftVersioned.LessThan(rightVersioned)
	})

	err = m.db.Transaction(func(tx *gorm.DB) error {
		for i, _ := range newMigrations {
			migration, err := repository.SaveMigration(tx, repository.SaveMigrationRequest{
				Rank:        maxRank + (i + 1),
				Type:        string(newMigrations[i].migrationType),
				Version:     newMigrations[i].version,
				Description: newMigrations[i].migrator.Description(),
				State:       models.StateRegistered,
			})
			if err != nil {
				return err
			}

			savedMigrations = append(savedMigrations, migration)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedMigrations, nil
}

func (m *MigrationManager) executeMigration(migrationModel models.MigrationModel, migration *Migration) error {
	m.logger.Printf(
		"Executing %s migration: version %s. State: %s\n",
		migrationModel.Type, migrationModel.Version, migrationModel.State,
	)

	var err error
	if migration.transaction {
		err = m.db.Transaction(migration.migrator.Migrate)
	} else {
		err = migration.migrator.Migrate(m.db)
	}
	if err != nil {
		m.logger.Println("Error occurred on migrate:", err)
		return err
	}

	m.logger.Println("Migration Complete")
	return nil
}

func (m *MigrationManager) saveStateOnSuccessfulMigration(
	savedMigrations []models.MigrationModel,
	migrationModel models.MigrationModel,
	migration *Migration,
) error {
	switch migration.migrationType {
	case TypeVersioned:
		err := repository.SaveVersion(m.db, migration.version)
		if err != nil {
			return err
		}

	case TypeBaseline:
		err := repository.SaveVersion(m.db, migration.version)
		if err != nil {
			return err
		}

		// все миграции до текущей TypeBaseline помечаем как пропущенные
		for i, _ := range savedMigrations {
			if migrationModel.Id == savedMigrations[i].Id {
				break
			}

			err = repository.UpdateMigrationState(m.db, &savedMigrations[i], models.StateSkipped)
			if err != nil {
				return err
			}
		}
	}

	err := repository.UpdateMigrationStateExecuted(m.db, &migrationModel, models.StateSuccess, migration.checksum)
	if err != nil {
		return err
	}

	return nil
}

func (m *MigrationManager) allowBypassNotFound(migrationModel models.MigrationModel) bool {
	return migrationModel.Type == string(TypeRepeatable)
}
