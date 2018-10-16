main.wasm: *.go
	 GOOS=js GOARCH=wasm go build -o main.wasm

all: main.wasm
	./update-statics.sh

devserver: main.wasm
	goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'

clean:
	rm -rf ext main.wasm

publish: clean
	./publish.sh
