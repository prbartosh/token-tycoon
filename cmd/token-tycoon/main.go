// Command token-tycoon is a terminal idle tycoon about the AI industry.
// At milestone M0 it ships the balance simulator; the UI arrives in M1.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/simulate"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	// The language is needed before parsing, for the translated flag help.
	cat := i18n.New(i18n.Detect(langArg(args), ""))
	fs := flag.NewFlagSet("token-tycoon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	simDur := fs.String("simulate", "", cat.T("cli.flag.simulate"))
	strategy := fs.String("strategy", simulate.Greedy, cat.T("cli.flag.strategy"))
	seed := fs.Uint64("seed", 42, cat.T("cli.flag.seed"))
	out := fs.String("out", "", cat.T("cli.flag.out"))
	sessions := fs.String("sessions", "10m/day", cat.T("cli.flag.sessions"))
	combo := fs.Float64("combo-rate", 0, cat.T("cli.flag.combo_rate"))
	latency := fs.Float64("click-latency", 0.5, cat.T("cli.flag.click_latency"))
	balPath := fs.String("balance", "", cat.T("cli.flag.balance"))
	fs.String("lang", "", cat.T("cli.flag.lang"))
	ci := fs.Bool("ci", false, cat.T("cli.flag.ci"))
	fs.Usage = func() {
		fmt.Fprintln(stderr, cat.T("cli.usage"))
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var b *balance.Balance
	var err error
	if *balPath != "" {
		if _, statErr := os.Stat(*balPath); statErr != nil {
			fmt.Fprintln(stderr, cat.T("cli.balance_error", statErr))
			return 2
		}
		b, err = balance.LoadWithOverride(*balPath)
	} else {
		b, err = balance.Load()
	}
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.balance_error", err))
		return 2
	}

	if *simDur == "" {
		fmt.Fprintln(stdout, cat.T("cli.ui_in_m1"))
		return 0
	}
	d, err := simulate.ParseDuration(*simDur)
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.bad_duration", *simDur))
		return 2
	}
	sess, err := simulate.ParseSessions(*sessions)
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.bad_sessions", *sessions))
		return 2
	}
	switch *strategy {
	case simulate.Greedy, simulate.Passive, simulate.Random:
	default:
		fmt.Fprintln(stderr, cat.T("cli.bad_strategy", *strategy))
		return 2
	}
	res, err := simulate.Run(b, simulate.Config{
		Duration: d, Strategy: *strategy, Seed: *seed, SessionS: sess,
		ComboRate: *combo, ClickLatencyS: *latency,
	})
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.error", err))
		return 2
	}
	fails := simulate.Report(stdout, res, b, cat)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(stderr, cat.T("cli.error", err))
			return 2
		}
		if err := simulate.WriteCSV(f, res, b); err != nil {
			f.Close()
			fmt.Fprintln(stderr, cat.T("cli.error", err))
			return 2
		}
		if err := f.Close(); err != nil {
			fmt.Fprintln(stderr, cat.T("cli.error", err))
			return 2
		}
		fmt.Fprintln(stdout, cat.T("cli.csv_written", *out))
	}
	if *ci && fails > 0 {
		return 1
	}
	return 0
}

// langArg returns the value of -lang/--lang in args ("" if absent), in any of
// the forms the flag package accepts: -lang x, --lang x, -lang=x, --lang=x.
func langArg(args []string) string {
	for i, a := range args {
		if a == "--" {
			break
		}
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if !strings.HasPrefix(a, "-") || name != "lang" {
			continue
		}
		if hasVal {
			return val
		}
		if i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
