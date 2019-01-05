#!/usr/bin/env bash

godocStatics="$GOPATH/src/golang.org/x/tools/godoc/static"

mkdir -p ext

cp ${godocStatics}/godocs.js \
    ${godocStatics}/jquery.js \
    ${godocStatics}/jquery.treeview.edit.js \
    ${godocStatics}/jquery.treeview.js \
    ${godocStatics}/jquery.treeview.css \
    ${godocStatics}/style.css \
    ./ext
