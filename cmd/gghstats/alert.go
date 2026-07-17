package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hrodrig/gghstats/internal/alert"
)

// exitError carries a process exit code (groot/kzero style: 4 = notify delivery failed).
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("exit %d", e.code)
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error { return e.err }

func exitCodeOf(err error) int {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return 1
}

func runAlert(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gghstats alert test [--kind traffic|ops] [--sink slack|webhook|loki]")
	}
	switch args[0] {
	case "test":
		return runAlertTest(args[1:])
	case "--help", "-h", "help":
		fmt.Fprint(os.Stdout, `Usage: gghstats alert test [--kind traffic|ops] [--sink TYPE]

Send a synthetic alert to configured sinks (GGHSTATS_ALERT_SINKS).
Does not start serve or run sync. See SPEC §8.8.

`)
		return nil
	default:
		return fmt.Errorf("unknown alert subcommand %q (want: test)", args[0])
	}
}

func runAlertTest(args []string) error {
	fs := flag.NewFlagSet("alert test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	kind := fs.String("kind", alert.KindTraffic, "payload kind: traffic or ops")
	sinkFilter := fs.String("sink", "", "optional sink type filter: slack, webhook, or loki")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sinks, err := alert.SinksForTest(os.Getenv)
	if err != nil {
		return err
	}
	sinks, err = alert.FilterSinks(sinks, *sinkFilter)
	if err != nil {
		return err
	}

	p, err := alert.SyntheticPayload(*kind)
	if err != nil {
		return err
	}

	senders := alert.BuildSenders(sinks, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := alert.FanOut(ctx, senders, p); err != nil {
		return &exitError{code: 4, err: fmt.Errorf("alert test: %w", err)}
	}
	fmt.Fprintf(os.Stdout, "alert test: sent kind %q to %d sink(s)\n", p.Kind, len(senders))
	return nil
}
