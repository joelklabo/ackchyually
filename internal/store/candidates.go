package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

type SuccessCandidate struct {
	Argv  []string
	Count int
	Last  time.Time
}

func (db *DB) ListSuccessCandidates(tool, ctxKey string, limit int) ([]SuccessCandidate, error) {
	rows, err := db.Query(`
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
		var last sql.NullTime
		if err := rows.Scan(&argvJSON, &n, &last); err != nil {
			continue
		}

		var argv []string
		if err := json.Unmarshal([]byte(argvJSON), &argv); err != nil || len(argv) == 0 {
			continue
		}

		out = append(out, SuccessCandidate{Argv: argv, Count: n, Last: last.Time})
	}
	return out, nil
}
