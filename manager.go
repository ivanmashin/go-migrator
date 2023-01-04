package go_migrator

import (
	"errors"
	"fmt"
	"github.com/MashinIvan/go-migrator/internal/models"
	"github.com/MashinIvan/go-migrator/internal/repository"
	"gorm.io/gorm"
	"hash/fnv"
	"log"
	"os"
)

var (
	ErrHasForthcomingMigrations = errors.New("found not completed forthcoming migrations, consider migrating")
	ErrHasFailedMigrations      = errors.New("found failed migrations, consider fixing your db")
	ErrTargetVersionNotLatest   = errors.New("target version falls behind migrations, consider raising target version")
)

// NewMigrationsManager создает экземпляр управляющего миграциями (выступает в качестве фасада).
// targetVersion - версия, до которой необходимо выполнить миграцию или до которой необоходимо осуществить откат.
func NewMigrationsManager(db *gorm.DB, targetVersion string, opts ...ManagerOption) (*MigrationManager, error) {
	target, err := parseVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	manager := MigrationManager{
		db:                      db,
		logger:                  log.New(os.Stderr, "", log.LstdFlags),
		targetVersion:           target,
		registeredMigrations:    make([]*Migration, 0),
		registeredMigrationsSet: make(map[uint32]*Migration),
	}
	for _, opt := range opts {
		opt(&manager)
	}

	return &manager, nil
}

type MigrationManager struct {
	db     *gorm.DB
	logger *log.Logger

	targetVersion Version

	registeredMigrations    []*Migration
	registeredMigrationsSet map[uint32]*Migration
}

// RegisterMigration сохраняет миграции в память.
// По умолчанию миграции осуществляются внутри транзакции.
//
// Паникует при регистрации миграций с одинаковымм версией и типом.
func (m *MigrationManager) RegisterMigration(migration *Migration, opts ...MigrationOption) {
	for _, opt := range opts {
		opt(migration)
	}

	identifier := getMigrationIdentifier(migration.version, string(migration.migrationType))
	if _, ok := m.registeredMigrationsSet[identifier]; ok {
		panic(fmt.Sprintf(
			"Migration with same identifier twice. Type: %s. Identifier: %d",
			migration.migrationType, identifier,
		))
	}

	migration.identifier = identifier
	m.registeredMigrationsSet[identifier] = migration
	m.registeredMigrations = append(m.registeredMigrations, migration)
	return
}

// CheckFulfillment проверяет корректность установки всех миграций. Проверяется, что нет миграций со статусом
// models.StateFailure, затем проверяется, что все зарегистрированные миграции выше послденей сохраненной версии сохранены и
// выполнены успешно, затем проверяется, что target версия установлена выше или равной последней найденной миграции.
func (m *MigrationManager) CheckFulfillment() (reasonErr error, ok bool, err error) {
	hasForthcoming, err := m.HasForthcomingMigrations()
	if err != nil {
		return nil, false, err
	}
	if hasForthcoming {
		return ErrHasForthcomingMigrations, false, nil
	}

	hasFailedMigrations, err := m.HasFailedMigrations()
	if err != nil {
		return nil, false, err
	}
	if hasFailedMigrations {
		return ErrHasFailedMigrations, false, err
	}

	targetVersionNotLatest, err := m.TargetVersionNotLatest()
	if err != nil {
		return nil, false, err
	}
	if targetVersionNotLatest {
		return ErrTargetVersionNotLatest, false, nil
	}

	return nil, true, nil
}

// HasFailedMigrations определяет есть ли миграции, не выполненные из-за ошибки.
func (m *MigrationManager) HasFailedMigrations() (bool, error) {
	// не было выполнено ни одной, следовательно пока ошибок не было
	if !repository.HasVersionTable(m.db) || !repository.HasMigrationsTable(m.db) {
		return false, nil
	}

	savedMigrations, err := repository.GetMigrationsSorted(m.db, repository.OrderASC)
	if err != nil {
		return false, err
	}

	for i, _ := range savedMigrations {
		if savedMigrations[i].State == models.StateFailure {
			return true, nil
		}
	}
	return false, nil
}

// HasForthcomingMigrations проверяет, есть ли зарегистрированные или сохраненные невыполненные миграции, выше текущей
// сохраненной версии.
func (m *MigrationManager) HasForthcomingMigrations() (bool, error) {
	// не было выполнено ни одной
	if !repository.HasVersionTable(m.db) || !repository.HasMigrationsTable(m.db) {
		return true, nil
	}

	savedVersion := m.getSavedAppVersion()

	savedMigrations, err := repository.GetMigrationsSorted(m.db, repository.OrderASC)
	if err != nil {
		return false, err
	}

	for i, _ := range savedMigrations {
		migrationVersion := mustParseVersion(savedMigrations[i].Version)
		if migrationVersion.MoreOrEqual(savedVersion) && savedMigrations[i].State != models.StateSuccess {
			return true, nil
		}
	}

	for i, _ := range m.registeredMigrations {
		// достаточно проверить, что миграция еще не сохранена, т.к. создание новых миграций разрешено только для версий
		// выше текущей максимальной версии сохраненных миграций
		if migrationIsNew(m.registeredMigrations[i], savedMigrations) {
			return true, nil
		}
	}

	return false, nil
}

// TargetVersionNotLatest проверяет, является ли target версия выше или равной максимальной версии зарегистрированной
// или сохраненной миграции.
func (m *MigrationManager) TargetVersionNotLatest() (bool, error) {
	// не было выполнено ни одной, следовательно пока ошибок не было
	if !repository.HasVersionTable(m.db) || !repository.HasMigrationsTable(m.db) {
		return false, nil
	}

	savedMigrations, err := repository.GetMigrationsSorted(m.db, repository.OrderASC)
	if err != nil {
		return false, err
	}

	for i, _ := range savedMigrations {
		migrationVersion := mustParseVersion(savedMigrations[i].Version)
		if !m.targetVersion.MoreOrEqual(migrationVersion) {
			return true, nil
		}
	}

	for i, _ := range m.registeredMigrations {
		migrationVersion := mustParseVersion(m.registeredMigrations[i].version)
		if !m.targetVersion.MoreOrEqual(migrationVersion) {
			return true, nil
		}
	}

	return false, nil
}

func (m *MigrationManager) findMigration(migrationModel models.MigrationModel) (*Migration, bool) {
	migrationModelIdentifier := getMigrationIdentifier(migrationModel.Version, migrationModel.Type)

	for _, migration := range m.registeredMigrations {
		registeredMigrationIdentifier := getMigrationIdentifier(migration.version, string(migration.migrationType))
		if registeredMigrationIdentifier == migrationModelIdentifier {
			return migration, true
		}
	}

	return nil, false
}

func (m *MigrationManager) getSavedAppVersion() Version {
	savedAppVersion, err := repository.GetVersion(m.db)
	// если текущая версия миграции не найдена, возвращаем версию 0.0.0, как минимально возможную
	if err == repository.ErrNotFound {
		return Version{}
	}
	if err != nil {
		return Version{}
	}

	return mustParseVersion(savedAppVersion)
}

func migrationIsNew(migration *Migration, savedMigrations []models.MigrationModel) bool {
	for j, _ := range savedMigrations {
		savedMigrationIdentifier := getMigrationIdentifier(savedMigrations[j].Version, savedMigrations[j].Type)
		if migration.identifier == savedMigrationIdentifier {
			return false
		}
	}
	return true
}

func getMigrationIdentifier(version, migrationType string) uint32 {
	h := fnv.New32a()
	// fmv.sum64a always writes with no error
	_, _ = h.Write([]byte(version + migrationType))
	return h.Sum32()
}
