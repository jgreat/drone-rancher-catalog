#!/bin/bash
# Set $DOCKER_USER, $DOCKER_PASS and $GITHUB_TOKEN in your environment so we don't commit them on accident
docker run -i --rm \
-v ${GOPATH}/src/github.com/jgreat/drone-rancher-catalog/.test-data/app:/app \
-v ${GOPATH}/src/github.com/jgreat/drone-rancher-catalog/.test-data/rancher-catalog:/rancher-catalog \
leankit/drone-rancher-catalog <<EOF
{
	"build": {
		"Number": 56
	},
	"workspace": {
		"path": "/app"
	},
	"repo": {
		"name": "core-leankit-api",
		"owner": "BanditSoftware"
	},
	"vargs": {
		"docker_username": "$DOCKER_USER",
		"docker_password": "$DOCKER_PASS",
		"docker_repo": "leankit/core-leankit-api",
		"catalog_repo": "jgreat/rancher-core-leankit-api",
		"github_token": "$GITHUB_TOKEN",
		"github_user": "Jason Greathouse",
		"github_email": "jason.greathouse@leankit.com"
	}
}
EOF
