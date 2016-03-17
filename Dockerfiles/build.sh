#!/bin/bash

# Make sure i) can sudo as root and ii) Docker is running

IMAGE_NAME=haih/remoteabac

cp $GOPATH/bin/remoteabac .

sudo docker build -f Dockerfile -t $IMAGE_NAME --no-cache .

# Optionally one can push the image
#docker push $IMAGE_NAME
