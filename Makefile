main.wasm: *.go
	 GOOS=js GOARCH=wasm go build -o main.wasm

install:
	npm install -g standard
	go get -u golang.org/x/lint/golint
	go get -t ./...

all: main.wasm
	./update-statics.sh

devserver: main.wasm
	goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'

clean:
	rm -rf ext main.wasm

publish: clean
	./publish.sh

lint-js: *.js
	standard *.js

lint-fix-js: *.js
	standard --fix *.js && standard *.js

lint-go: *.go
	golint ./...

lint-all: lint-go lint-js

test:
	go test -v ./...
