#! /bin/bash

echo "Building Begin: $(date '+%H:%M:%S')"

# Linux
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main

# Windows
#$env:CGO_ENABLED=0
#go build -o xray.exe -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main

echo "Building End: $(date '+%H:%M:%S')"
