package db_migrator

import "io"

type ManagerOption func(*MigrationManager)

func WithLogWriter(w io.Writer) ManagerOption {
	return func(m *MigrationManager) {
		m.logger.SetOutput(w)
	}
}

func WithLogFlags(flags int) ManagerOption {
	return func(m *MigrationManager) {
		m.logger.SetFlags(flags)
	}
}
