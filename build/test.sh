#!/usr/bin/env bash
set -e
go test -v -cover -race `go list ./... | grep -v assets`
