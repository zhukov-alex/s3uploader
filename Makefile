.PHONY: build docker-build docker-up docker-down test acceptance-test clean

build:
	go build -o s3uploader ./cmd/uploader/main.go

docker-build:
	docker build -t s3uploader:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

test:
	go test -v ./pkg/...

acceptance-test:
	go test -v ./tests/...

clean:
	rm -f s3uploader
