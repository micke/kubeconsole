set -e

mkdir -p build/mac/completion
mkdir -p build/linux/completion

env GOOS=darwin go build -ldflags="-w -s" -o build/mac/kubeconsole main.go
env GOOS=linux go build -ldflags="-w -s" -o build/linux/kubeconsole main.go

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  ./build/linux/kubeconsole completion bash | tee ./build/mac/completion/bash ./build/linux/completion/bash
  ./build/linux/kubeconsole completion zsh | tee ./build/mac/completion/zsh ./build/linux/completion/zsh
elif [["$OSTYPE" == "darwin"* ]]; then
  ./build/mac/kubeconsole completion bash | tee ./build/mac/completion/bash ./build/linux/completion/bash
  ./build/mac/kubeconsole completion zsh | tee ./build/mac/completion/zsh ./build/linux/completion/zsh
fi

cd build/mac
tar -czf ../mac.tar.gz kubeconsole completion
cd -
cd build/linux
tar -czf ../linux.tar.gz kubeconsole completion
cd -

echo mac SHA
shasum -a 256 build/mac.tar.gz
echo linux SHA
shasum -a 256 build/linux.tar.gz
