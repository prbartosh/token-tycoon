package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoSimulatePrintsM1(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errb bytes.Buffer
	if code := run([]string{"--lang", "en"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "M1") {
		t.Fatal(out.String())
	}
}

func TestSimulateWritesCSV(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	csv := filepath.Join(dir, "run.csv")
	var out, errb bytes.Buffer
	if code := run([]string{"--simulate", "5m", "--lang", "pl", "--out", csv}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "Raport symulatora") {
		t.Fatal(out.String())
	}
	if fi, err := os.Stat(csv); err != nil || fi.Size() == 0 {
		t.Fatalf("csv: %v", err)
	}
	if code := run([]string{"--simulate", "1w"}, &out, &errb); code != 2 {
		t.Fatalf("bad duration exit %d", code)
	}
}

func TestLangArg(t *testing.T) {
	cases := map[string][]string{
		"pl": {"--simulate", "1h", "--lang", "pl"},
		"en": {"-lang=en"},
		"":   {"--simulate", "1h"},
	}
	for want, args := range cases {
		if got := langArg(args); got != want {
			t.Errorf("langArg(%v) = %q, want %q", args, got, want)
		}
	}
	var out, errb bytes.Buffer
	if code := run([]string{"--lang", "pl", "-h"}, &out, &errb); code != 2 {
		t.Fatalf("-h exit %d", code)
	}
	if !strings.Contains(errb.String(), "Użycie") || !strings.Contains(errb.String(), "ziarno") {
		t.Fatalf("help not translated: %s", errb.String())
	}
}
