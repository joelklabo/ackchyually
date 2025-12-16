package app

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// These are set at build time by goreleaser (ldflags). They intentionally have
// simple names so `-X` is easy to use.
var (
	buildVersion = "dev"
	buildCommit  = ""
	buildDate    = ""
)

type versionInfo struct {
	Version  string
	Commit   string
	Date     string
	Modified bool
}

func printVersion() {
	v := getVersionInfo()

	fmt.Println("ackchyually")
	fmt.Printf("  version: %s\n", v.Version)
	if v.Commit != "" {
		commit := shortSHA(v.Commit)
		if v.Modified {
			commit += " (dirty)"
		}
		fmt.Printf("  commit:  %s\n", commit)
	}
	if v.Date != "" {
		fmt.Printf("  built:   %s\n", v.Date)
	}
}

func getVersionInfo() versionInfo {
	return getVersionInfoWithReader(debug.ReadBuildInfo)
}

func getVersionInfoWithReader(readBuildInfo func() (*debug.BuildInfo, bool)) versionInfo {
	v := versionInfo{
		Version: normalizeVersion(buildVersion),
		Commit:  strings.TrimSpace(buildCommit),
		Date:    strings.TrimSpace(buildDate),
	}

	if bi, ok := readBuildInfo(); ok {
		// If this is a module build (not a local "devel" build), use its version
		// unless we already have an explicit buildVersion.
		if (v.Version == "" || v.Version == "dev") && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v.Version = normalizeVersion(bi.Main.Version)
		}

		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if v.Commit == "" {
					v.Commit = strings.TrimSpace(s.Value)
				}
			case "vcs.time":
				if v.Date == "" {
					v.Date = strings.TrimSpace(s.Value)
				}
			case "vcs.modified":
				v.Modified = strings.EqualFold(strings.TrimSpace(s.Value), "true")
			}
		}
	}

	if v.Version == "" {
		v.Version = "dev"
	}
	return v
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if v == "dev" {
		return v
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}
