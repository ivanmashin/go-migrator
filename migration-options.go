package go_migrator

type MigrationOption func(*Migration)

// WithTransaction позволяет выполнить текущую миграцию внутри транзации. По умолчанию равен true. При выполнении
// группы миграций внутри транзакции, что конфигурируется в MigrationManager с помощью опции WithGroupTransaction,
// вложенная транзакция для данной миграции не создается.
func WithTransaction(useTransaction bool) MigrationOption {
	return func(m *Migration) {
		m.transaction = useTransaction
	}
}

type RepeatableMigratorOption func(*Migration)

// WithRepeatUnconditional позволяет игнорировать значение checksum для миграции типа TypeRepeatable и выполнять
// ее при каждом запуске Migrate.
func WithRepeatUnconditional() RepeatableMigratorOption {
	return func(m *Migration) {
		m.repeatUnconditional = true
	}
}
