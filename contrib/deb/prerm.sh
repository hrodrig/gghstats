#!/bin/sh
# prerm: stop and disable gghstats before package removal.
set -e
if command -v systemctl >/dev/null 2>&1; then
    systemctl stop gghstats.service 2>/dev/null || true
    systemctl disable gghstats.service 2>/dev/null || true
fi
