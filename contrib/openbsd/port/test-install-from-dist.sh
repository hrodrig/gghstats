#!/bin/ksh
# Install gghstats like the OpenBSD port (do-install) without a full /usr/ports tree.
# Use on lab VMs that only have the port skeleton + local distfile.
#
#   doas sh test-install-from-dist.sh /tmp/gghstats_0.6.4_openbsd_amd64.tar.gz
#
# Optional second arg: directory containing files/gghstats, gghstats-serve, gghstats-start
#   (default: same directory as this script, then files/ subdir)

set -e

if [ "$(id -u)" -ne 0 ]; then
	echo "Run as root (doas sh $0 ...)" >&2
	exit 1
fi

dist="${1:?usage: $0 /path/to/gghstats_VERSION_openbsd_ARCH.tar.gz [port-files-dir]}"
portdir="${2:-$(dirname "$0")}"
files="$portdir/files"
[ -d "$files" ] || files="$portdir"

for f in "$files/gghstats" "$files/gghstats-serve" "$files/gghstats-start"; do
	[ -f "$f" ] || { echo "missing port file: $f" >&2; exit 1; }
done

stage="/tmp/gghstats-port-install-$$"
trap 'rm -rf "$stage"' EXIT INT TERM
mkdir -p "$stage"
echo "Extracting $dist ..."
tar xzf "$dist" -C "$stage"

PREFIX=/usr/local
install -d "$PREFIX/bin" "$PREFIX/man/man1" "$PREFIX/share/doc/gghstats" \
	"$PREFIX/share/examples/gghstats" "$PREFIX/etc/rc.d"

install -m 755 "$stage/gghstats" "$PREFIX/bin/gghstats"
install -m 755 "$files/gghstats-serve" "$PREFIX/bin/gghstats-serve"
install -m 755 "$files/gghstats-start" "$PREFIX/bin/gghstats-start"

if [ -f "$stage/share/man/man1/gghstats.1" ]; then
	install -m 644 "$stage/share/man/man1/gghstats.1" "$PREFIX/man/man1/gghstats.1"
fi
if [ -f "$stage/share/doc/gghstats/LICENSE" ]; then
	install -m 644 "$stage/share/doc/gghstats/LICENSE" "$PREFIX/share/doc/gghstats/LICENSE"
fi
if [ -f "$stage/etc/gghstats/gghstats.env.example" ]; then
	install -m 644 "$stage/etc/gghstats/gghstats.env.example" "$PREFIX/share/examples/gghstats/gghstats.env.example"
fi

install -m 555 "$files/gghstats" "$PREFIX/etc/rc.d/gghstats"

echo "Installed (port-equivalent):"
echo "  $PREFIX/bin/gghstats $PREFIX/bin/gghstats-serve $PREFIX/bin/gghstats-start"
echo "  $PREFIX/etc/rc.d/gghstats"
echo ""
echo "Next:"
echo "  mkdir -p /etc/gghstats /var/lib/gghstats"
echo "  cp $PREFIX/share/examples/gghstats/gghstats.env.example /etc/gghstats/gghstats.env"
echo "  vi /etc/gghstats/gghstats.env"
echo "  rcctl enable gghstats && rcctl start gghstats"
