#!/bin/bash
for test in command config; do
    go test -cover "./${test}"
done
