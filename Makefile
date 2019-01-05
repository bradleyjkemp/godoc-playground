godoc-playground.js: *.go
	gopherjs build -m .

install:
	npm install -g standard
	go get -u golang.org/x/lint/golint
	go get -u github.com/gopherjs/gopherjs
	go get -t ./...

all: godoc-playground.js
	./update-statics.sh

devserver: all
	goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'

clean:
	rm -rf ext godoc-playground.js*

publish: clean
	./publish.sh

lint-js: *.js
	standard index.js

lint-fix-js: *.js
	standard --fix index.js && standard index.js

lint-go: *.go
	golint ./...

lint: lint-go lint-js

test:
	go test -v ./...
