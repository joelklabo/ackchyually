package app

import (
	"strings"
	"testing"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestTagAdd_Errors(t *testing.T) {
	setTempHomeAndCWD(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no args",
			args: []string{},
			want: "usage: ackchyually tag add",
		},
		{
			name: "missing separator",
			args: []string{"mytag", "git", "status"},
			want: "usage: ackchyually tag add",
		},
		{
			name: "separator at end",
			args: []string{"mytag", "--"},
			want: "usage: ackchyually tag add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := captureStdoutStderr(t, func() int {
				return tagAdd(tt.args)
			})
			if code != 2 {
				t.Errorf("tagAdd(%v) code = %d, want 2", tt.args, code)
			}
			if !strings.Contains(errOut, tt.want) {
				t.Errorf("tagAdd(%v) stderr missing %q, got:\n%s", tt.args, tt.want, errOut)
			}
		})
	}
}

func TestTagRun_Errors(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	// Seed a corrupt tag
	err := store.WithDB(func(db *store.DB) error {
		return db.UpsertTag(store.Tag{
			ContextKey: ctxKey,
			Tag:        "corrupt",
			Tool:       "git",
			ArgvJSON:   "{invalid-json",
		})
	})
	if err != nil {
		t.Fatalf("seed corrupt tag: %v", err)
	}

	// Seed an empty argv tag
	err = store.WithDB(func(db *store.DB) error {
		return db.UpsertTag(store.Tag{
			ContextKey: ctxKey,
			Tag:        "empty",
			Tool:       "git",
			ArgvJSON:   "[]",
		})
	})
	if err != nil {
		t.Fatalf("seed empty tag: %v", err)
	}

	tests := []struct {
		name string
		args []string
		want string
		code int
	}{
		{
			name: "no args",
			args: []string{},
			want: "usage: ackchyually tag run",
			code: 2,
		},
		{
			name: "tag not found",
			args: []string{"missing"},
			want: "tag not found: missing",
			code: 1,
		},
		{
			name: "corrupt tag",
			args: []string{"corrupt"},
			want: "corrupt tag argv",
			code: 1,
		},
		{
			name: "empty tag",
			args: []string{"empty"},
			want: "corrupt tag argv",
			code: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := captureStdoutStderr(t, func() int {
				return tagRun(tt.args)
			})
			if code != tt.code {
				t.Errorf("tagRun(%v) code = %d, want %d", tt.args, code, tt.code)
			}
			if !strings.Contains(errOut, tt.want) {
				t.Errorf("tagRun(%v) stderr missing %q, got:\n%s", tt.args, tt.want, errOut)
			}
		})
	}
}
