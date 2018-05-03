#!/bin/bash -x

ARG_PKG=github.com/container-bootcamp/einbuchen
VERSION=latest
BIN=einbuchen
REPO=quay.io/containerbootcamp

# start compile build step
build=`buildah from golang:1.9`
buildah copy $build . /go/src/$ARG_PKG/
buildah config --workingdir /go/src/$ARG_PKG/ $build
buildah run $build -- go get -u github.com/golang/dep/cmd/dep
buildah run $build -- dep ensure
buildah run $build -- /bin/sh -c "GOARCH=amd64 CGO_ENABLED=0 go install \
    -installsuffix 'static' \
    ./..."

# create and push target image
target=`buildah from scratch`
buildah copy $target $(buildah mount $build)/go/bin/$BIN /$BIN
buildah config --cmd "/$BIN" $target
imageId=`buildah commit --rm $target $REPO/$BIN:$VERSION`
buildah push $imageId docker-daemon:$REPO/$BIN:$VERSION

# cleanup
buildah rm $build