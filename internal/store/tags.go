package store

import (
	"context"
	"database/sql"
)

func (db *DB) ListTags(ctxKey string, tool string) ([]Tag, error) {
	var rows *sql.Rows
	var err error

	if tool == "" {
		rows, err = db.QueryContext(context.Background(), `
SELECT context_key, tag, tool, argv_json
FROM tags
WHERE context_key = ?
ORDER BY tag ASC`, ctxKey)
	} else {
		rows, err = db.QueryContext(context.Background(), `
SELECT context_key, tag, tool, argv_json
FROM tags
WHERE context_key = ? AND tool = ?
ORDER BY tag ASC`, ctxKey, tool)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ContextKey, &t.Tag, &t.Tool, &t.ArgvJSON); err != nil {
			continue
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
