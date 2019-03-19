#!/bin/sh

if [ -z $TAG ]; then
	TAG=`git describe`
fi

NAME="pdwfs-${TAG}"
export GOARCH="amd64"
export GOOS="linux"

RELEASE_NAME="${NAME}-${GOOS}-${GOARCH}"
RELEASE="${RELEASE_NAME}.tar.gz"

mkdir -p dist
rm -f dist/${RELEASE}

echo "Generating a release distribution: dist/${RELEASE}"
BRANCH=`git rev-parse --abbrev-ref HEAD`
git checkout ${TAG} || exit 1
make PREFIX=${PWD}/build/${RELEASE_NAME} install
tar czf dist/${RELEASE} -C build ${RELEASE_NAME} 
rm -rf build/${RELEASE_NAME}
git checkout ${BRANCH}