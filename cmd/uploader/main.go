package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	upl "github.com/zhukov-alex/s3uploader/pkg/uploader"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func usage() {
	log.Printf("Usage: uploader -b <bucket> -f <file> [-k key] [-create-bucket]")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	bucket := flag.String("b", "", "S3 bucket name")
	filePath := flag.String("f", "", "Path to the file to upload")
	key := flag.String("k", "", "Object key in S3 (optional, defaults to filePath)")
	createBucket := flag.Bool("create-bucket", false, "Force create the bucket before uploading")
	flag.Parse()

	if *bucket == "" || *filePath == "" {
		showUsageAndExit(1)
	}

	objectKey := *key
	if objectKey == "" {
		objectKey = *filePath
	}

	cfg := upl.S3Config{
		Region:    getEnv("S3_REGION", "us-west-2"),
		AccessKey: getEnv("S3_ACCESS_KEY", ""),
		SecretKey: getEnv("S3_SECRET_KEY", ""),
		Url:       getEnv("S3_URL", ""),
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		log.Fatal("Missing S3_ACCESS_KEY or S3_SECRET_KEY environment variables")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("Received shutdown signal, canceling context...")
		cancel()
	}()

	// Using 5MB part size and concurrency of 5
	uploader, err := upl.NewS3Uploader(cfg, 5*1024*1024, 5)
	if err != nil {
		log.Fatalf("Error initializing uploader: %v", err)
	}

	// If -create-bucket is set, create the bucket explicitly
	if *createBucket {
		fmt.Printf("Creating bucket: %s...\n", *bucket)
		err := uploader.CreateBucket(context.Background(), &s3.CreateBucketInput{
			Bucket: aws.String(*bucket),
		})
		if err != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}
		fmt.Printf("Bucket %s created successfully.\n", *bucket)
	}

	req := upl.UploadRequest{
		Bucket:   *bucket,
		FilePath: *filePath,
		Key:      objectKey,
	}

	if err := uploader.Upload(ctx, req); err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("Done!")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
