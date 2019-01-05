#!/usr/bin/env bash
# Fail if any command fails
set -e

git fetch
git checkout gh-pages
git reset --hard origin/master
make all
git add ext godoc-playground.js*
git commit -m "Publish"
echo "Now check this works and push"
