set -e

mkdir -p build/mac

go build  -ldflags="-w -s" -o build/mac/kubeconsole main.go
cd build/mac
tar -czf ../mac.tar.gz kubeconsole
cd ../..
shasum -a 256 build/mac.tar.gz
