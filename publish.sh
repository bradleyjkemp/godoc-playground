#!/usr/bin/env bash
# Fail if any command fails
set -e

git checkout gh-pages
git reset --hard master
make all
git add ext main.wasm
git commit -m "Publish"
echo "Now check this works and push"
