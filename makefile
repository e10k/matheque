run:
	go run --tags "sqlite_fts5" .

build-linux-amd64:
	docker run --rm -it --name mthq -v $(shell pwd):/go/src/github.com/e10k/matheque -w /go/src/github.com/e10k/matheque golang env GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static" --tags fts5 -o bin/matheque-amd64-linux
	cp init.sql bin/init.sql
	cp .config.example bin/.config.example

deploy:
	./deploy.sh