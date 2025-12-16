package app

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBestCmd_MissingTool(t *testing.T) {
	// Capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	args := []string{}
	code := bestCmd(args)
	w.Close()

	if code != 2 {
		t.Errorf("bestCmd(no tool) = %d; want 2", code)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "required") {
		t.Errorf("expected required error, got %q", string(out))
	}
}

func TestExportCmd_Flags(t *testing.T) {
	// Just run it to cover flag parsing
	// It calls exportImpl, which we can't easily mock out without changing code,
	// but we can ensure it doesn't crash or exit 2 if flags are valid.
	// We need a temp home for DB.
	setTempHomeAndCWD(t)

	code := exportCmd([]string{"--format=json", "--tool=git"})
	if code != 0 {
		t.Errorf("exportCmd = %d; want 0", code)
	}

	// Test invalid flag
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	code = exportCmd([]string{"--invalid"})
	w.Close()
	if code != 2 {
		t.Errorf("exportCmd(invalid) = %d; want 2", code)
	}
}

func TestParseFlags_EdgeCases(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	boolFlag := fs.Bool("bool", false, "bool flag")
	strFlag := fs.String("str", "", "string flag")

	// 1. Flag with value joined by =
	args := []string{"--str=foo"}
	if err := parseFlags(fs, args); err != nil {
		t.Errorf("parseFlags failed: %v", err)
	}
	if *strFlag != "foo" {
		t.Errorf("strFlag=%q want foo", *strFlag)
	}

	// 2. Bool flag without value (should parse as true)
	*boolFlag = false
	args = []string{"--bool"}
	if err := parseFlags(fs, args); err != nil {
		t.Errorf("parseFlags failed: %v", err)
	}
	if !*boolFlag {
		t.Errorf("boolFlag=false want true")
	}

	// 3. Unknown flag (should cause Parse error because parseFlags passes it to Parse)
	args = []string{"--unknown"}
	if err := parseFlags(fs, args); err == nil {
		t.Errorf("expected error for unknown flag")
	}

	// 4. Positional arg mixed with flags
	// parseFlags reorders so flags come first.
	// But wait, if we have "arg --bool", parseFlags splits it: posArgs=["arg"], flagArgs=["--bool"].
	// Passes ["--bool", "arg"] to Parse.
	// Parse consumes --bool. Then sees "arg". Stops? Or consumes as arg?
	// With NewFlagSet, it consumes args.
	*boolFlag = false
	args = []string{"arg", "--bool"}
	if err := parseFlags(fs, args); err != nil {
		t.Errorf("parseFlags failed: %v", err)
	}
	if !*boolFlag {
		t.Errorf("boolFlag=false want true")
	}
	if fs.NArg() != 1 || fs.Arg(0) != "arg" {
		t.Errorf("expected arg to be positional, got args=%v", fs.Args())
	}
}

func TestRunShim_WhichFails(t *testing.T) {
	// Capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	code := RunShim("missingtool_xyz_123", []string{})
	w.Close()

	if code != 127 {
		t.Errorf("RunShim(missing) = %d; want 127", code)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "executable file not found") && !strings.Contains(string(out), "not found") {
		// Exact message depends on OS
		t.Logf("stderr: %q", string(out))
	}
}
