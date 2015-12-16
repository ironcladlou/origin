#!/bin/bash
set -e

tmp=$(mktemp -d $TMPDIR/origin-rebase.XXX)
echo "Rebasing in $tmp"

source_gopath="$1"
GOPATH=$tmp

if [ "$source_gopath" != "" ]; then
  echo "Cloning from local GOPATH $source_gopath"
  git clone $source_gopath/src/k8s.io/kubernetes $GOPATH/src/k8s.io/kubernetes
  git clone $source_gopath/src/github.com/openshift/origin $GOPATH/src/github.com/openshift/origin
else
  echo "Cloning from GitHub"
  git clone https://github.com/kubernetes/kubernetes.git $GOPATH/src/k8s.io/kubernetes
  git clone https://github.com/openshift/origin.git $GOPATH/src/github.com/openshift/origin
fi

echo "Installing godep..."
pushd $GOPATH > /dev/null
GOPATH=$GOPATH go get github.com/tools/godep
cd $GOPATH/src/github.com/tools/godep
curl https://patch-diff.githubusercontent.com/raw/tools/godep/pull/365.patch | git am
GOPATH=$GOPATH go install github.com/tools/godep
popd > /dev/null

echo "Preparing kubernetes..."
pushd $GOPATH/src/k8s.io/kubernetes > /dev/null
git checkout master
git pull
git checkout -b stable_proposed
popd > /dev/null

echo "Preparing origin..."
pushd $GOPATH/src/github.com/openshift/origin > /dev/null
git checkout master
git pull
popd > /dev/null

echo "Restoring origin dependencies..."
pushd $GOPATH/src/github.com/openshift/origin > /dev/null
make clean
GOPATH=$GOPATH GOOS=linux $GOPATH/bin/godep restore
popd > /dev/null

echo "Restoring Kubernetes dependencies ..."
pushd $GOPATH/src/k8s.io/kubernetes > /dev/null
git checkout stable_proposed
rm -rf _output Godeps/_workspace/pkg
GOPATH=$GOPATH GOOS=linux $GOPATH/bin/godep restore -v
popd > /dev/null

echo "Restore complete, update any packages which must diverge from Kubernetes now"
echo
echo "Hit ENTER to continue"
read

echo "Saving dependencies ..."
pushd $GOPATH/src/github.com/openshift/origin > /dev/null
git rm -r Godeps
GOPATH=$GOPATH GOOS=linux $GOPATH/bin/godep save ./...
git add .
popd > /dev/null

echo "SUCCESS: Added all new dependencies, review Godeps/Godeps.json"
echo "  To check upstreams, run: git log -E --grep=\"^UPSTREAM:|^bump\" --oneline"
