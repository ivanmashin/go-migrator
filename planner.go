package go_migrator

import (
	"container/list"
	"github.com/MashinIvan/go-migrator/internal/models"
	"sort"
)

type migrationsPlan struct {
	migrationsToRun *list.List
}

func newMigrationsPlan() migrationsPlan {
	return migrationsPlan{
		migrationsToRun: list.New(),
	}
}

func (p migrationsPlan) IsEmpty() bool {
	return p.migrationsToRun.Len() == 0
}

func (p migrationsPlan) PopFirst() models.MigrationModel {
	first := p.migrationsToRun.Front()
	p.migrationsToRun.Remove(first)
	return first.Value.(models.MigrationModel)
}

type migratePlanner struct {
	manager         *MigrationManager
	savedMigrations []models.MigrationModel

	plannedBaseline   models.MigrationModel
	baselineIsPlanned bool
}

func (p *migratePlanner) MakePlan() migrationsPlan {
	plan := newMigrationsPlan()
	p.planMigrationsBaseline(&plan)
	p.planMigrationsVersioned(&plan)
	p.planMigrationsRepeatable(&plan)

	return plan
}

func (p *migratePlanner) planMigrationsBaseline(plan *migrationsPlan) {
	if !p.baselineRequired() {
		return
	}
	p.manager.logger.Println("No successful baseline migrations found, planning to execute latest available")

	relevantBaseline, ok := p.findRelevantBaseline()
	if !ok {
		p.manager.logger.Println("No relevant baseline migrations for current target version found")
		return
	}

	plan.migrationsToRun.PushFront(relevantBaseline)

	p.baselineIsPlanned = true
	p.plannedBaseline = relevantBaseline
}

func (p *migratePlanner) planMigrationsVersioned(plan *migrationsPlan) {
	sort.SliceStable(p.savedMigrations, func(i, j int) bool {
		leftVersioned := mustParseVersion(p.savedMigrations[i].Version)
		rightVersioned := mustParseVersion(p.savedMigrations[j].Version)

		return rightVersioned.MoreThan(leftVersioned)
	})

	for _, migrationModel := range p.savedMigrations {
		if migrationModel.Type != string(TypeVersioned) {
			continue
		}
		if migrationModel.State == models.StateSuccess {
			continue
		}
		if migrationModel.State == models.StateSkipped {
			continue
		}

		migrationVersion := mustParseVersion(migrationModel.Version)

		if migrationVersion.MoreThan(p.manager.targetVersion) {
			continue
		}
		if migrationVersion.LessOrEqual(p.manager.getSavedAppVersion()) {
			continue
		}

		if p.baselineIsPlanned {
			baselineVersion := mustParseVersion(p.plannedBaseline.Version)
			if baselineVersion.MoreThan(migrationVersion) {
				continue
			}
		}

		plan.migrationsToRun.PushBack(migrationModel)
	}
}

func (p *migratePlanner) planMigrationsRepeatable(plan *migrationsPlan) {
	sort.SliceStable(p.savedMigrations, func(i, j int) bool {
		leftVersioned := mustParseVersion(p.savedMigrations[i].Version)
		rightVersioned := mustParseVersion(p.savedMigrations[j].Version)

		return rightVersioned.MoreThan(leftVersioned)
	})

	for _, migrationModel := range p.savedMigrations {
		if migrationModel.Type != string(TypeRepeatable) {
			continue
		}

		migration, ok := p.manager.findMigration(migrationModel)
		if !ok {
			// добавляем в очередь, чтобы при выполнении проставить необходимые статусы
			plan.migrationsToRun.PushBack(migrationModel)
			continue
		}

		if !migration.repeatUnconditional && migrationModel.Checksum == migration.checksum {
			p.manager.logger.Printf(
				"migration (type: %s, version: %s, checksum: %s) checksum not changed, skipping\n",
				migrationModel.Type, migrationModel.Version, migrationModel.Checksum,
			)
			continue
		}

		plan.migrationsToRun.PushBack(migrationModel)
	}
}

func (p *migratePlanner) baselineRequired() bool {
	for _, migration := range p.savedMigrations {
		if migration.Type == string(TypeBaseline) && migration.State == models.StateSuccess {
			return false
		}
	}
	return true
}

func (p *migratePlanner) findRelevantBaseline() (models.MigrationModel, bool) {
	var latestBaselineMigration models.MigrationModel
	var latestBaselineMigrationFound bool

	for _, migrationModel := range p.savedMigrations {
		if migrationModel.Type != string(TypeBaseline) {
			continue
		}

		version := mustParseVersion(migrationModel.Version)
		if version.LessOrEqual(p.manager.targetVersion) {
			latestBaselineMigration = migrationModel
			latestBaselineMigrationFound = true
		}
	}

	return latestBaselineMigration, latestBaselineMigrationFound
}

type downgradePlanner struct {
	manager         *MigrationManager
	savedMigrations []models.MigrationModel
}

func (p *downgradePlanner) MakePlan() migrationsPlan {
	plan := newMigrationsPlan()

	sort.SliceStable(p.savedMigrations, func(i, j int) bool {
		leftVersioned := mustParseVersion(p.savedMigrations[i].Version)
		rightVersioned := mustParseVersion(p.savedMigrations[j].Version)

		return leftVersioned.MoreThan(rightVersioned)
	})

	for _, migrationModel := range p.savedMigrations {
		migrationVersion := mustParseVersion(migrationModel.Version)

		if migrationModel.Type != string(TypeVersioned) {
			continue
		}
		if migrationVersion.MoreThan(p.manager.getSavedAppVersion()) {
			continue
		}
		if migrationVersion.LessOrEqual(p.manager.targetVersion) {
			continue
		}
		if migrationModel.State == models.StateUndone {
			continue
		}

		plan.migrationsToRun.PushBack(migrationModel)
	}

	return plan
}
