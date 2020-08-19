set -e

mkdir -p build/mac

go build  -ldflags="-w -s" -o build/mac/devflow main.go
cd build/mac
tar -czf ../mac.tar.gz devflow
cd ../..
shasum -a 256 build/mac.tar.gz
