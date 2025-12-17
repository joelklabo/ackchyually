package logscan

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Options struct {
	// MaxFiles limits the number of files scanned per tool (including history files).
	MaxFiles int
	// MaxBytes limits the number of bytes read from each file (tail read).
	MaxBytes int64
}

func (o Options) withDefaults() Options {
	if o.MaxFiles <= 0 {
		o.MaxFiles = 20
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = 1 << 20 // 1 MiB
	}
	return o
}

type Summary struct {
	Tool          string
	FilesScanned  int
	FilesErrored  int
	FilesWithShim int
	FilesWithAbs  int
	BytesRead     int64
	FileNames     []string
}

func (s Summary) FoundShim() bool { return s.FilesWithShim > 0 }
func (s Summary) FoundAbs() bool  { return s.FilesWithAbs > 0 }

func ScanCodex(homeDir, shimDir string, opts Options) Summary {
	opts = opts.withDefaults()
	codexDir := filepath.Join(homeDir, ".codex")

	var files []string
	history := filepath.Join(codexDir, "history.jsonl")
	if fileExists(history) {
		files = append(files, history)
	}

	sessionsDir := filepath.Join(codexDir, "sessions")
	sessionFiles := listFilesBySuffixRecursive(sessionsDir, ".jsonl")
	files = append(files, takeMostRecent(sessionFiles, maxInt(0, opts.MaxFiles-len(files)))...)

	return scanFiles("codex", homeDir, shimDir, files, opts)
}

func ScanCopilot(homeDir, shimDir string, opts Options) Summary {
	opts = opts.withDefaults()

	// Primary location per GitHub Copilot CLI docs: ~/.copilot (config/state).
	// Logs are commonly under ~/.copilot/logs/ when present.
	root := filepath.Join(homeDir, ".copilot")
	logsDir := filepath.Join(root, "logs")

	var files []string
	files = append(files, takeMostRecent(listFilesAllRecursive(logsDir), opts.MaxFiles)...)
	if len(files) == 0 {
		// Fallback: if logs/ is absent, scan top-level *.log / *.jsonl files under ~/.copilot.
		files = append(files, takeMostRecent(listFilesMatching(root, func(name string) bool {
			l := strings.ToLower(name)
			return strings.HasSuffix(l, ".log") || strings.HasSuffix(l, ".log.txt") || strings.HasSuffix(l, ".jsonl")
		}), opts.MaxFiles)...)
	}

	return scanFiles("copilot", homeDir, shimDir, files, opts)
}

func scanFiles(tool, homeDir, shimDir string, paths []string, opts Options) Summary {
	sum := Summary{Tool: tool}
	if len(paths) > opts.MaxFiles {
		paths = paths[:opts.MaxFiles]
	}

	for _, p := range paths {
		b, n, err := readTailBytes(p, opts.MaxBytes)
		sum.FilesScanned++
		sum.BytesRead += n
		sum.FileNames = append(sum.FileNames, homePath(p, homeDir))
		if err != nil {
			sum.FilesErrored++
			continue
		}

		foundShim, foundAbs := scanBytesForGitPaths(b, shimDir)
		if foundShim {
			sum.FilesWithShim++
		}
		if foundAbs {
			sum.FilesWithAbs++
		}
	}

	return sum
}

func scanBytesForGitPaths(b []byte, shimDir string) (foundShim bool, foundAbs bool) {
	// Scan raw bytes; do not attempt to parse JSONL (robust to malformed/non-UTF8).
	shimPatterns := []string{
		filepath.Join(shimDir, "git"),
		filepath.Join(shimDir, "git.exe"),
	}
	for _, p := range shimPatterns {
		if p == "" {
			continue
		}
		// Also check with forward slashes in case logs normalize separators.
		if bytes.Contains(b, []byte(p)) || bytes.Contains(b, []byte(filepath.ToSlash(p))) {
			foundShim = true
			break
		}
	}

	absPatterns := [][]byte{
		[]byte("/usr/bin/git"),
		[]byte("/bin/git"),
		[]byte("/opt/homebrew/bin/git"),
		[]byte("\\\\usr\\\\bin\\\\git"),
		[]byte("C:\\\\Program Files\\\\Git\\\\cmd\\\\git.exe"),
	}
	for _, p := range absPatterns {
		if bytes.Contains(b, p) {
			foundAbs = true
			break
		}
	}
	return foundShim, foundAbs
}

func readTailBytes(path string, maxBytes int64) ([]byte, int64, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	size := st.Size()
	start := int64(0)
	if maxBytes > 0 && size > maxBytes {
		start = size - maxBytes
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, 0, err
		}
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	return b, int64(len(b)), nil
}

type fileInfo struct {
	path  string
	mtime time.Time
}

func listFilesBySuffixRecursive(root, suffix string) []string {
	return listFilesMatchingRecursive(root, func(name string) bool { return strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) })
}

func listFilesAllRecursive(root string) []string {
	return listFilesMatchingRecursive(root, func(string) bool { return true })
}

func listFilesMatching(root string, match func(name string) bool) []string {
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	infos := make([]fileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !match(e.Name()) {
			continue
		}
		full := filepath.Join(root, e.Name())
		st, err := e.Info()
		if err != nil {
			continue
		}
		infos = append(infos, fileInfo{path: full, mtime: st.ModTime()})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].mtime.After(infos[j].mtime) })

	out := make([]string, 0, len(infos))
	for _, fi := range infos {
		out = append(out, fi.path)
	}
	return out
}

func listFilesMatchingRecursive(root string, match func(name string) bool) []string {
	if root == "" {
		return nil
	}

	if _, statErr := os.Stat(root); statErr != nil {
		return nil
	}

	infos := make([]fileInfo, 0, 32)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && d != nil && !d.IsDir() && match(d.Name()) {
			if st, infoErr := d.Info(); infoErr == nil {
				infos = append(infos, fileInfo{path: path, mtime: st.ModTime()})
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].mtime.After(infos[j].mtime) })

	out := make([]string, 0, len(infos))
	for _, fi := range infos {
		out = append(out, fi.path)
	}
	return out
}

func takeMostRecent(paths []string, n int) []string {
	if n <= 0 || len(paths) == 0 {
		return nil
	}
	if len(paths) <= n {
		return append([]string(nil), paths...)
	}
	return append([]string(nil), paths[:n]...)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func homePath(path, home string) string {
	home = filepath.Clean(home)
	path = filepath.Clean(path)
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(os.PathSeparator) + strings.TrimPrefix(path, prefix)
	}
	// If someone passed a symlinked home dir but logs resolve a different form, keep best-effort.
	if strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(prefix)) {
		return "~/" + strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(prefix))
	}
	return path
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s Summary) String() string {
	shim := "no"
	if s.FoundShim() {
		shim = "yes"
	}
	abs := "no"
	if s.FoundAbs() {
		abs = "yes"
	}
	return fmt.Sprintf("%s: files=%d shim_refs=%s abs_git_refs=%s", s.Tool, s.FilesScanned, shim, abs)
}
