#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

VERSION=$1
if [[ -z "${VERSION}" ]]; then
    echo "usage $0 version"
    exit 1
fi

BUILD_DIR=${DIR}/../build/snap/navidrome
mkdir -p ${BUILD_DIR}

ARCH=$(dpkg --print-architecture)
case ${ARCH} in
    amd64) NARCH=amd64; FFARCH=amd64 ;;
    arm64) NARCH=arm64; FFARCH=arm64 ;;
    armhf) NARCH=armv7; FFARCH=armhf ;;
    *) echo "unsupported arch ${ARCH}"; exit 1 ;;
esac

apt-get update
apt-get install -y wget ca-certificates xz-utils

wget -c --progress=dot:giga \
    https://github.com/navidrome/navidrome/releases/download/v${VERSION}/navidrome_${VERSION}_linux_${NARCH}.tar.gz \
    -O ${DIR}/../build/navidrome.tar.gz

tar xf ${DIR}/../build/navidrome.tar.gz -C ${BUILD_DIR} navidrome
chmod +x ${BUILD_DIR}/navidrome

wget -c --progress=dot:giga \
    https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${FFARCH}-static.tar.xz \
    -O ${DIR}/../build/ffmpeg.tar.xz

tar xf ${DIR}/../build/ffmpeg.tar.xz -C ${BUILD_DIR} --wildcards --strip-components=1 '*/ffmpeg'
chmod +x ${BUILD_DIR}/ffmpeg
${BUILD_DIR}/ffmpeg -version | head -1
