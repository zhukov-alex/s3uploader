# S3Uploader

A robust, concurrent, and testable file uploader to AWS S3-compatible storage (e.g., Minio, AWS S3), implemented in Go.

---

## Project Overview

`S3Uploader` provides functionality to upload small and large files efficiently. Small files are uploaded using the simple S3 PutObject API call, while large files utilize multipart uploads for better performance.

## Architectural Decisions

### 1. Interface Abstraction

The `S3API` interface abstracts the AWS SDK client, allowing easy mocking and replacement for testing purposes.

### 2. Concurrency and Buffer Management

- Utilizes `sync.Pool` to reuse buffers efficiently and minimize memory allocation overhead.
- Employs `errgroup` from `golang.org/x/sync` with concurrency limits for controlled parallel uploads.

## Usage

### Build and Run Locally

```shell
make
./s3uploader
```

### Docker Build and Run

```shell
make docker-build
docker run -v $(pwd):/data --network host --env-file .env s3uploader:latest -b <bucket> -f <file> [-k key] [-create-bucket]
```

### Running Tests

- Run unit tests:

```shell
make test
```

- Run acceptance tests (minio):

```shell
make docker-up # starts Minio
make acceptance-test
```
Access the Minio UI at: `http://localhost:9001` (User: `minioadmin`, Password: `minioadmin`).

- Stop Minio when done:

```shell
make docker-down
```

## Configuration

Credentials and configurations can be set using environment variables (.env.example):

```shell
S3_REGION
S3_ACCESS_KEY
S3_SECRET_KEY
S3_URL
```

## Makefile commands

- `make`: Builds the project.
- `make docker-build`: Builds Docker image.
- `make docker-up`: Runs Minio locally via Docker.
- `make docker-down`: Stops Minio.
- `make test`: Runs unit tests.
- `make acceptance-test`: Runs acceptance tests.
- `make clean`: Cleans compiled binaries.
