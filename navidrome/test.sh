#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/navidrome
${BUILD_DIR}/navidrome --help > /dev/null
echo "navidrome binary ok"
