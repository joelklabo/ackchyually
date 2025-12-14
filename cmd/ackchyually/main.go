package main

import (
	"os"
	"path/filepath"

	"github.com/joelklabo/ackchyually/internal/app"
)

func main() {
	argv0 := filepath.Base(os.Args[0])

	// Busybox-style: invoked as ackchyually => CLI, invoked as something else => shim.
	if argv0 != "ackchyually" && argv0 != "ackchyually.exe" {
		os.Exit(app.RunShim(argv0, os.Args[1:]))
	}
	os.Exit(app.RunCLI(os.Args[1:]))
}
