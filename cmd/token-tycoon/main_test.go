package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"token-tycoon/internal/ui"
)

func stubProgram(t *testing.T, drive func(m *ui.Model) *ui.Model) {
	t.Helper()
	prev := runProgram
	runProgram = func(m *ui.Model, _ ...tea.ProgramOption) (*ui.Model, error) { return drive(m), nil }
	t.Cleanup(func() { runProgram = prev })
}

func send(m *ui.Model, msgs ...tea.Msg) *ui.Model {
	for _, msg := range msgs {
		nm, _ := m.Update(msg)
		m = nm.(*ui.Model)
	}
	return m
}

func TestPlaySavesAndResumesWithAwayScreen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	base := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	prevNow := now
	t.Cleanup(func() { now = prevNow })
	now = func() time.Time { return base }
	var first string
	stubProgram(t, func(m *ui.Model) *ui.Model {
		m = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
		m.State().Money = 100
		m = send(m, tea.KeyPressMsg{Code: 'm', Text: "m"})
		first = ansi.Strip(m.View().Content)
		return m
	})
	var out, errb bytes.Buffer
	if code := run([]string{"--lang", "en", "--no-color"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(first, "TOKEN TYCOON") || !strings.Contains(first, "API Wrapper") || !strings.Contains(first, "TUTORIAL 1/7") {
		t.Fatal(first)
	}
	path := filepath.Join(dir, "token-tycoon", "save.json")
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Fatalf("save: %v", err)
	}
	now = func() time.Time { return base.Add(2 * time.Hour) }
	var second string
	stubProgram(t, func(m *ui.Model) *ui.Model {
		m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
		second = ansi.Strip(m.View().Content)
		return m
	})
	if code := run([]string{"--lang", "pl"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(second, "GDY CIĘ NIE BYŁO") || !strings.Contains(second, "2h00m") || strings.Contains(second, "SAMOUCZEK") {
		t.Fatal(second)
	}
}

func TestPlayRecoversFromCorruptSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "token-tycoon", "save.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	var view string
	stubProgram(t, func(m *ui.Model) *ui.Model {
		m = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
		view = ansi.Strip(m.View().Content)
		return m
	})
	var out, errb bytes.Buffer
	if code := run([]string{"--lang", "en"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if _, err := os.Stat(path + ".corrupt"); err != nil {
		t.Fatal("corrupt file not kept")
	}
	if !strings.Contains(view, "New game started") {
		t.Fatal(view)
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
