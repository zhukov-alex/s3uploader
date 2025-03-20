FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o s3uploader ./cmd/uploader/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/s3uploader .

ENTRYPOINT ["./s3uploader"]
