#!/bin/sh
# Copyright (c) 2025 ne43, Inc.
# Licensed under the MIT License. See LICENSE in the project root for details.


docker stop foks-postgresql
docker rm foks-postgresql
docker volume rm foks-db
