package app

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAgentCLIHint_NonTTY_NoOutput(t *testing.T) {
	setTempHomeAndCWD(t)

	tmp := t.TempDir()
	writeExec(t, tmp, "codex",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 1.2.3; exit 0; fi\nexit 0\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo 1.2.3\r\n  exit /b 0\r\n)\r\nexit /b 0\r\n",
	)
	t.Setenv("PATH", tmp)

	_, _, errOut := captureStdoutStderr(t, func() int {
		maybePrintAgentCLIHint(time.Unix(1000, 0))
		return 0
	})

	if strings.Contains(errOut, "ackchyually: tip:") {
		t.Fatalf("unexpected hint in non-TTY:\n%s", errOut)
	}

	if _, err := os.Stat(agentCLIHintStatePath()); err == nil {
		t.Fatalf("hint state should not be written in non-TTY")
	}
}

func TestAgentCLIHint_TTY_RateLimited(t *testing.T) {
	setTempHomeAndCWD(t)

	tmp := t.TempDir()
	writeExec(t, tmp, "codex",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 1.2.3; exit 0; fi\nexit 0\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo 1.2.3\r\n  exit /b 0\r\n)\r\nexit /b 0\r\n",
	)
	t.Setenv("PATH", tmp)

	t0 := time.Unix(1000, 0)

	var out1 bytes.Buffer
	maybePrintAgentCLIHintImpl(context.Background(), &out1, t0)
	if !strings.Contains(out1.String(), "ackchyually integrate all") {
		t.Fatalf("expected hint output, got:\n%s", out1.String())
	}

	var out2 bytes.Buffer
	maybePrintAgentCLIHintImpl(context.Background(), &out2, t0.Add(2*time.Hour))
	if out2.String() != "" {
		t.Fatalf("expected rate-limited (no hint), got:\n%s", out2.String())
	}

	var out3 bytes.Buffer
	maybePrintAgentCLIHintImpl(context.Background(), &out3, t0.Add(25*time.Hour))
	if !strings.Contains(out3.String(), "ackchyually integrate all") {
		t.Fatalf("expected hint output after interval, got:\n%s", out3.String())
	}
}
