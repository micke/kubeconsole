#!/bin/sh

set -e

rm -rf completions
mkdir -p completions

for shell in bash zsh; do
  go run main.go completion "$shell" > "completions/kubeconsole.$shell"
done
