package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/term"
)

func main() {
	fmt.Println("PROMPTLY_START")
	fmt.Printf("TTY stdin=%v stdout=%v stderr=%v\n",
		term.IsTerminal(int(os.Stdin.Fd())),
		term.IsTerminal(int(os.Stdout.Fd())),
		term.IsTerminal(int(os.Stderr.Fd())),
	)

	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		cols, rows = 0, 0
	}
	fmt.Printf("SIZE_BEFORE rows=%d cols=%d\n", rows, cols)

	fmt.Print("PROMPT enter y: ")
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		line = ""
	}
	fmt.Printf("\nLINE %q\n", line)

	cols2, rows2, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		cols2, rows2 = 0, 0
	}
	fmt.Printf("SIZE_AFTER rows=%d cols=%d\n", rows2, cols2)

	fmt.Println("PROMPTLY_END")
}
