package toolid

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joelklabo/ackchyually/internal/store"
)

type ToolIdentity struct {
	ID         int64
	ExePath    string
	SHA256     string
	VersionStr string
}

var mu sync.Mutex

func Identify(exe string) (ToolIdentity, error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := os.Stat(exe)
	if err != nil {
		return ToolIdentity{}, err
	}
	size := st.Size()
	mtimeNS := st.ModTime().UnixNano()

	var ti ToolIdentity
	err = store.WithDB(func(db *store.DB) error {
		sha := ""
		cached, err2 := db.GetToolPathCache(exe)
		if err2 == nil {
			if cached.FileSize == size && cached.FileMtimeNS == mtimeNS {
				sha = cached.SHA256
			}
		}
		if sha == "" {
			sha2, err3 := sha256File(exe)
			if err3 != nil {
				return err3
			}
			sha = sha2
			if err := db.UpsertToolPathCache(store.ToolPathCache{
				ExePath:     exe,
				FileSize:    size,
				FileMtimeNS: mtimeNS,
				SHA256:      sha,
			}); err != nil {
				_ = err // best-effort
			}
		}

		found, err2 := db.GetToolBySHA(sha)
		if err2 == nil && found.ID != 0 {
			ti = ToolIdentity{
				ID:         found.ID,
				ExePath:    found.ExePath,
				SHA256:     found.SHA256,
				VersionStr: found.VersionStr,
			}
			return nil
		}

		ver := detectVersion(exe)
		id, err3 := db.UpsertTool(store.ToolIdentity{
			ExePath:    exe,
			SHA256:     sha,
			VersionStr: ver,
		})
		if err3 != nil {
			return err3
		}
		ti = ToolIdentity{ID: id, ExePath: exe, SHA256: sha, VersionStr: ver}
		return nil
	})
	return ti, err
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func detectVersion(exe string) string {
	base := filepath.Base(exe)
	candidates := [][]string{{"--version"}, {"version"}, {"-V"}, {"-v"}}
	for _, argv := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		cmd := exec.CommandContext(ctx, exe, argv...)
		out, err := cmd.CombinedOutput()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			continue
		}

		s := strings.TrimSpace(string(out))
		if s == "" {
			continue
		}
		if len(s) > 4096 {
			s = s[:4096]
		}

		// Accept version-ish output even if the tool exits non-zero.
		if err == nil || s != "" {
			return base + " " + s
		}
	}
	return base + " (version unknown)"
}
