package models

type Settings struct {
	PasswordHash     string `json:"password_hash"`
	StorageDirectory string `json:"storage_directory"`
	BackupEnabled    bool   `json:"backup_enabled"`
	BackupInterval   int    `json:"backup_interval"`
	PasswordLength   int    `json:"password_length"`
	UseUpper         bool   `json:"use_upper"`
	UseNumbers       bool   `json:"use_numbers"`
	UseSpecial       bool   `json:"use_special"`
}
