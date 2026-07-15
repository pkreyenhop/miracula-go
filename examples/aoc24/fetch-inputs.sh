#!/bin/sh
# Fetch your personal Advent of Code 2024 inputs into examples/aoc24/inputs/.
#
# The dayN.txt files in inputs/ are seeded with the official example data so
# every solver runs out of the box; this script replaces them with your real
# puzzle inputs (the dayN-example.txt files are left untouched).
#
# Usage:
#   1. Log in at https://adventofcode.com and copy your "session" cookie value
#      (browser dev tools -> Application/Storage -> Cookies).
#   2. AOC_SESSION=<cookie value> sh examples/aoc24/fetch-inputs.sh
#
# Please keep your inputs to yourself — adventofcode.com asks that puzzle
# inputs not be republished.

if [ -z "$AOC_SESSION" ]; then
    echo "Set AOC_SESSION to your adventofcode.com session cookie first." >&2
    exit 1
fi

dir="$(dirname "$0")/inputs"
for day in $(seq 1 25); do
    out="$dir/day$day.txt"
    curl -sf --cookie "session=$AOC_SESSION" \
         -A "miracula-go examples fetch script" \
         "https://adventofcode.com/2024/day/$day/input" -o "$out" \
        && echo "day $day -> $out" \
        || echo "day $day FAILED (check AOC_SESSION)" >&2
    sleep 1
done
