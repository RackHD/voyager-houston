#!/usr/bin/env bash

set -e -x
source voyager-houston/ci/tasks/util.sh

check_param GITHUB_EMAIL
check_param GITHUB_USER
check_param GITHUB_PASSWORD

echo -e "machine github.com\n  login $GITHUB_USER\n  password $GITHUB_PASSWORD" >> ~/.netrc
git config --global user.email ${GITHUB_EMAIL}
git config --global user.name ${GITHUB_USER}
git config --global push.default current

export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
mkdir -p $GOPATH/src/github.com/RackHD/
cp -r voyager-houston $GOPATH/src/github.com/RackHD/voyager-houston

pushd $GOPATH/src/github.com/RackHD/voyager-houston
  make deps
  make build

  release_version=`cat version | tr -d '\n'`
  release_version=$((release_version+1))
  printf ${release_version} > version

  git add version
  git commit -m ":airplane: New release v${release_version}" -m "[ci skip]"

  printf "Houston Release v${release_version}" > name
  printf "v${release_version}" > tag
  tar -czvf voyager-houston-v${release_version}.tar.gz ./bin/*
  echo "New version released."
popd

cp -r $GOPATH/src/github.com/RackHD/voyager-houston release
