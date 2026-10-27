// Package packaging holds release-packaging checks.
//
// The systemd unit shipped by the .deb/.rpm declares
// WorkingDirectory=/var/lib/gghstats. systemd does not create that directory,
// so the nfpm contents have to ship it explicitly as an empty directory entry
// (type: dir); otherwise the packaged service fails to start. The invariant
// between the unit and the package manifest is covered by the tests in this
// package.
package packaging
