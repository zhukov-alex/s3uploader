package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	upl "github.com/zhukov-alex/s3uploader/pkg/uploader"
)

const testBucket = "test-bucket"

var uploader *upl.S3Uploader

func TestMain(m *testing.M) {
	// Setup tests
	cfg := upl.S3Config{
		Region:    "us-west-2",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Url:       "http://127.0.0.1:9000",
	}

	var err error
	uploader, err = upl.NewS3Uploader(cfg, 5*1024*1024, 5)
	if err != nil {
		log.Fatalf("Failed to initialize uploader: %v", err)
	}

	err = uploader.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		log.Fatalf("Failed to create test bucket: %v", err)
	}
	fmt.Println("Test bucket created:", testBucket)

	// Run tests
	code := m.Run()

	// Clean the bucket before deletion
	err = upl.CleanupBucket(context.Background(), uploader.GetClient(), testBucket)
	if err != nil {
		log.Printf("Failed to cleanup test bucket: %v", err)
	}

	// Delete the test bucket
	err = uploader.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		log.Printf("Failed to delete test bucket: %v", err)
	} else {
		fmt.Println("Test bucket deleted:", testBucket)
	}

	// Exit with the test execution code
	os.Exit(code)
}

// Acceptance tests
func TestAcceptance_SimpleUpload(t *testing.T) {
	// Create a temporary small file (1KB)
	tempFile := createTempFile(t, 1024)

	req := upl.UploadRequest{
		Bucket:   testBucket,
		FilePath: tempFile,
		Key:      "acceptance/small-file.txt",
	}

	err := uploader.Upload(context.Background(), req)
	require.NoError(t, err)

	fmt.Println("Simple upload test passed")
}

func TestAcceptance_MultipartUpload(t *testing.T) {
	// Create a temporary large file (11MB)
	tempFile := createTempFile(t, 11*1024*1024)

	req := upl.UploadRequest{
		Bucket:   testBucket,
		FilePath: tempFile,
		Key:      "acceptance/large-file.bin",
	}

	err := uploader.Upload(context.Background(), req)
	require.NoError(t, err)

	fmt.Println("Multipart upload test passed")
}

func TestAcceptance_InvalidCredentials(t *testing.T) {
	invalidCfg := upl.S3Config{
		Region:    "us-west-2",
		AccessKey: "invalid",
		SecretKey: "invalid",
		Url:       "http://127.0.0.1:9000",
	}

	invalidUploader, err := upl.NewS3Uploader(invalidCfg, 5*1024*1024, 5)
	require.NoError(t, err)

	tempFile := createTempFile(t, 1024)

	req := upl.UploadRequest{
		Bucket:   testBucket,
		FilePath: tempFile,
		Key:      "acceptance/invalid-credentials.txt",
	}

	err = invalidUploader.Upload(context.Background(), req)
	require.Error(t, err)

	fmt.Println("Invalid credentials test passed")
}

// createTempFile creates a temporary file of a specified size
func createTempFile(t *testing.T, size int64) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.bin")

	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer f.Close()

	data := bytes.Repeat([]byte("A"), 1024)
	var written int64

	for written < size {
		toWrite := size - written
		if toWrite > int64(len(data)) {
			toWrite = int64(len(data))
		}
		_, err := f.Write(data[:toWrite])
		require.NoError(t, err)
		written += toWrite
	}
	return filePath
}
