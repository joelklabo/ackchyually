package store

const schema = `
CREATE TABLE IF NOT EXISTS tool_identities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  exe_path TEXT NOT NULL,
  sha256 TEXT NOT NULL UNIQUE,
  version_str TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invocations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME NOT NULL,
  duration_ms INTEGER NOT NULL,
  context_key TEXT NOT NULL,
  tool TEXT NOT NULL,
  exe_path TEXT NOT NULL,
  tool_id INTEGER,
  argv_json TEXT NOT NULL,
  exit_code INTEGER NOT NULL,
  mode TEXT NOT NULL,
  stdout_tail TEXT NOT NULL,
  stderr_tail TEXT NOT NULL,
  combined_tail TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS invocations_lookup
  ON invocations(tool, context_key, exit_code, created_at);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME NOT NULL,
  context_key TEXT NOT NULL,
  tag TEXT NOT NULL,
  tool TEXT NOT NULL,
  argv_json TEXT NOT NULL,
  UNIQUE(context_key, tag)
);
`
