package agentclis

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

// Schema:
//
// This manifest is consumed by `ackchyually integrate status` to decide whether an
// installed agent CLI version is within the claimed supported range.
//
//   - Versions are expressed as semver strings without a leading "v" (example: "1.2.3").
//   - The checker normalizes installed versions by extracting the first "x.y.z" (with
//     optional pre-release/build suffix) and prefixing with "v" for comparison.
//   - Ranges are: [min_inclusive, max_exclusive). An empty max_exclusive means "no max".
//
// Keep this file stable: it is intended to be editable without changing Go code.
//
//go:embed supported_versions.json
var supportedVersionsJSON []byte

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	Tools         []Tool `json:"tools"`
}

type Tool struct {
	ID               string   `json:"id"`
	Binary           string   `json:"binary"`
	NPMPackage       string   `json:"npm_package"`
	SupportedRange   Range    `json:"supported_range"`
	CITestedVersions []string `json:"ci_tested_versions"`
	DocsURL          string   `json:"docs_url"`
}

type Range struct {
	MinInclusive string `json:"min_inclusive"`
	MaxExclusive string `json:"max_exclusive"`
}

func LoadManifest() (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(supportedVersionsJSON, &m); err != nil {
		return Manifest{}, err
	}
	if m.SchemaVersion != 1 {
		return Manifest{}, fmt.Errorf("supported_versions: unsupported schema_version %d", m.SchemaVersion)
	}
	return m, nil
}

func (m Manifest) ToolByID(id string) (Tool, bool) {
	for _, t := range m.Tools {
		if t.ID == id {
			return t, true
		}
	}
	return Tool{}, false
}

func (t Tool) CheckInstalledVersion(installedVersion string) (CheckResult, error) {
	norm, ok := NormalizeInstalledVersion(installedVersion)
	if !ok {
		return CheckResult{
			InstalledVersion: installedVersion,
			Normalized:       "",
			WithinRange:      false,
			Parseable:        false,
		}, nil
	}

	minV, err := normalizeManifestVersion(t.SupportedRange.MinInclusive)
	if err != nil {
		return CheckResult{}, err
	}
	maxV, err := normalizeManifestVersion(t.SupportedRange.MaxExclusive)
	if err != nil {
		return CheckResult{}, err
	}

	within := (minV == "" || semver.Compare(norm, minV) >= 0) && (maxV == "" || semver.Compare(norm, maxV) < 0)

	return CheckResult{
		InstalledVersion: installedVersion,
		Normalized:       norm,
		WithinRange:      within,
		Parseable:        true,
	}, nil
}

type CheckResult struct {
	InstalledVersion string
	Normalized       string
	WithinRange      bool
	Parseable        bool
}

var semverRe = regexp.MustCompile(`\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?`)

func NormalizeInstalledVersion(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}

	// Prefer already-normalized semver.
	if strings.HasPrefix(s, "v") && semver.IsValid(s) {
		return s, true
	}

	match := semverRe.FindString(s)
	if match == "" {
		return "", false
	}

	v := "v" + match
	if !semver.IsValid(v) {
		return "", false
	}
	return v, true
}

func normalizeManifestVersion(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	v := s
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "", errors.New("supported_versions: invalid semver: " + s)
	}
	return v, nil
}
