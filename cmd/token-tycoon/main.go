// Command token-tycoon is a terminal idle tycoon about the AI industry.
// Without --simulate it starts the game UI; with it, the balance simulator.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/save"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/simulate"
	"github.com/prbartosh/token-tycoon/internal/ui"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
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
	lang := fs.String("lang", "", cat.T("cli.flag.lang"))
	ci := fs.Bool("ci", false, cat.T("cli.flag.ci"))
	noColor := fs.Bool("no-color", false, cat.T("cli.flag.no_color"))
	debug := fs.Bool("debug", false, cat.T("cli.flag.debug"))
	showVersion := fs.Bool("version", false, cat.T("cli.flag.version"))
	fs.Usage = func() {
		fmt.Fprintln(stderr, cat.T("cli.usage"))
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "token-tycoon", version)
		return 0
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
		return play(b, playOpts{lang: *lang, noColor: *noColor, debug: *debug}, stdout, stderr)
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

type playOpts struct {
	lang    string
	noColor bool
	debug   bool
}

var runProgram = func(m *ui.Model, opts ...tea.ProgramOption) (*ui.Model, error) {
	final, err := tea.NewProgram(m, opts...).Run()
	if fm, ok := final.(*ui.Model); ok {
		m = fm
	}
	return m, err
}

var now = time.Now

func play(b *balance.Balance, o playOpts, stdout, stderr io.Writer) int {
	cat := i18n.New(i18n.Detect(o.lang, ""))
	path, err := save.DefaultPath()
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.error", err))
		return 1
	}
	st, elapsed, notice, err := loadGame(path, b)
	if err != nil {
		fmt.Fprintln(stderr, cat.T("cli.error", err))
		return 1
	}
	if l, ok := i18n.Parse(o.lang); ok {
		st.Settings.Lang = l.String()
	}
	cat = i18n.New(i18n.Detect(o.lang, st.Settings.Lang))
	noticeText := ""
	if notice.key != "" {
		noticeText = cat.T(notice.key, notice.arg)
	}
	opts := save.Options{Backups: b.Engine.Backups, Indent: o.debug}
	write := func(s *sim.State) error { return save.Save(path, s, b, now(), opts) }
	profile := colorprofile.Detect(stdout, os.Environ())
	if o.noColor {
		profile = colorprofile.ASCII
	}
	m := ui.New(ui.Config{
		Balance: b, State: st, Catalog: cat, Profile: profile, Debug: o.debug,
		Save: write, OfflineElapsed: elapsed, Notice: noticeText, NewGame: notice.fresh, Now: now,
	})
	popts := []tea.ProgramOption{tea.WithOutput(stdout), tea.WithFPS(b.Engine.FPSAnimation)}
	if o.noColor {
		popts = append(popts, tea.WithColorProfile(colorprofile.ASCII))
	}
	final, runErr := runProgram(m, popts...)
	code := 0
	if runErr != nil && !errors.Is(runErr, tea.ErrInterrupted) {
		fmt.Fprintln(stderr, cat.T("cli.error", runErr))
		code = 1
	}
	if err := write(final.State()); err != nil {
		fmt.Fprintln(stderr, cat.T("ui.msg.save_failed", err))
		return 1
	}
	return code
}

type notice struct {
	key, arg string
	fresh    bool
}

func loadGame(path string, b *balance.Balance) (*sim.State, float64, notice, error) {
	l, err := save.Load(path, b, b.Engine.Backups)
	switch {
	case err == nil:
		var n notice
		if l.Recovered() {
			n = notice{key: "msg.save_recovered", arg: l.Source}
		}
		return l.State, max(0, now().Sub(l.SavedAt).Seconds()), n, nil
	case errors.Is(err, save.ErrNewerVersion):
		return nil, 0, notice{}, err
	case errors.Is(err, save.ErrNoSave):
		s := sim.NewState(b, uint64(now().UnixNano()))
		return &s, 0, notice{fresh: true}, nil
	}
	if rerr := os.Rename(path, path+".corrupt"); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		return nil, 0, notice{}, rerr
	}
	s := sim.NewState(b, uint64(now().UnixNano()))
	return &s, 0, notice{key: "ui.msg.save_corrupt", arg: path + ".corrupt"}, nil
}
