#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

VERSION=$1
if [ -z "${VERSION}" ]; then
    echo "usage $0 version"
    exit 1
fi

BUILD_DIR=${DIR}/../build/snap/navidrome

OUT=$(${BUILD_DIR}/navidrome --version)
echo "navidrome --version: ${OUT}"
echo "${OUT}" | grep -q "${VERSION}"
echo "navidrome ${VERSION} ok"
