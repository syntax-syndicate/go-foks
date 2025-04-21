#!/bin/sh
# Copyright (c) 2025 ne43, Inc.
# Licensed under the MIT License. See LICENSE in the project root for details.


. ./env.sh

export GOBIN=$(realpath ${BINDIR})

(cd ${SRCDIR}/server/foks-tool && go install)
(cd ${SRCDIR}/server/foks-server && go install)

export GOBIN=$(realpath ${BINDIR_CLI})
(cd ${SRCDIR}/client/foks && go install)
