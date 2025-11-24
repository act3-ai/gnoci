#!/usr/bin/env sh

go tool cover -func="$1" | grep total | awk '{print $3}' | sed 's/%//'