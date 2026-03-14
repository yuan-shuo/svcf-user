#!/bin/bash
go test ./internal/utils/... \
        ./internal/logic/... \
        ./internal/db/... \
        ./internal/mqs/... \
        -cover