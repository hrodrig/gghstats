#!/bin/sh
# Load local env and start gghstats serve (launchd / manual background use).
#
# Default env file: ~/.gghstats.env (copy from contrib/gghstats.env.example and edit).
# Override: GGHSTATS_ENV_FILE=/path/to/envfile gghstats-serve.sh

set -eu

ENV_FILE="${GGHSTATS_ENV_FILE:-$HOME/.gghstats.env}"
if [ -f "$ENV_FILE" ]; then
	set -a
	# shellcheck disable=SC1090
	. "$ENV_FILE"
	set +a
fi

exec gghstats serve "$@"
