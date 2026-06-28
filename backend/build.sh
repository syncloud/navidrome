#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
ROOT=$( cd "${DIR}/.." && pwd )
BUILD_DIR=${ROOT}/build/snap/backend

mkdir -p ${BUILD_DIR}
cd ${DIR}

go vet ./...

export CGO_ENABLED=0
go build -trimpath -buildvcs=false -ldflags '-s -w' -o ${BUILD_DIR}/backend ./cmd/backend
