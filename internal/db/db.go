package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Target struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Type      string    `json:"type"` // http, tcp, ping, dns, visual
	Interval  int       `json:"interval_seconds"`
	Selector  string    `json:"selector,omitempty"` // CSS selector for change detection
	Headers   string    `json:"headers,omitempty"`  // JSON string of custom headers
	Expect    string    `json:"expect,omitempty"`   // Expected keyword in response
	Timeout   int       `json:"timeout,omitempty"`  // Per-target timeout in seconds
	Retries   int       `json:"retries,omitempty"`  // Retry count before marking down
	Threshold   float64   `json:"threshold,omitempty"`    // Visual diff threshold percentage (default 5.0)
	TriggerRule  string    `json:"trigger_rule,omitempty"`  // JSON trigger condition for notifications
	JQFilter     string    `json:"jq_filter,omitempty"`     // jq expression to filter JSON responses
	Method       string    `json:"method,omitempty"`        // HTTP method (GET, POST, etc.)
	Body         string    `json:"body,omitempty"`          // Request body for POST/PUT/PATCH
	NoFollow     bool      `json:"no_follow,omitempty"`     // Don't follow redirects
	AcceptStatus string    `json:"accept_status,omitempty"` // Accepted status codes (e.g. "200-299,301")
	Insecure     bool      `json:"insecure,omitempty"`      // Skip TLS verification
	CreatedAt    time.Time `json:"created_at"`
	Paused       bool      `json:"paused"`
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
	// Use XDG_DATA_HOME if set, otherwise fall back to ~/.upp
	// This ensures consistency between CLI and daemon/systemd contexts
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		newDir := filepath.Join(xdg, "upp")
		oldDir := filepath.Join(xdg, "watchdog")
		
		// Migrate from old directory if needed
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			if _, err := os.Stat(oldDir); err == nil {
				os.Rename(oldDir, newDir)
			}
		}
		// Rename old DB file if needed
		oldDB := filepath.Join(newDir, "watchdog.db")
		newDB := filepath.Join(newDir, "upp.db")
		if _, err := os.Stat(newDB); os.IsNotExist(err) {
			if _, err := os.Stat(oldDB); err == nil {
				os.Rename(oldDB, newDB)
			}
		}
		
		os.MkdirAll(newDir, 0755)
		return filepath.Join(newDir, "upp.db")
	}

	home := getHomeDir()
	newDir := filepath.Join(home, ".upp")
	oldDir := filepath.Join(home, ".watchdog")
	
	// Migrate from old directory if needed
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		if _, err := os.Stat(oldDir); err == nil {
			os.Rename(oldDir, newDir)
		}
	}
	
	// Rename old DB file if needed
	oldDB := filepath.Join(newDir, "watchdog.db")
	newDB := filepath.Join(newDir, "upp.db")
	if _, err := os.Stat(newDB); os.IsNotExist(err) {
		if _, err := os.Stat(oldDB); err == nil {
			os.Rename(oldDB, newDB)
		}
	}
	
	os.MkdirAll(newDir, 0755)
	return filepath.Join(newDir, "upp.db")
}

// getHomeDir returns the current user's home directory reliably,
// even in contexts where $HOME is not set (e.g. systemd services).
func getHomeDir() string {
	// Prefer $HOME if set
	if home := os.Getenv("HOME"); home != "" {
		return home
	}

	// Fall back to os.UserHomeDir (reads /etc/passwd on Linux)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}

	// Last resort: use current user from /etc/passwd via os/user
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}

	return "/"
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
		expect TEXT DEFAULT '',
		timeout INTEGER DEFAULT 30,
		retries INTEGER DEFAULT 1,
		threshold REAL DEFAULT 5.0,
		trigger_rule TEXT DEFAULT '',
		jq_filter TEXT DEFAULT '',
		method TEXT DEFAULT '',
		body TEXT DEFAULT '',
		no_follow INTEGER DEFAULT 0,
		accept_status TEXT DEFAULT '',
		insecure INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		paused INTEGER DEFAULT 0,
		UNIQUE(url, type, selector)
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
	if err != nil {
		return err
	}

	// Migration: Add threshold column if it doesn't exist (for existing databases)
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN threshold REAL DEFAULT 5.0")
	if err != nil {
		// Column might already exist, which is fine
		// SQLite returns an error if column already exists
		if !strings.Contains(err.Error(), "duplicate column name") {
			// If it's not a duplicate column error, something else went wrong
			return err
		}
	}

	// Migration: Add trigger_rule column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN trigger_rule TEXT DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add jq_filter column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN jq_filter TEXT DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add method column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN method TEXT DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add body column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN body TEXT DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add no_follow column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN no_follow INTEGER DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add accept_status column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN accept_status TEXT DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Add insecure column
	_, err = db.Exec("ALTER TABLE targets ADD COLUMN insecure INTEGER DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Migration: Update unique constraint from (url, selector) to (url, type, selector)
	// SQLite can't alter constraints, so we recreate the table
	var tableSql string
	db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='targets'").Scan(&tableSql)
	if strings.Contains(tableSql, "UNIQUE(url, selector)") && !strings.Contains(tableSql, "UNIQUE(url, type, selector)") {
		db.Exec(`CREATE TABLE targets_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'http',
			interval_seconds INTEGER NOT NULL DEFAULT 300,
			selector TEXT DEFAULT '',
			headers TEXT DEFAULT '',
			expect TEXT DEFAULT '',
			timeout INTEGER DEFAULT 30,
			retries INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			paused INTEGER DEFAULT 0,
			threshold REAL DEFAULT 5.0,
			trigger_rule TEXT DEFAULT '',
			jq_filter TEXT DEFAULT '',
			method TEXT DEFAULT '',
			body TEXT DEFAULT '',
			no_follow INTEGER DEFAULT 0,
			accept_status TEXT DEFAULT '',
			insecure INTEGER DEFAULT 0,
			UNIQUE(url, type, selector)
		)`)
		db.Exec(`INSERT INTO targets_new SELECT * FROM targets`)
		db.Exec(`DROP TABLE targets`)
		db.Exec(`ALTER TABLE targets_new RENAME TO targets`)
	}
	
	return nil
}

func DB() *sql.DB {
	return db
}

type AddTargetOpts struct {
	TriggerRule  string
	JQFilter     string
	Method       string
	Body         string
	NoFollow     bool
	AcceptStatus string
	Insecure     bool
}

func AddTarget(name, url, typ string, interval int, selector, headers, expect string, timeout, retries int, threshold float64, opts AddTargetOpts) (*Target, error) {
	if name == "" {
		name = url
	}
	if timeout <= 0 {
		timeout = 30
	}
	if retries <= 0 {
		retries = 1
	}
	if threshold <= 0 {
		threshold = 5.0
	}
	noFollow := 0
	if opts.NoFollow {
		noFollow = 1
	}
	insecure := 0
	if opts.Insecure {
		insecure = 1
	}
	res, err := db.Exec(
		"INSERT INTO targets (name, url, type, interval_seconds, selector, headers, expect, timeout, retries, threshold, trigger_rule, jq_filter, method, body, no_follow, accept_status, insecure) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		name, url, typ, interval, selector, headers, expect, timeout, retries, threshold, opts.TriggerRule, opts.JQFilter, opts.Method, opts.Body, noFollow, opts.AcceptStatus, insecure,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add target (may already exist): %w", err)
	}
	id, _ := res.LastInsertId()
	return &Target{ID: id, Name: name, URL: url, Type: typ, Interval: interval, Selector: selector, Headers: headers, Expect: expect, Timeout: timeout, Retries: retries, Threshold: threshold, TriggerRule: opts.TriggerRule, JQFilter: opts.JQFilter, Method: opts.Method, Body: opts.Body, NoFollow: opts.NoFollow, AcceptStatus: opts.AcceptStatus, Insecure: opts.Insecure, CreatedAt: time.Now()}, nil
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
	rows, err := db.Query("SELECT id, name, url, type, interval_seconds, selector, headers, expect, timeout, retries, threshold, trigger_rule, jq_filter, method, body, no_follow, accept_status, insecure, created_at, paused FROM targets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		var paused, noFollow, insecure int
		err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.Type, &t.Interval, &t.Selector, &t.Headers, &t.Expect, &t.Timeout, &t.Retries, &t.Threshold, &t.TriggerRule, &t.JQFilter, &t.Method, &t.Body, &noFollow, &t.AcceptStatus, &insecure, &t.CreatedAt, &paused)
		if err != nil {
			return nil, err
		}
		t.Paused = paused == 1
		t.NoFollow = noFollow == 1
		t.Insecure = insecure == 1
		targets = append(targets, t)
	}
	return targets, nil
}

func GetTarget(identifier string) (*Target, error) {
	var t Target
	var paused, noFollow, insecure int
	err := db.QueryRow(
		"SELECT id, name, url, type, interval_seconds, selector, headers, expect, timeout, retries, threshold, trigger_rule, jq_filter, method, body, no_follow, accept_status, insecure, created_at, paused FROM targets WHERE name = ? OR url = ? OR id = ?",
		identifier, identifier, identifier,
	).Scan(&t.ID, &t.Name, &t.URL, &t.Type, &t.Interval, &t.Selector, &t.Headers, &t.Expect, &t.Timeout, &t.Retries, &t.Threshold, &t.TriggerRule, &t.JQFilter, &t.Method, &t.Body, &noFollow, &t.AcceptStatus, &insecure, &t.CreatedAt, &paused)
	if err != nil {
		return nil, fmt.Errorf("target not found: %s", identifier)
	}
	t.Paused = paused == 1
	t.NoFollow = noFollow == 1
	t.Insecure = insecure == 1
	return &t, nil
}

func UpdateTarget(t *Target) error {
	noFollow := 0
	if t.NoFollow {
		noFollow = 1
	}
	insecure := 0
	if t.Insecure {
		insecure = 1
	}
	res, err := db.Exec(
		`UPDATE targets SET name=?, url=?, type=?, interval_seconds=?, selector=?, headers=?, expect=?, timeout=?, retries=?, threshold=?, trigger_rule=?, jq_filter=?, method=?, body=?, no_follow=?, accept_status=?, insecure=? WHERE id=?`,
		t.Name, t.URL, t.Type, t.Interval, t.Selector, t.Headers, t.Expect, t.Timeout, t.Retries, t.Threshold, t.TriggerRule, t.JQFilter, t.Method, t.Body, noFollow, t.AcceptStatus, insecure, t.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("target not found: %d", t.ID)
	}
	return nil
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

func SetPaused(identifier string, paused bool) error {
	val := 0
	if paused {
		val = 1
	}
	res, err := db.Exec("UPDATE targets SET paused = ? WHERE name = ? OR url = ? OR id = ?", val, identifier, identifier, identifier)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("target not found: %s", identifier)
	}
	return nil
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
