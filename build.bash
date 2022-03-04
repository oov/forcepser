#!/bin/bash

mkdir -p bin

# copy readme
sed 's/\r$//' README.md | sed 's/$/\r/' > bin/forcepser.txt

# copy template
sed 's/\r$//' src/setting.txt-template | sed 's/$/\r/' > bin/setting.txt-template
sed 's/\r$//' src/setting.txt-template-old | sed 's/$/\r/' > bin/setting.txt-template-old

# copy script
sed 's/\r$//' src/lua/_entrypoint.lua | sed 's/$/\r/' > bin/_entrypoint.lua

# update version string
VERSION='v1.5'
GITHASH=`git rev-parse --short HEAD`
cat << EOS | sed 's/\r$//' | sed 's/$/\r/' > 'src/go/ver.go'
package main

const version = "$VERSION ( $GITHASH )"
EOS

# build forcepser.exe
pushd src/go > /dev/null
env.exe GOARCH=386 go build -x -ldflags="-s" -o ../../bin/forcepser.exe
popd > /dev/null