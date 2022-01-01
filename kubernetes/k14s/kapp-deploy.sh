#!/bin/bash
dir="${0%/*}"
cd $dir

gitVersion="0.0.0-dev"
gitCommit=$(git rev-parse --verify HEAD)
gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty)

kapp deploy -a introspect -f <(ytt template -v gitVersion=$gitVersion -v gitCommit=$gitCommit -v gitTreeState=$gitTreeState -f . | kbld --lock-output kbld.lock.yaml -f -) --diff-changes
