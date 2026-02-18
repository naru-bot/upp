package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Target struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Type      string    `json:"type"` // http, tcp, ping, dns
	Interval  int       `json:"interval_seconds"`
	Selector  string    `json:"selector,omitempty"` // CSS selector for change detection
	Headers   string    `json:"headers,omitempty"`  // JSON string of custom headers
	CreatedAt time.Time `json:"created_at"`
	Paused    bool      `json:"paused"`
}

type CheckResult struct {
	ID           int64     `json:"id"`
	TargetID     int64     `json:"target_id"`
	Status       string    `json:"status"` // up, down, changed, unchanged, error
	StatusCode   int       `json:"status_code,omitempty"`
	ResponseTime int64     `json:"response_time_ms"`
	ContentHash  string    `json:"content_hash,omitempty"`
	Error        string    `json:"error,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

type Snapshot struct {
	ID        int64     `json:"id"`
	TargetID  int64     `json:"target_id"`
	Content   string    `json:"content"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

type NotifyConfig struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // webhook, command, slack, telegram, discord
	Config   string `json:"config"` // JSON config
	Enabled  bool   `json:"enabled"`
}

var db *sql.DB

func GetDBPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".watchdog")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "watchdog.db")
}

func Init() error {
	return InitWithPath(GetDBPath())
}

func InitWithPath(path string) error {
	var err error
	db, err = sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'http',
		interval_seconds INTEGER NOT NULL DEFAULT 300,
		selector TEXT DEFAULT '',
		headers TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		paused INTEGER DEFAULT 0,
		UNIQUE(url, selector)
	);

	CREATE TABLE IF NOT EXISTS check_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		status_code INTEGER DEFAULT 0,
		response_time_ms INTEGER DEFAULT 0,
		content_hash TEXT DEFAULT '',
		error TEXT DEFAULT '',
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS notify_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		config TEXT NOT NULL,
		enabled INTEGER DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_results_target ON check_results(target_id, checked_at);
	CREATE INDEX IF NOT EXISTS idx_snapshots_target ON snapshots(target_id, created_at);
	`
	_, err = db.Exec(schema)
	return err
}

func DB() *sql.DB {
	return db
}

func AddTarget(name, url, typ string, interval int, selector, headers string) (*Target, error) {
	if name == "" {
		name = url
	}
	res, err := db.Exec(
		"INSERT INTO targets (name, url, type, interval_seconds, selector, headers) VALUES (?, ?, ?, ?, ?, ?)",
		name, url, typ, interval, selector, headers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add target (may already exist): %w", err)
	}
	id, _ := res.LastInsertId()
	return &Target{ID: id, Name: name, URL: url, Type: typ, Interval: interval, Selector: selector, Headers: headers, CreatedAt: time.Now()}, nil
}

func RemoveTarget(identifier string) error {
	// Try by name first, then URL, then ID
	res, err := db.Exec("DELETE FROM targets WHERE name = ? OR url = ? OR id = ?", identifier, identifier, identifier)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("target not found: %s", identifier)
	}
	return nil
}

func ListTargets() ([]Target, error) {
	rows, err := db.Query("SELECT id, name, url, type, interval_seconds, selector, headers, created_at, paused FROM targets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		var paused int
		err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.Type, &t.Interval, &t.Selector, &t.Headers, &t.CreatedAt, &paused)
		if err != nil {
			return nil, err
		}
		t.Paused = paused == 1
		targets = append(targets, t)
	}
	return targets, nil
}

func GetTarget(identifier string) (*Target, error) {
	var t Target
	var paused int
	err := db.QueryRow(
		"SELECT id, name, url, type, interval_seconds, selector, headers, created_at, paused FROM targets WHERE name = ? OR url = ? OR id = ?",
		identifier, identifier, identifier,
	).Scan(&t.ID, &t.Name, &t.URL, &t.Type, &t.Interval, &t.Selector, &t.Headers, &t.CreatedAt, &paused)
	if err != nil {
		return nil, fmt.Errorf("target not found: %s", identifier)
	}
	t.Paused = paused == 1
	return &t, nil
}

func SaveCheckResult(r *CheckResult) error {
	_, err := db.Exec(
		"INSERT INTO check_results (target_id, status, status_code, response_time_ms, content_hash, error) VALUES (?, ?, ?, ?, ?, ?)",
		r.TargetID, r.Status, r.StatusCode, r.ResponseTime, r.ContentHash, r.Error,
	)
	return err
}

func GetCheckHistory(targetID int64, limit int) ([]CheckResult, error) {
	rows, err := db.Query(
		"SELECT id, target_id, status, status_code, response_time_ms, content_hash, error, checked_at FROM check_results WHERE target_id = ? ORDER BY checked_at DESC LIMIT ?",
		targetID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var r CheckResult
		err := rows.Scan(&r.ID, &r.TargetID, &r.Status, &r.StatusCode, &r.ResponseTime, &r.ContentHash, &r.Error, &r.CheckedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func SaveSnapshot(targetID int64, content, hash string) error {
	_, err := db.Exec(
		"INSERT INTO snapshots (target_id, content, hash) VALUES (?, ?, ?)",
		targetID, content, hash,
	)
	return err
}

func GetLatestSnapshots(targetID int64, limit int) ([]Snapshot, error) {
	rows, err := db.Query(
		"SELECT id, target_id, content, hash, created_at FROM snapshots WHERE target_id = ? ORDER BY created_at DESC LIMIT ?",
		targetID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []Snapshot
	for rows.Next() {
		var s Snapshot
		err := rows.Scan(&s.ID, &s.TargetID, &s.Content, &s.Hash, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, nil
}

func GetUptimeStats(targetID int64, since time.Time) (total int, up int, avgResponseMs float64, err error) {
	err = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status='up' OR status='unchanged' OR status='changed' THEN 1 ELSE 0 END), 0), COALESCE(AVG(response_time_ms), 0)
		FROM check_results WHERE target_id = ? AND checked_at >= ?`,
		targetID, since,
	).Scan(&total, &up, &avgResponseMs)
	return
}

func SaveNotifyConfig(name, typ, config string) error {
	_, err := db.Exec("INSERT INTO notify_configs (name, type, config) VALUES (?, ?, ?)", name, typ, config)
	return err
}

func ListNotifyConfigs() ([]NotifyConfig, error) {
	rows, err := db.Query("SELECT id, name, type, config, enabled FROM notify_configs ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []NotifyConfig
	for rows.Next() {
		var c NotifyConfig
		var enabled int
		err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Config, &enabled)
		if err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		configs = append(configs, c)
	}
	return configs, nil
}

func RemoveNotifyConfig(identifier string) error {
	res, err := db.Exec("DELETE FROM notify_configs WHERE name = ? OR id = ?", identifier, identifier)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("notification config not found: %s", identifier)
	}
	return nil
}
