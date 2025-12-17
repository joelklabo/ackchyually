package execx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrependToPATH_EmptyDir_NoChange(t *testing.T) {
	in := joinPathList("a", "b")
	if got := PrependToPATH("", in); got != in {
		t.Fatalf("got %q, want %q", got, in)
	}
}

func TestPrependToPATH_EmptyPATH(t *testing.T) {
	shim := filepath.Join("tmp", "shims")
	if got := PrependToPATH(shim, ""); got != shim {
		t.Fatalf("got %q, want %q", got, shim)
	}
}

func TestPrependToPATH_AlreadyFirst(t *testing.T) {
	shim := filepath.Join("tmp", "shims")
	a := filepath.Join("usr", "bin")
	in := joinPathList(shim, a)
	if got := PrependToPATH(shim, in); got != in {
		t.Fatalf("got %q, want %q", got, in)
	}
}

func TestPrependToPATH_MovesExistingToFront(t *testing.T) {
	shim := filepath.Join("tmp", "shims")
	a := filepath.Join("usr", "bin")
	b := filepath.Join("opt", "bin")
	in := joinPathList(a, shim, b)
	want := joinPathList(shim, a, b)
	if got := PrependToPATH(shim, in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPrependToPATH_DedupesMultipleOccurrences(t *testing.T) {
	shim := filepath.Join("tmp", "shims")
	a := filepath.Join("usr", "bin")
	b := filepath.Join("opt", "bin")
	in := joinPathList(shim, a, shim, b, shim)
	want := joinPathList(shim, a, b)
	if got := PrependToPATH(shim, in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPrependToPATH_IgnoresEmptyEntries(t *testing.T) {
	shim := filepath.Join("tmp", "shims")
	a := filepath.Join("usr", "bin")
	in := strings.Join([]string{"", a, "", shim, ""}, string(os.PathListSeparator))
	want := joinPathList(shim, a)
	if got := PrependToPATH(shim, in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func joinPathList(parts ...string) string {
	return strings.Join(parts, string(os.PathListSeparator))
}
