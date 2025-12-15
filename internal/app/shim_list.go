package app

import (
	"fmt"
	"os"
	"sort"
)

func shimList(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: ackchyually shim list")
		return 2
	}

	dir := shimDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no shims installed)")
			return 0
		}
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name[0] == '.' {
			continue
		}
		if name == "ackchyually" || name == "ackchyually.exe" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Println("(no shims installed)")
		return 0
	}

	for _, name := range names {
		fmt.Println(name)
	}
	return 0
}
