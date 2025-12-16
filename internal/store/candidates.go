package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type SuccessCandidate struct {
	Argv  []string
	Count int
	Last  time.Time
}

func (db *DB) ListSuccessCandidates(tool, ctxKey string, limit int) ([]SuccessCandidate, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT argv_json, COUNT(*) as n, MAX(created_at) as last_at
FROM invocations
WHERE tool = ? AND context_key = ? AND exit_code = 0
GROUP BY argv_json
ORDER BY last_at DESC
LIMIT ?`, tool, ctxKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SuccessCandidate
	for rows.Next() {
		var argvJSON string
		var n int
		var lastRaw sql.NullString
		if err := rows.Scan(&argvJSON, &n, &lastRaw); err != nil {
			continue
		}

		var argv []string
		if err := json.Unmarshal([]byte(argvJSON), &argv); err != nil || len(argv) == 0 {
			continue
		}

		out = append(out, SuccessCandidate{Argv: argv, Count: n, Last: parseDBTime(lastRaw.String)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseDBTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if i := strings.Index(s, " m="); i != -1 {
		s = strings.TrimSpace(s[:i])
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
