#!/usr/bin/env sh

grep -v \
    -e zz_generated \
    -e '\.gen\.go'
