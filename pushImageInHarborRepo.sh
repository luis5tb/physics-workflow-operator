#!/bin/sh
IMAGE=physics-workflow-operator
TAG=v0.0.1
REPOPATH=wp5
docker login https://registry.apps.ocphub.physics-faas.eu
docker tag $IMAGE:$TAG registry.apps.ocphub.physics-faas.eu/$REPOPATH/$IMAGE:$TAG
docker push registry.apps.ocphub.physics-faas.eu/$REPOPATH/$IMAGE:$TAG
