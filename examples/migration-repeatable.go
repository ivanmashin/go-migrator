package examples

import (
	"encoding/base64"
	"github.com/MashinIvan/go-migrator"
	"gorm.io/gorm"
	"os"
)

func NewRepeatableMigration() *go_migrator.Migration {
	// initial struct here
	migrator := &repeatableMigration{}
	return go_migrator.NewRepeatableMigration(migrator)
}

type repeatableMigration struct {
	files []*os.File
}

func (m *repeatableMigration) Migrate(db *gorm.DB) error {
	// do something with m.files
	return nil
}

func (m *repeatableMigration) Description() string {
	return "repeatable migration based on variable checksum"
}

func (m *repeatableMigration) Version() go_migrator.Version {
	return go_migrator.Version{
		Major:      1,
		Minor:      1,
		Patch:      0,
		PreRelease: 0,
	}
}

func (m *repeatableMigration) Checksum() string {
	filesUpdates := make([]byte, 0)
	for i, _ := range m.files {
		stat, _ := m.files[i].Stat()
		modTimeBinary, _ := stat.ModTime().MarshalBinary()
		filesUpdates = append(filesUpdates, modTimeBinary...)
	}

	checksum := make([]byte, 0)
	base64.StdEncoding.Encode(checksum, filesUpdates)

	return string(checksum)
}
