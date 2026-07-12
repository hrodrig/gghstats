package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hrodrig/gghstats/internal/store"
)

func runBackup(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	var dbPath, output string
	fs.StringVar(&dbPath, "db", envOr("GGHSTATS_DB", defaultDBPath()), "SQLite database path")
	fs.StringVar(&output, "output", "", "destination backup file path (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if output == "" {
		return fmt.Errorf("--output is required")
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.BackupTo(output); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Backup written to %s\n", output)
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	var dbPath, input string
	fs.StringVar(&dbPath, "db", envOr("GGHSTATS_DB", defaultDBPath()), "SQLite database path to overwrite")
	fs.StringVar(&input, "input", "", "backup file to restore from (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	if err := store.RestoreDBFile(input, dbPath); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Restored %s -> %s (stop serve before overwrite if the DB is in use)\n", input, dbPath)
	return nil
}
