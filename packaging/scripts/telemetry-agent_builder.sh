#!/bin/sh

shell_quote_string() {
    echo "$1" | sed -e 's,\([^a-zA-Z0-9/_.=-]\),\\\1,g'
}

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]
    The following options may be given :
        --builddir=DIR      Absolute path to the dir where all actions will be performed
        --get_sources       Source will be downloaded from github
        --build_src_rpm     If it is set - src rpm will be built
        --build_src_deb  If it is set - source deb package will be built
        --build_rpm         If it is set - rpm will be built
        --build_deb         If it is set - deb will be built
        --install_deps      Install build dependencies(root privilages are required)
        --branch            Branch for build
        --repo              Repo for build
        --version           Version to build

        --help) usage ;;
Example $0 --builddir=/tmp/percona-telemetry-agent --get_sources=1 --build_src_rpm=1 --build_rpm=1
EOF
    exit 1
}

append_arg_to_args() {
    args="$args "$(shell_quote_string "$1")
}

parse_arguments() {
    pick_args=
    if test "$1" = PICK-ARGS-FROM-ARGV; then
        pick_args=1
        shift
    fi

    for arg; do
        val=$(echo "$arg" | sed -e 's;^--[^=]*=;;')
        case "$arg" in
        --builddir=*) WORKDIR="$val" ;;
        --build_src_rpm=*) SRPM="$val" ;;
        --build_src_deb=*) SDEB="$val" ;;
        --build_rpm=*) RPM="$val" ;;
        --build_deb=*) DEB="$val" ;;
        --get_sources=*) SOURCE="$val" ;;
        --branch=*) BRANCH="$val" ;;
        --repo=*) REPO="$val" ;;
        --version=*) VERSION="$val" ;;
        --install_deps=*) INSTALL="$val" ;;
        --help) usage ;;
        *)
            if test -n "$pick_args"; then
                append_arg_to_args "$arg"
            fi
            ;;
        esac
    done
}

check_workdir() {
    if [ "x$WORKDIR" = "x$CURDIR" ]; then
        echo >&2 "Current directory cannot be used for building!"
        exit 1
    else
        if ! test -d "$WORKDIR"; then
            echo >&2 "$WORKDIR is not a directory."
            exit 1
        fi
    fi
    return
}

get_sources() {
    cd "${WORKDIR}"
    if [ "${SOURCE}" = 0 ]; then
        echo "Sources will not be downloaded"
        return 0
    fi
    PRODUCT=percona-telemetry-agent
    PRODUCT_FULL=${PRODUCT}-${VERSION}
    echo "PRODUCT=${PRODUCT}" >percona-telemetry-agent.properties
    echo "BUILD_NUMBER=${BUILD_NUMBER}" >>percona-telemetry-agent.properties
    echo "BUILD_ID=${BUILD_ID}" >>percona-telemetry-agent.properties
    echo "VERSION=${VERSION}" >>percona-telemetry-agent.properties
    echo "BRANCH=${BRANCH}" >>percona-telemetry-agent.properties
    git clone "$REPO" ${PRODUCT}
    retval=$?
    if [ $retval != 0 ]; then
        echo "There were some issues during repo cloning from github. Please retry one more time"
        exit 1
    fi
    cd percona-telemetry-agent
    if [ ! -z "$BRANCH" ]; then
        git reset --hard
        git clean -xdf
        git checkout "$BRANCH"
    fi
    REVISION=$(git rev-parse --short HEAD)
    GITCOMMIT=$(git rev-parse HEAD 2>/dev/null)
    GITBRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
    echo "VERSION=${VERSION}" >VERSION
    echo "REVISION=${REVISION}" >>VERSION
    echo "GITCOMMIT=${GITCOMMIT}" >>VERSION
    echo "GITBRANCH=${GITBRANCH}" >>VERSION
    echo "REVISION=${REVISION}" >>${WORKDIR}/percona-telemetry-agent.properties
    rm -fr debian rpm
    echo "percona-telemetry-agent (${VERSION}-${RELEASE}) unstable; urgency=low" >> packaging/debian/changelog
    echo "  * Initial Release." >> packaging/debian/changelog
    echo " -- SurabhiBhat <surabhi.bhat@percona.com> $(date -R)" >> packaging/debian/changelog
    cp packaging/conf/percona-telemetry-agent.logrotate packaging/debian/
    cd ${WORKDIR}

    mv percona-telemetry-agent ${PRODUCT}-${VERSION}
    tar --owner=0 --group=0 -czf ${PRODUCT}-${VERSION}.tar.gz ${PRODUCT}-${VERSION}
    echo "UPLOAD=UPLOAD/experimental/BUILDS/${PRODUCT}/${PRODUCT}-${VERSION}/${BRANCH}/${REVISION}/${BUILD_ID}" >>percona-telemetry-agent.properties
    mkdir $WORKDIR/source_tarball
    mkdir $CURDIR/source_tarball
    cp ${PRODUCT}-${VERSION}.tar.gz $WORKDIR/source_tarball
    cp ${PRODUCT}-${VERSION}.tar.gz $CURDIR/source_tarball
    cd $CURDIR
    rm -rf percona-telemetry-agent
    return
}

get_system() {
    if [ -f /etc/redhat-release ]; then
        RHEL=$(rpm --eval %rhel)
        ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')
        OS_NAME="el$RHEL"
        OS="rpm"
    else
        ARCH=$(uname -m)
        OS_NAME="$(lsb_release -sc)"
        OS="deb"
    fi
    return
}

install_golang() {
    wget https://golang.org/dl/go1.21.1.linux-amd64.tar.gz -O /tmp/golang1.21.1.tar.gz
    tar --transform=s,go,go1.21.1, -zxf /tmp/golang1.21.1.tar.gz
    rm -rf /usr/local/go*
    mv go1.21.1 /usr/local/
    ln -s /usr/local/go1.21.1 /usr/local/go
}

install_deps() {
    if [ $INSTALL = 0 ]; then
        echo "Dependencies will not be installed"
        return
    fi
    if [ ! $(id -u) -eq 0 ]; then
        echo "It is not possible to install dependencies. Please run as root"
        exit 1
    fi
    CURPLACE=$(pwd)

    if [ "x$OS" = "xrpm" ]; then
        RHEL=$(rpm --eval %rhel)
        yum clean all
        yum -y install epel-release git wget
        yum -y install rpm-build make rpmlint rpmdevtools golang
        install_golang
    else
        until apt-get update; do
            sleep 1
            echo "waiting"
        done
        DEBIAN_FRONTEND=noninteractive apt-get -y install lsb-release
        export DEBIAN=$(lsb_release -sc)
        export ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')
        INSTALL_LIST="wget devscripts debhelper debconf pkg-config curl make golang git"
        until DEBIAN_FRONTEND=noninteractive apt-get -y install ${INSTALL_LIST}; do
            sleep 1
            echo "waiting"
        done
        install_golang
    fi
    return
}

get_tar() {
    TARBALL=$1
    TARFILE=$(basename $(find $WORKDIR/$TARBALL -name 'percona-telemetry-agent*.tar.gz' | sort | tail -n1))
    if [ -z $TARFILE ]; then
        TARFILE=$(basename $(find $CURDIR/$TARBALL -name 'percona-telemetry-agent*.tar.gz' | sort | tail -n1))
        if [ -z $TARFILE ]; then
            echo "There is no $TARBALL for build"
            exit 1
        else
            cp $CURDIR/$TARBALL/$TARFILE $WORKDIR/$TARFILE
        fi
    else
        cp $WORKDIR/$TARBALL/$TARFILE $WORKDIR/$TARFILE
    fi
    return
}

get_deb_sources() {
    param=$1
    echo $param
    FILE=$(basename $(find $WORKDIR/source_deb -name "percona-telemetry-agent*.$param" | sort | tail -n1))
    if [ -z $FILE ]; then
        FILE=$(basename $(find $CURDIR/source_deb -name "percona-telemetry-agent*.$param" | sort | tail -n1))
        if [ -z $FILE ]; then
            echo "There is no sources for build"
            exit 1
        else
            cp $CURDIR/source_deb/$FILE $WORKDIR/
        fi
    else
        cp $WORKDIR/source_deb/$FILE $WORKDIR/
    fi
    return
}

build_srpm() {
    if [ $SRPM = 0 ]; then
        echo "SRC RPM will not be created"
        return
    fi
    if [ "x$OS" = "xdeb" ]; then
        echo "It is not possible to build src rpm here"
        exit 1
    fi
    cd $WORKDIR
    get_tar "source_tarball"
    rm -fr rpmbuild
    ls | grep -v tar.gz | xargs rm -rf
    TARFILE=$(find . -name 'percona-telemetry-agent*.tar.gz' | sort | tail -n1)
    SRC_DIR=${TARFILE%.tar.gz}
    mkdir -vp rpmbuild/{SOURCES,SPECS,BUILD,SRPMS,RPMS}
    tar vxzf ${WORKDIR}/${TARFILE} --wildcards '*/packaging' --strip=1
    tar vxzf ${WORKDIR}/${TARFILE} --wildcards '*/VERSION' --strip=1
    source VERSION
    #
    sed -e "s:@@VERSION@@:${VERSION}:g" \
        -e "s:@@RELEASE@@:${RELEASE}:g" \
        -e "s:@@REVISION@@:${REVISION}:g" \
        packaging/rpm/telemetry-agent.spec >rpmbuild/SPECS/telemetry-agent.spec
    mv -fv ${TARFILE} ${WORKDIR}/rpmbuild/SOURCES
    rpmbuild -bs --define "_topdir ${WORKDIR}/rpmbuild" --define "version ${VERSION}" --define "dist .generic" rpmbuild/SPECS/telemetry-agent.spec
    mkdir -p ${WORKDIR}/srpm
    mkdir -p ${CURDIR}/srpm
    cp rpmbuild/SRPMS/*.src.rpm ${CURDIR}/srpm
    cp rpmbuild/SRPMS/*.src.rpm ${WORKDIR}/srpm
    return
}

build_rpm() {
    if [ $RPM = 0 ]; then
        echo "RPM will not be created"
        return
    fi
    if [ "x$OS" = "xdeb" ]; then
        echo "It is not possible to build rpm here"
        exit 1
    fi
    SRC_RPM=$(basename $(find $WORKDIR/srpm -name 'percona-telemetry-agent*.src.rpm' | sort | tail -n1))
    if [ -z $SRC_RPM ]; then
        SRC_RPM=$(basename $(find $CURDIR/srpm -name 'percona-telemetry-agent*.src.rpm' | sort | tail -n1))
        if [ -z $SRC_RPM ]; then
            echo "There is no src rpm for build"
            echo "You can create it using key --build_src_rpm=1"
            exit 1
        else
            cp $CURDIR/srpm/$SRC_RPM $WORKDIR
        fi
    else
        cp $WORKDIR/srpm/$SRC_RPM $WORKDIR
    fi
    cd $WORKDIR
    rm -fr rpmbuild
    mkdir -vp rpmbuild/{SOURCES,SPECS,BUILD,SRPMS,RPMS}
    cp $SRC_RPM rpmbuild/SRPMS/

    RHEL=$(rpm --eval %rhel)
    ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')

    echo "RHEL=${RHEL}" >>percona-telemetry-agent.properties
    echo "ARCH=${ARCH}" >>percona-telemetry-agent.properties
    [[ ${PATH} == *"/usr/local/go/bin"* && -x /usr/local/go/bin/go ]] || export PATH=/usr/local/go/bin:${PATH}
    export GOROOT="/usr/local/go/"
    export GOPATH=$(pwd)/
    export PATH="/usr/local/go/bin:$PATH:$GOPATH"
    export GOBINPATH="/usr/local/go/bin"
    #fi
    rpmbuild --define "_topdir ${WORKDIR}/rpmbuild" --define "dist .$OS_NAME" --rebuild rpmbuild/SRPMS/$SRC_RPM

    return_code=$?
    if [ $return_code != 0 ]; then
        exit $return_code
    fi
    mkdir -p ${WORKDIR}/rpm
    mkdir -p ${CURDIR}/rpm
    cp rpmbuild/RPMS/*/*.rpm ${WORKDIR}/rpm
    cp rpmbuild/RPMS/*/*.rpm ${CURDIR}/rpm
}

build_source_deb() {
    if [ $SDEB = 0 ]; then
        echo "source deb package will not be created"
        return
    fi
    if [ "x$OS" = "xrmp" ]; then
        echo "It is not possible to build source deb here"
        exit 1
    fi
    rm -rf percona-telemetry-agent*
    get_tar "source_tarball"
    rm -f *.dsc *.orig.tar.gz *.changes
    #
    TARFILE=$(basename $(find . -name 'percona-telemetry-agent*.tar.gz' | sort | tail -n1))
    DEBIAN=$(lsb_release -sc)
    ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')
    tar zxf ${TARFILE}
    BUILDDIR=${TARFILE%.tar.gz}
    #
    rm -fr ${BUILDDIR}/debian
    cp -av ${BUILDDIR}/packaging/debian ${BUILDDIR}
    #
    mv ${TARFILE} ${PRODUCT}_${VERSION}.orig.tar.gz
    cd ${BUILDDIR}
    source VERSION
    cp -r packaging/debian ./
    sed -i "s:@@VERSION@@:${VERSION}:g" debian/rules
    sed -i "s:@@REVISION@@:${REVISION}:g" debian/rules
    sed -i "s:sysconfig:default:" packaging/conf/percona-telemetry-agent.service
    dch -D unstable --force-distribution -v "${VERSION}-${RELEASE}" "Update to new telemetry-agent version ${VERSION}"
    dpkg-buildpackage -S
    cd ../
    mkdir -p $WORKDIR/source_deb
    mkdir -p $CURDIR/source_deb
    cp *_source.changes $WORKDIR/source_deb
    cp *.dsc $WORKDIR/source_deb
    cp *.orig.tar.gz $WORKDIR/source_deb
    cp *.diff.gz $WORKDIR/source_deb
    cp *_source.changes $CURDIR/source_deb
    cp *.dsc $CURDIR/source_deb
    cp *.orig.tar.gz $CURDIR/source_deb
    cp *.diff.gz $CURDIR/source_deb
}

build_deb() {
    if [ $DEB = 0 ]; then
        echo "Binary deb package will not be created"
        return
    fi
    if [ "x$OS" = "xrmp" ]; then
        echo "It is not possible to build binary deb here"
        exit 1
    fi
    for file in 'dsc' 'orig.tar.gz' 'changes' 'diff.gz'; do
        get_deb_sources $file
    done
    cd $WORKDIR
    rm -fv *.deb
    #
    export DEBIAN=$(lsb_release -sc)
    export ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')
    #
    echo "DEBIAN=${DEBIAN}" >>percona-telemetry-agent.properties
    echo "ARCH=${ARCH}" >>percona-telemetry-agent.properties

    #
    DSC=$(basename $(find . -name '*.dsc' | sort | tail -n1))
    #
    dpkg-source -x ${DSC}
    #
    cd ${PRODUCT}-${VERSION}
    source VERSION

    dch -m -D "${DEBIAN}" --force-distribution -v "${VERSION}-${RELEASE}.${DEBIAN}" 'Update distribution'

    export PATH=/usr/local/go/bin:${PATH}
    export GOROOT="/usr/local/go/"
    export GOPATH=$(pwd)/build
    export PATH="/usr/local/go/bin:$PATH:$GOPATH"
    export GO_BUILD_LDFLAGS="-w -s -X main.version=${VERSION} -X main.commit=${REVISION}"
    export GOBINPATH="/usr/local/go/bin"

    dpkg-buildpackage -rfakeroot -us -uc -b
    mkdir -p $CURDIR/deb
    mkdir -p $WORKDIR/deb
    cp $WORKDIR/*.deb $WORKDIR/deb
    cp $WORKDIR/*.deb $CURDIR/deb
}

CURDIR=$(pwd)
VERSION_FILE=$CURDIR/percona-telemetry-agent.properties
args=
WORKDIR=
SRPM=0
SDEB=0
RPM=0
DEB=0
SOURCE=0
TARBALL=0
OS_NAME=
ARCH=
OS=
INSTALL=0
RPM_RELEASE=1
DEB_RELEASE=1
VERSION="0.1"
RELEASE="1"
REVISION=0
BRANCH="nocoord"
REPO="https://github.com/percona/telemetry-agent.git"
PRODUCT=percona-telemetry-agent
parse_arguments PICK-ARGS-FROM-ARGV "$@"

check_workdir
get_system
install_deps
get_sources
build_srpm
build_source_deb
build_rpm
build_deb
