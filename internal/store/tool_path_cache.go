package store

import "context"

type ToolPathCache struct {
	ExePath     string
	FileSize    int64
	FileMtimeNS int64
	SHA256      string
}

func (db *DB) GetToolPathCache(exePath string) (ToolPathCache, error) {
	var c ToolPathCache
	err := db.QueryRowContext(context.Background(), `
SELECT exe_path, file_size, file_mtime_ns, sha256
FROM tool_path_cache
WHERE exe_path = ?`, exePath).Scan(&c.ExePath, &c.FileSize, &c.FileMtimeNS, &c.SHA256)
	if err != nil {
		return ToolPathCache{}, err
	}
	return c, nil
}

func (db *DB) UpsertToolPathCache(c ToolPathCache) error {
	_, err := db.ExecContext(context.Background(), `
INSERT INTO tool_path_cache(exe_path, file_size, file_mtime_ns, sha256, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(exe_path) DO UPDATE SET
  file_size=excluded.file_size,
  file_mtime_ns=excluded.file_mtime_ns,
  sha256=excluded.sha256,
  updated_at=CURRENT_TIMESTAMP`,
		c.ExePath, c.FileSize, c.FileMtimeNS, c.SHA256,
	)
	return err
}
