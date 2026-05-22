#!/bin/sh
# postrm: on purge, remove config directory (secrets in gghstats.env). Data under /var/lib/gghstats is left intact.
case "$1" in
    purge)
        rm -rf /etc/gghstats
        ;;
esac
exit 0
