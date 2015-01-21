#!/bin/sh
# go test ./... -run Unit -v
go test ./... -run Integration -v
