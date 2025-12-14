package store

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // register sqlite driver
)

type DB struct{ *sql.DB }

type Invocation struct {
	At           time.Time
	DurationMS   int64
	ContextKey   string
	Tool         string
	ExePath      string
	ToolID       int64
	ArgvJSON     string
	ExitCode     int
	Mode         string
	StdoutTail   string
	StderrTail   string
	CombinedTail string
}

type ToolIdentity struct {
	ID         int64
	ExePath    string
	SHA256     string
	VersionStr string
}

type Tag struct {
	ContextKey string
	Tag        string
	Tool       string
	ArgvJSON   string
}

func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "ackchyually")
}

func dbPath() string { return filepath.Join(dataDir(), "ackchyually.sqlite") }

func Open() (*DB, error) {
	if err := os.MkdirAll(dataDir(), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath())
	if err != nil {
		return nil, err
	}

	var _mode string
	if err := db.QueryRow(`PRAGMA journal_mode=WAL;`).Scan(&_mode); err != nil {
		_ = err // best-effort
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func WithDB(fn func(*DB) error) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()
	return fn(db)
}

func (db *DB) InsertInvocation(inv Invocation) error {
	_, err := db.Exec(`
INSERT INTO invocations
(created_at, duration_ms, context_key, tool, exe_path, tool_id, argv_json, exit_code, mode, stdout_tail, stderr_tail, combined_tail)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.At, inv.DurationMS, inv.ContextKey, inv.Tool, inv.ExePath, nullIfZero(inv.ToolID),
		inv.ArgvJSON, inv.ExitCode, inv.Mode, inv.StdoutTail, inv.StderrTail, inv.CombinedTail,
	)
	return err
}

func (db *DB) UpsertTool(t ToolIdentity) (int64, error) {
	if _, err := db.Exec(`INSERT OR IGNORE INTO tool_identities(exe_path, sha256, version_str) VALUES (?, ?, ?)`,
		t.ExePath, t.SHA256, t.VersionStr,
	); err != nil {
		return 0, err
	}
	var id int64
	err := db.QueryRow(`SELECT id FROM tool_identities WHERE sha256 = ?`, t.SHA256).Scan(&id)
	return id, err
}

func (db *DB) GetToolBySHA(sha string) (ToolIdentity, error) {
	var t ToolIdentity
	err := db.QueryRow(`SELECT id, exe_path, sha256, version_str FROM tool_identities WHERE sha256 = ?`, sha).
		Scan(&t.ID, &t.ExePath, &t.SHA256, &t.VersionStr)
	return t, err
}

func (db *DB) ListSuccessful(tool, ctxKey string, limit int) ([][]string, error) {
	rows, err := db.Query(`
SELECT argv_json FROM invocations
WHERE tool = ? AND context_key = ? AND exit_code = 0
ORDER BY created_at DESC
LIMIT ?`, tool, ctxKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][]string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		var argv []string
		if err := json.Unmarshal([]byte(s), &argv); err != nil {
			continue
		}
		out = append(out, argv)
	}
	return out, nil
}

func (db *DB) UpsertTag(t Tag) error {
	_, err := db.Exec(`
INSERT INTO tags(created_at, context_key, tag, tool, argv_json)
VALUES (CURRENT_TIMESTAMP, ?, ?, ?, ?)
ON CONFLICT(context_key, tag) DO UPDATE SET tool=excluded.tool, argv_json=excluded.argv_json
`, t.ContextKey, t.Tag, t.Tool, t.ArgvJSON)
	return err
}

func (db *DB) GetTag(ctxKey, tag string) (Tag, error) {
	var t Tag
	err := db.QueryRow(`SELECT context_key, tag, tool, argv_json FROM tags WHERE context_key=? AND tag=?`, ctxKey, tag).
		Scan(&t.ContextKey, &t.Tag, &t.Tool, &t.ArgvJSON)
	return t, err
}

func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
