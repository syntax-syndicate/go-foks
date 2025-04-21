FROM golang:1.24.0-bookworm

WORKDIR /foks

COPY go.mod go-foks-alpha/go.mod
COPY go.sum go-foks-alpha/go.sum

RUN (cd go-foks-alpha && go mod download)

RUN apt-get update && apt-get install -y libpcsclite-dev && rm -rf /var/lib/apt/lists/*

COPY . go-foks-alpha
RUN (cd go-foks-alpha/client/foks && go build)
RUN (cd / && ls -lsR .)
