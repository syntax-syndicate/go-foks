#!/bin/sh
# Copyright (c) 2025 ne43, Inc.
# Licensed under the MIT License. See LICENSE in the project root for details.


bin=$(go env BIN)
if [ -n "$bin" ]; then
    echo $bin
    exit 0
fi

path=$(go env GOPATH)
if [ -n "$path"  -a -d $path/bin ]; then
    echo $path/bin
    exit 0
fi

echo "Not found"
exit 1