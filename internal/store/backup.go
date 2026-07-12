package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BackupTo writes a consistent SQLite snapshot to dest using VACUUM INTO.
// Dest must not be the live database path; parent directories are created as needed.
func (s *Store) BackupTo(dest string) error {
	if dest == "" {
		return fmt.Errorf("backup destination path is required")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	if _, err := os.Stat(abs); err == nil {
		return fmt.Errorf("backup destination already exists: %s", abs)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat backup destination: %w", err)
	}

	q := "VACUUM INTO " + quoteSQLiteLiteral(abs)
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("VACUUM INTO: %w", err)
	}
	return nil
}

// RestoreDBFile copies a backup SQLite file onto destPath.
// Stop any process using destPath before calling (single-writer SQLite).
func RestoreDBFile(srcPath, destPath string) error {
	if srcPath == "" || destPath == "" {
		return fmt.Errorf("restore requires --input and --db (or GGHSTATS_DB)")
	}
	srcAbs, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}
	if srcAbs == destAbs {
		return fmt.Errorf("input and db path must differ")
	}
	in, err := os.Open(srcAbs)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	tmp := destAbs + ".restore-tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create temp restore file: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("copy backup: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temp restore file: %w", err)
	}
	// Best-effort remove WAL/SHM sidecars for the destination before replace.
	_ = os.Remove(destAbs + "-wal")
	_ = os.Remove(destAbs + "-shm")
	if err := os.Rename(tmp, destAbs); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace database: %w", err)
	}
	return nil
}

func quoteSQLiteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
