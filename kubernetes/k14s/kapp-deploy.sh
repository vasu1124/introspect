#!/bin/bash
dir="${0%/*}"
cd $dir

VERSION=1.0.0
COMMIT=`git rev-parse HEAD`
BRANCH=`git rev-parse --abbrev-ref HEAD`

kapp deploy -a introspect -f <(ytt template -v VERSION=$VERSION -v COMMIT=$COMMIT -v BRANCH=$BRANCH -f . | kbld --lock-output kbld.lock.yaml -f -) --diff-changes
