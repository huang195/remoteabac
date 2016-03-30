#!/bin/bash

# Make sure i) can sudo as root and ii) Docker is running

IMAGE_NAME=haih/remoteabac

godep go install ../cmd/remoteabac/remoteabac.go
godep go install ../cmd/ruser/ruser.go

cp $GOPATH/bin/remoteabac .
cp $GOPATH/bin/ruser .

sudo docker build -f Dockerfile -t $IMAGE_NAME --no-cache .

# Optionally one can push the image
#docker push $IMAGE_NAME
