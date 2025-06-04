#!/bin/bash

# set -euo pipefail
set -x 

usage() {
    echo "Usage: $0 -p {arm64|amd64} [-s]"
    exit 1
}

strip=0
plat=""

# take two arguments: -p which can be arm64 or amd64, and also
# -s, which is a boolean flag that means to strip the binary
while getopts ":p:s" opt; do
    case $opt in
        p)
            plat=$OPTARG
            ;;
        s)
            strip=1
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            usage
            ;;
        :)
            echo "Option -$OPTARG requires an argument." >&2
            usage
            ;;
    esac
done


shift $((OPTIND -1))

if [ $# -ne 0 ]; then
   usage
fi

if [ "$plat" != "arm64" ] && [ "$plat" != "amd64" ]; then
    usage
fi

version=$(git describe --tags --always)
trimpath=""

ldflags="-X main.LinkerVersion=${version}"
if [ $strip -eq 1 ]; then
    ldflags="${ldflags} -w"
    trimpath="-trimpath"
fi

GOOS=linux GOARCH=${plat} GOFLAGS="-tags=noresinit" \
    go build \
        -C server/foks-tool \
        -o ../../build/foks-tool.linux.${plat} \
        -ldflags "${ldflags}" \
        $trimpath \
        .

gzip -f build/foks-tool.linux.${plat}
