#!/bin/sh
set -eu

if [ "$#" -ne 2 ] || [ "$1" != "count" ]; then
	printf 'usage: sh word-count.sh count <file>\n' >&2
	exit 2
fi

count=$(wc -w < "$2")
printf 'Word count: %s\n' "$count"
