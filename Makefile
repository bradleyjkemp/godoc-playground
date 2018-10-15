main.wasm: *.go
	 GOOS=js GOARCH=wasm go build -o main.wasm

all: main.wasm
	./update-statics.sh
