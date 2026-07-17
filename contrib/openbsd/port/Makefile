# OpenBSD port for gghstats
# Copy to /usr/ports/sysutils/gghstats/ and run: make makesum
# Submit: send diff to ports@openbsd.org
#
# From the gghstats repo root after bumping VERSION:
#   gmake port-openbsd-sync
#   gmake dist-openbsd
# See README.md and PORT-RELEASE.md.

COMMENT =	GitHub traffic dashboard and CLI (SQLite)
DISTNAME =	gghstats_0.10.0_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}
PKGNAME =	gghstats-0.10.0
CATEGORIES =	sysutils
HOMEPAGE =	https://github.com/hrodrig/gghstats
MAINTAINER =	hrodrig@usb.ve

# MIT
PERMIT_PACKAGE =	Yes
PERMIT_DISTFILES =	Yes

# Local test before release: copy tarball to DISTDIR or use
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local tarball".
MASTER_SITES =	https://github.com/hrodrig/gghstats/releases/download/v0.10.0/
DISTFILES =	gghstats_0.10.0_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX =	.tar.gz

ONLY_FOR_ARCHS =	amd64 arm64 aarch64
NO_BUILD =	Yes
NO_TEST =	Yes
PKG_ARCH =	*

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/gghstats ${PREFIX}/bin/gghstats
	${INSTALL_SCRIPT} ${FILESDIR}/gghstats-serve ${PREFIX}/bin/gghstats-serve
	${INSTALL_SCRIPT} ${FILESDIR}/gghstats-start ${PREFIX}/bin/gghstats-start
	${INSTALL_MAN} ${WRKDIR}/share/man/man1/gghstats.1 ${PREFIX}/man/man1/gghstats.1
	${INSTALL_DATA_DIR} ${PREFIX}/share/doc/gghstats
	${INSTALL_DATA} ${WRKDIR}/share/doc/gghstats/LICENSE ${PREFIX}/share/doc/gghstats/
	${INSTALL_DATA_DIR} ${PREFIX}/share/examples/gghstats
	${INSTALL_DATA} ${WRKDIR}/etc/gghstats/gghstats.env.example ${PREFIX}/share/examples/gghstats/
# rc.d: pkg/gghstats.rc + @rcscript in PLIST — generate-readmes copies *.rc → ${RCDIR}/gghstats.

.include <bsd.port.mk>
