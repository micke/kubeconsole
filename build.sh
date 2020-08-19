set -e

mkdir -p build/mac/completion

go build  -ldflags="-w -s" -o build/mac/kubeconsole main.go
cd build/mac
./kubeconsole completion bash > completion/bash
./kubeconsole completion zsh > completion/zsh
tar -czf ../mac.tar.gz kubeconsole completion
cd ../..
shasum -a 256 build/mac.tar.gz
