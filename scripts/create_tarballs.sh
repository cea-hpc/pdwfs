#!/bin/sh

if [ -z $TAG ]; then
	TAG=`git describe`
fi
NAME="pdwfs-${TAG}"
ARCH="x86_64"
OS="linux"

SRC_TAR="${NAME}.tar"
RELEASE_NAME="${NAME}-${OS}-${ARCH}"
RELEASE_TAR="${RELEASE_NAME}.tar"

mkdir -p dist
rm -f dist/${SRC_TAR}.gz
rm -f dist/${RELEASE_TAR}.gz

echo "Generating source distribution: dist/${SRC_TAR}.gz"
git archive ${TAG} --prefix pdwfs-${TAG}/ > dist/${SRC_TAR} || exit 1
gzip -9 dist/${SRC_TAR}


echo "Generating a release distribution: dist/${RELEASE_TAR}"
BRANCH=`git rev-parse --abbrev-ref HEAD`
git checkout ${TAG} || exit 1
make PREFIX=${PWD}/build/${RELEASE_NAME} install
tar czf dist/${RELEASE_TAR}.gz -C build ${RELEASE_NAME} 
rm -rf build/${RELEASE_NAME}
git checkout ${BRANCH}