package packaging

import (
	"os"
	"strings"
	"testing"
)

const (
	goreleaserManifest = "../../.goreleaser.yaml"
	systemdUnit        = "../../contrib/systemd/gghstats.service"

	// dataDirMode is the mode the packaged data directory ships with. 0755 (not
	// 0750) because the unit also supports a dedicated service user: a
	// root-owned 0750 directory would make that setup fail to write the
	// database.
	dataDirMode = "0755"
)

// TestPackagedDataDirMatchesUnit guards the invariant between the shipped
// systemd unit and the nfpm package contents: systemd does not create
// WorkingDirectory, so the package must ship it as an empty directory entry
// (type: dir) or the service fails to start on .deb/.rpm installs.
func TestPackagedDataDirMatchesUnit(t *testing.T) {
	workingDir := systemdWorkingDir(readFile(t, systemdUnit))
	if workingDir == "" {
		t.Fatalf("%s: no WorkingDirectory declared", systemdUnit)
	}

	entry, found := findDirEntry(readFile(t, goreleaserManifest), workingDir)
	if !found {
		t.Fatalf("%s: no contents entry for %s (the packaged service would fail to start)", goreleaserManifest, workingDir)
	}
	if !entry.hasTypeDir {
		t.Errorf("%s: contents entry for %s is missing 'type: dir'", goreleaserManifest, workingDir)
	}
	if entry.mode != dataDirMode {
		t.Errorf("%s: contents entry for %s has mode %q, want %q", goreleaserManifest, workingDir, entry.mode, dataDirMode)
	}
}

// systemdWorkingDir returns the WorkingDirectory declared by a systemd unit.
func systemdWorkingDir(unit string) string {
	for _, line := range strings.Split(unit, "\n") {
		rest, ok := strings.CutPrefix(line, "WorkingDirectory=")
		if ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// dirEntry describes the parts of one nfpm contents entry this package cares
// about.
type dirEntry struct {
	hasTypeDir bool
	mode       string
}

// findDirEntry returns the nfpm contents entry that ships dir as an empty
// directory. It scans the manifest textually instead of parsing YAML so the
// repository keeps its current dependency footprint.
func findDirEntry(manifest, dir string) (dirEntry, bool) {
	lines := strings.Split(manifest, "\n")
	want := "- dst: " + dir

	for i, line := range lines {
		if strings.TrimSpace(line) != want {
			continue
		}

		entry := dirEntry{}
		for _, next := range lines[i+1 : min(i+8, len(lines))] {
			trimmed := strings.TrimSpace(next)
			if strings.HasPrefix(trimmed, "- ") {
				break // next contents entry
			}
			switch {
			case trimmed == "type: dir":
				entry.hasTypeDir = true
			case strings.HasPrefix(trimmed, "mode:"):
				entry.mode = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "mode:")), `"`)
			}
		}
		return entry, true
	}
	return dirEntry{}, false
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
