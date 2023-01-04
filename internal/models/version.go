package models

type VersionModel struct {
	Version string
}

func (v VersionModel) TableName() string {
	return "version"
}
