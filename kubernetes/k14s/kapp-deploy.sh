#!/bin/bash
VERSION=v1.0
COMMIT=`git rev-parse HEAD`
BRANCH=`git rev-parse --abbrev-ref HEAD`

kapp deploy -a introspect -f <(ytt template -v VERSION=$VERSION -v COMMIT=$COMMIT -v BRANCH=$BRANCH -f . | kbld -f -) --diff-changes
