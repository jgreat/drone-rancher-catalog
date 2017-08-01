#!/bin/bash
# Set GITHUB vars in your local environment
set -e

echo "-- Installing Dependancies"
glide install
echo "-- Tests"
go test -v 
echo "-- Build"
go build -v

version=$(./drone-rancher-catalog --version | awk '{print $4}')

echo "-- Build Docker Container"
docker build --rm -t jgreat/drone-rancher-catalog:latest .
docker tag jgreat/drone-rancher-catalog:latest jgreat/drone-rancher-catalog:${version}

# echo "-- Run Docker Container"
# docker run -it --rm \
# -e PLUGIN_DEBUG=true \
# -e PLUGIN_TAGS=1.0.1-1499290301.test-branch.1234abcd,1.0.1,latest \
# -e PLUGIN_DRY_RUN=false \
# -e PLUGIN_DOCKER_REPO=jgreat/catalog-test \
# -e PLUGIN_CATALOG_REPO=jgreat/rancher-catalog-test \
# -e PLUGIN_RELEASE_BRANCH=master \
# -e 'PLUGIN_TAG_REGEX=[0-9]+[.][0-9]+[.][0-9]+$' \
# -e DRONE_BUILD_NUMBER=1499290301 \
# -e DRONE_COMMIT_BRANCH=test-branch \
# -e DRONE_REPO_NAME=catalog-test \
# -e GITHUB_EMAIL=${GITHUB_EMAIL} \
# -e GITHUB_USERNAME=${GITHUB_USERNAME} \
# -e GITHUB_TOKEN=${GITHUB_TOKEN} \
# jgreat/drone-rancher-catalog:${version}

