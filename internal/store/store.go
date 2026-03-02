package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

type Store struct {
	db   *sql.DB
	path string
}

type FolderInfo struct {
	ID   int64
	Name string
}

type EntryMeta struct {
	ID        int64
	FolderID  int64
	Title     string
	EntryType string
}

type EntryFull struct {
	ID         int64
	FolderID   int64
	Type       string
	Title      string
	Username   string
	Password   string
	Link       string
	TotpSecret string
	CardNumber string
	Expiry     string
	CVV        string
	CustomText string
	FileName   string
	FileData   []byte
	History    []PasswordHistory
	Attachments []AttachmentMeta
}

type PasswordHistory struct {
	Password string
	Date     string
}

type AttachmentMeta struct {
	ID       string
	FileName string
	Size     int64
}

type PinConfig struct {
	Mode       string
	PinKey     []byte
	PinTag     string
	TotpSecret string
}

func Open(dbPath string, key string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma_key=%s", dbPath, url.QueryEscape(key))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	var count int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		db.Close()
		return nil, fmt.Errorf("wrong database key or corrupt database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting journal mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	s := &Store{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating schema: %w", err)
	}

	if err := os.Chmod(dbPath, 0600); err != nil && !os.IsNotExist(err) {
		// best-effort permission tightening
	}

	return s, nil
}

func DBExists(dbPath string) bool {
	_, err := os.Stat(dbPath)
	return err == nil
}

func VerifyKey(dbPath string, key string) error {
	s, err := Open(dbPath, key)
	if err != nil {
		return err
	}
	s.Close()
	return nil
}

func (s *Store) Rekey(newKey string) error {
	escaped := strings.ReplaceAll(newKey, "'", "''")
	_, err := s.db.Exec(fmt.Sprintf("PRAGMA rekey = '%s'", escaped))
	return err
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS folders (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS entries (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		folder_id   INTEGER NOT NULL DEFAULT 0,
		entry_type  TEXT NOT NULL DEFAULT '',
		title       TEXT NOT NULL,
		username    TEXT NOT NULL DEFAULT '',
		password    TEXT NOT NULL DEFAULT '',
		link        TEXT NOT NULL DEFAULT '',
		totp_secret TEXT NOT NULL DEFAULT '',
		card_number TEXT NOT NULL DEFAULT '',
		expiry      TEXT NOT NULL DEFAULT '',
		cvv         TEXT NOT NULL DEFAULT '',
		custom_text TEXT NOT NULL DEFAULT '',
		file_name   TEXT NOT NULL DEFAULT '',
		file_data   BLOB
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_folder_title
		ON entries(folder_id, title);

	CREATE TABLE IF NOT EXISTS password_history (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
		password TEXT NOT NULL,
		date     TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_password_history_entry
		ON password_history(entry_id);

	CREATE TABLE IF NOT EXISTS attachments (
		id        TEXT PRIMARY KEY,
		entry_id  INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
		file_name TEXT NOT NULL DEFAULT '',
		size      INTEGER NOT NULL DEFAULT 0,
		data      BLOB NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_attachments_entry
		ON attachments(entry_id);

	CREATE TABLE IF NOT EXISTS pin_config (
		id          INTEGER PRIMARY KEY CHECK (id = 1),
		mode        TEXT NOT NULL DEFAULT '',
		pin_key     BLOB,
		pin_tag     TEXT NOT NULL DEFAULT '',
		totp_secret TEXT NOT NULL DEFAULT ''
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// ── Folders ─────────────────────────────────────────────────────────

func (s *Store) ListFolders() ([]FolderInfo, error) {
	rows, err := s.db.Query("SELECT id, name FROM folders ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []FolderInfo
	for rows.Next() {
		var f FolderInfo
		if err := rows.Scan(&f.ID, &f.Name); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (s *Store) CreateFolder(name string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO folders (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) RenameFolder(id int64, name string) error {
	_, err := s.db.Exec("UPDATE folders SET name = ? WHERE id = ?", name, id)
	return err
}

func (s *Store) DeleteFolder(id int64) error {
	_, err := s.db.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}

func (s *Store) GetFolderByName(name string) (*FolderInfo, error) {
	var f FolderInfo
	err := s.db.QueryRow("SELECT id, name FROM folders WHERE name = ?", name).Scan(&f.ID, &f.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) GetFolder(id int64) (*FolderInfo, error) {
	var f FolderInfo
	err := s.db.QueryRow("SELECT id, name FROM folders WHERE id = ?", id).Scan(&f.ID, &f.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ── Entries ─────────────────────────────────────────────────────────

func (s *Store) SaveEntry(folderID int64, e *EntryFull) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO entries (folder_id, entry_type, title, username, password, link,
		 totp_secret, card_number, expiry, cvv, custom_text, file_name, file_data)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		folderID, e.Type, e.Title, e.Username, e.Password, e.Link,
		e.TotpSecret, e.CardNumber, e.Expiry, e.CVV, e.CustomText,
		e.FileName, e.FileData)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.replaceHistory(id, e.History); err != nil {
		return id, err
	}
	return id, nil
}

func (s *Store) UpdateEntryFull(id, folderID int64, e *EntryFull) error {
	_, err := s.db.Exec(
		`UPDATE entries SET folder_id=?, entry_type=?, title=?, username=?, password=?,
		 link=?, totp_secret=?, card_number=?, expiry=?, cvv=?, custom_text=?,
		 file_name=?, file_data=? WHERE id=?`,
		folderID, e.Type, e.Title, e.Username, e.Password,
		e.Link, e.TotpSecret, e.CardNumber, e.Expiry, e.CVV, e.CustomText,
		e.FileName, e.FileData, id)
	if err != nil {
		return err
	}
	return s.replaceHistory(id, e.History)
}

func (s *Store) replaceHistory(entryID int64, history []PasswordHistory) error {
	if _, err := s.db.Exec("DELETE FROM password_history WHERE entry_id = ?", entryID); err != nil {
		return err
	}
	for _, h := range history {
		if _, err := s.db.Exec(
			"INSERT INTO password_history (entry_id, password, date) VALUES (?, ?, ?)",
			entryID, h.Password, h.Date); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LoadEntry(id int64) (*EntryFull, error) {
	e := &EntryFull{ID: id}
	err := s.db.QueryRow(
		`SELECT folder_id, entry_type, title, username, password, link, totp_secret,
		 card_number, expiry, cvv, custom_text, file_name, file_data
		 FROM entries WHERE id = ?`, id,
	).Scan(&e.FolderID, &e.Type, &e.Title, &e.Username, &e.Password, &e.Link,
		&e.TotpSecret, &e.CardNumber, &e.Expiry, &e.CVV, &e.CustomText,
		&e.FileName, &e.FileData)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		"SELECT password, date FROM password_history WHERE entry_id = ? ORDER BY id", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var h PasswordHistory
			if rows.Scan(&h.Password, &h.Date) == nil {
				e.History = append(e.History, h)
			}
		}
	}

	attRows, err := s.db.Query(
		"SELECT id, file_name, size FROM attachments WHERE entry_id = ? ORDER BY file_name", id)
	if err == nil {
		defer attRows.Close()
		for attRows.Next() {
			var a AttachmentMeta
			if attRows.Scan(&a.ID, &a.FileName, &a.Size) == nil {
				e.Attachments = append(e.Attachments, a)
			}
		}
	}

	return e, nil
}

func (s *Store) ListEntries(folderID int64) ([]EntryMeta, error) {
	rows, err := s.db.Query(
		"SELECT id, folder_id, title, entry_type FROM entries WHERE folder_id = ? ORDER BY title",
		folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryMeta
	for rows.Next() {
		var e EntryMeta
		if err := rows.Scan(&e.ID, &e.FolderID, &e.Title, &e.EntryType); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) ListAllEntries() ([]EntryMeta, error) {
	rows, err := s.db.Query(
		"SELECT id, folder_id, title, entry_type FROM entries ORDER BY folder_id, title")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryMeta
	for rows.Next() {
		var e EntryMeta
		if err := rows.Scan(&e.ID, &e.FolderID, &e.Title, &e.EntryType); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) GetEntryMeta(id int64) (*EntryMeta, error) {
	var e EntryMeta
	err := s.db.QueryRow(
		"SELECT id, folder_id, title, entry_type FROM entries WHERE id = ?", id,
	).Scan(&e.ID, &e.FolderID, &e.Title, &e.EntryType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) DeleteEntry(id int64) error {
	_, err := s.db.Exec("DELETE FROM entries WHERE id = ?", id)
	return err
}

func (s *Store) EntryExistsInFolder(folderID int64, title string) bool {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM entries WHERE folder_id = ? AND title = ?",
		folderID, title).Scan(&count)
	return err == nil && count > 0
}

func (s *Store) EntryExistsInFolderExcluding(folderID int64, title string, excludeID int64) bool {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM entries WHERE folder_id = ? AND title = ? AND id != ?",
		folderID, title, excludeID).Scan(&count)
	return err == nil && count > 0
}

func (s *Store) HasEntries() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM entries").Scan(&count)
	return err == nil && count > 0
}

func (s *Store) CountEntriesInFolder(folderID int64) int {
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE folder_id = ?", folderID).Scan(&count)
	return count
}

// ── Attachments ─────────────────────────────────────────────────────

func (s *Store) ReadAttachment(id string) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM attachments WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attachment not found: %s", id)
	}
	return data, err
}

func (s *Store) WriteAttachment(id string, entryID int64, fileName string, size int64, data []byte) error {
	_, err := s.db.Exec(
		`INSERT INTO attachments (id, entry_id, file_name, size, data) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, file_name = excluded.file_name, size = excluded.size`,
		id, entryID, fileName, size, data)
	return err
}

func (s *Store) DeleteAttachment(id string) error {
	_, err := s.db.Exec("DELETE FROM attachments WHERE id = ?", id)
	return err
}

// ── PIN Config ──────────────────────────────────────────────────────

func (s *Store) ReadPinConfig() (*PinConfig, error) {
	var cfg PinConfig
	err := s.db.QueryRow(
		"SELECT mode, pin_key, pin_tag, totp_secret FROM pin_config WHERE id = 1",
	).Scan(&cfg.Mode, &cfg.PinKey, &cfg.PinTag, &cfg.TotpSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if cfg.Mode == "" {
		return nil, nil
	}
	return &cfg, nil
}

func (s *Store) WritePinConfig(cfg *PinConfig) error {
	_, err := s.db.Exec(
		`INSERT INTO pin_config (id, mode, pin_key, pin_tag, totp_secret) VALUES (1, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET mode=excluded.mode, pin_key=excluded.pin_key,
		 pin_tag=excluded.pin_tag, totp_secret=excluded.totp_secret`,
		cfg.Mode, cfg.PinKey, cfg.PinTag, cfg.TotpSecret)
	return err
}

func (s *Store) PinConfigExists() bool {
	cfg, err := s.ReadPinConfig()
	return err == nil && cfg != nil
}
