package app

import (
	"fmt"
	"os"
	"runtime"
)

func printPathInstructions(shimDir string) {
	if runtime.GOOS == "windows" {
		fmt.Printf("  $env:PATH = \"%s;\" + $env:PATH\n", shimDir)
		fmt.Println("  # run: ackchyually shim enable")
	} else {
		fmt.Printf("  export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
		fmt.Println("  # for future shells, add that line to your ~/.zshrc or ~/.bashrc")
		fmt.Println("  # or run: ackchyually shim enable")
	}
}
