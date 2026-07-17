PORTNAME=	gghstats
PORTVERSION=	0.10.0
DISTVERSIONPREFIX=	v
CATEGORIES=	sysutils
# Local test before release: copy the tarball into DISTDIR (see `make -V DISTDIR`
# from this port directory) using the same name as DISTFILES, or run e.g.
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local distfile".
MASTER_SITES=	https://github.com/hrodrig/gghstats/releases/download/v${PORTVERSION}/
DISTFILES=	gghstats_${PORTVERSION}_freebsd_${ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX=	.tar.gz

MAINTAINER=	hrodrig@usb.ve
COMMENT=	Self-hosted GitHub traffic dashboard and CLI with SQLite storage
WWW=		https://github.com/hrodrig/gghstats
LICENSE=	MIT

NO_BUILD=	yes
# amd64/aarch64 only: matches GitHub release tarballs (see README.md "Supported architectures").
ONLY_FOR_ARCHS=	amd64 aarch64

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/gghstats ${STAGEDIR}${PREFIX}/bin/gghstats
	${MKDIR} ${STAGEDIR}${PREFIX}/share/man/man1
	${INSTALL_MAN} ${WRKDIR}/share/man/man1/gghstats.1 ${STAGEDIR}${PREFIX}/share/man/man1/
	${MKDIR} ${STAGEDIR}${PREFIX}/share/doc/gghstats
	${INSTALL_DATA} ${WRKDIR}/share/doc/gghstats/LICENSE ${STAGEDIR}${PREFIX}/share/doc/gghstats/
	${MKDIR} ${STAGEDIR}${PREFIX}/etc/gghstats
	${INSTALL_DATA} ${WRKDIR}/etc/gghstats/gghstats.env.example ${STAGEDIR}${PREFIX}/etc/gghstats/
	${MKDIR} ${STAGEDIR}${PREFIX}/etc/rc.d
	${INSTALL_SCRIPT} ${.CURDIR}/rc.d/gghstats ${STAGEDIR}${PREFIX}/etc/rc.d/gghstats

.include <bsd.port.mk>
