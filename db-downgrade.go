package go_migrator

import (
	"fmt"
	"github.com/MashinIvan/go-migrator/internal/models"
	"github.com/MashinIvan/go-migrator/internal/repository"
	"sort"
)

// Downgrade осуществляет отмену успешно выполненных или пропущенных миграций в обратном порядке.
// Миграции типа TypeRepeatable и TypeBaseline не отменяются.
// Новые миграции при вызове Downgrade не сохраняются.
//
// Паникует в случае, если какая-либо из миграций не была найдена.
func (m *MigrationManager) Downgrade() (err error) {
	m.logger.Println("Preparing downgrade execution")

	if !repository.HasVersionTable(m.db) || !repository.HasVersionTable(m.db) {
		panic("No migration table or version table found. Cannot perform downgrade")
	}

	savedMigrations, err := repository.GetMigrationsSorted(m.db, repository.OrderDESC)
	if err != nil {
		return err
	}

	plan, err := m.planDowngrade()
	if err != nil {
		return err
	}

	for !plan.IsEmpty() {
		migrationModel := plan.PopFirst()

		migration, ok := m.findMigration(migrationModel)
		if !ok {
			panic(fmt.Sprintf(
				"migration (type: %s, version: %s) not found\n",
				migrationModel.Type, migrationModel.Version,
			))
		}

		err = m.executeDowngrade(migrationModel, migration)
		if err != nil {
			return err
		}

		err = m.saveStateAfterDowngrading(savedMigrations, migrationModel, migration)
		if err != nil {
			return err
		}
	}

	m.logger.Println("Downgrade completed")
	return
}

func (m *MigrationManager) planDowngrade() (migrationsPlan, error) {
	savedMigrations, err := m.saveNewMigrations()
	if err != nil {
		return migrationsPlan{}, err
	}

	planner := downgradePlanner{
		manager:         m,
		savedMigrations: savedMigrations,
	}

	return planner.MakePlan(), nil
}

func (m *MigrationManager) executeDowngrade(migrationModel models.MigrationModel, migration *Migration) error {
	m.logger.Printf(
		"Downgrading %s migration: version %s. State: %s\n",
		migrationModel.Type, migrationModel.Version, migrationModel.State,
	)

	versionedMigrator, ok := migration.migrator.(VersionedMigrator)
	if !ok {
		panic("versioned migration must satisfy VersionedMigrator interface")
	}

	var err error
	if migration.transaction {
		err = m.db.Transaction(versionedMigrator.Downgrade)
	} else {
		err = versionedMigrator.Downgrade(m.db)
	}
	if err != nil {
		m.logger.Println("Error occurred on migrate:", err)
		return err
	}

	m.logger.Println("Downgrade complete")
	return nil
}

func (m *MigrationManager) saveStateAfterDowngrading(savedMigrations []models.MigrationModel, migrationModel models.MigrationModel, migration *Migration) error {
	err := repository.UpdateMigrationStateExecuted(m.db, &migrationModel, models.StateUndone, migration.checksum)
	if err != nil {
		return err
	}

	return m.saveVersionDowngrade(migrationModel, savedMigrations)
}

func (m *MigrationManager) saveVersionDowngrade(
	migrationModel models.MigrationModel,
	savedMigrations []models.MigrationModel,
) error {
	// фильтруем миграции типа TypeRepeatable
	filteredMigrations := make([]models.MigrationModel, 0, len(savedMigrations))
	for i, _ := range savedMigrations {
		if savedMigrations[i].Type == string(TypeRepeatable) {
			continue
		}
		filteredMigrations = append(filteredMigrations, savedMigrations[i])
	}

	sort.SliceStable(filteredMigrations, func(i, j int) bool {
		leftVersioned := mustParseVersion(filteredMigrations[i].Version)
		rightVersioned := mustParseVersion(filteredMigrations[j].Version)

		return leftVersioned.LessThan(rightVersioned)
	})

	undoneMigrationVersion := mustParseVersion(migrationModel.Version)
	versionToSave := Version{Major: 0, Minor: 0, Patch: 0, PreRelease: 0}
	// находим предыдущую версию
	for i, _ := range filteredMigrations {
		if filteredMigrations[i].Type != string(TypeVersioned) {
			continue
		}

		migrationVersion := mustParseVersion(filteredMigrations[i].Version)
		if migrationVersion == undoneMigrationVersion {
			if i != 0 {
				versionToSave = mustParseVersion(filteredMigrations[i-1].Version)
			}
			break
		}
	}

	return repository.SaveVersion(m.db, versionToSave.String())
}
