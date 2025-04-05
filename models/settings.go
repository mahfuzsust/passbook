package models

type Settings struct {
	PasswordHash     string `json:"password_hash"`
	StorageDirectory string `json:"storage_directory"`
	BackupEnabled    bool   `json:"backup_enabled"`
	BackupInterval   int    `json:"backup_interval"` // in minutes
}
