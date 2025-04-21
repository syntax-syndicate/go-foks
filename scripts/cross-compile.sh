#!/bin/sh
# Copyright (c) 2025 ne43, Inc.
# Licensed under the MIT License. See LICENSE in the project root for details.


usage() {
    echo "Usage: $0 {linux-arm64|linux-amd64}"
    exit 1
}

if [ $# -ne 1 ]; then
    usage
fi

plat=$1
if [ "$plat" != "linux-arm64" ] && [ "$plat" != "linux-amd64" ]; then
    usage
fi

build() {
    plat=$1
    output=foks.${plat}
    name=foks-${plat}-build
    tmp=temp-foks-${plat}
    docker_plat=$(echo $plat | sed 's#-#/#')

    docker build --platform=${docker_plat} -t ${name} . 
    docker create --platform=${docker_plat} --name=${tmp} ${name}
    docker cp ${tmp}:/foks/go-foks/client/foks/foks build/${output}
    docker rm ${tmp}

    echo "Build for ${plat} is complete: build/${output}"
}

build $plat
