package uploader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

// dummyHTTPClient mock HTTP client that always returns code 200
type dummyHTTPClient struct{}

func (d *dummyHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("")),
		Header:     make(http.Header),
	}, nil
}

// mockS3API mock implementation of S3API
type mockS3API struct {
	putObjectCalled             bool
	createMultipartUploadCalled bool
	uploadPartCalls             int
	completeMultipartCalled     bool
	abortMultipartCalled        bool

	putObjectErr  error
	uploadPartErr error
}

func (m *mockS3API) PutObject(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	m.putObjectCalled = true
	if m.putObjectErr != nil {
		return nil, m.putObjectErr
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3API) CreateMultipartUpload(ctx context.Context, in *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	m.createMultipartUploadCalled = true
	return &s3.CreateMultipartUploadOutput{UploadId: stringPtr("upload-id")}, nil
}

func (m *mockS3API) UploadPart(ctx context.Context, in *s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	m.uploadPartCalls++
	if m.uploadPartErr != nil {
		return nil, m.uploadPartErr
	}
	return &s3.UploadPartOutput{ETag: stringPtr(fmt.Sprintf("etag-%d", *in.PartNumber))}, nil
}

func (m *mockS3API) CompleteMultipartUpload(ctx context.Context, in *s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error) {
	m.completeMultipartCalled = true
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (m *mockS3API) AbortMultipartUpload(ctx context.Context, in *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	m.abortMultipartCalled = true
	return &s3.AbortMultipartUploadOutput{}, nil
}

func (m *mockS3API) GetClient() *s3.Client {
	return s3.NewFromConfig(aws.Config{Region: "us-east-1"}, func(o *s3.Options) {
		o.HTTPClient = &dummyHTTPClient{}
	})
}

func (m *mockS3API) CreateBucket(ctx context.Context, in *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{}, nil
}

func (m *mockS3API) DeleteBucket(ctx context.Context, in *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return &s3.DeleteBucketOutput{}, nil
}

func stringPtr(s string) *string { return &s }

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

func newTestS3Uploader(mock *mockS3API, partSize int, concurrencyLimit int) *S3Uploader {
	uploader, err := NewS3Uploader(S3Config{
		Region:    "us-east-1",
		AccessKey: "dummy",
		SecretKey: "dummy",
		Url:       "",
	}, partSize, concurrencyLimit)
	if err != nil {
		panic(err)
	}

	uploader.s3api = mock
	uploader.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, partSize)
		},
	}
	return uploader
}

func TestUploader_Uploads(t *testing.T) {
	t.Parallel()

	// Test case for uploading a small file (should use PutObject)
	t.Run("Small file", func(t *testing.T) {
		mock := &mockS3API{}
		uploader := newTestS3Uploader(mock, 5*1024*1024, 0) // 5MB partSize
		smallFile := createTempFile(t, 1024) // 1KB
		err := uploader.Upload(context.Background(), UploadRequest{
			Bucket:   "my-bucket",
			FilePath: smallFile,
			Key:      "my-small-object",
		})
		require.NoError(t, err)
		require.True(t, mock.putObjectCalled, "PutObject should be called for small files")
		require.False(t, mock.createMultipartUploadCalled, "Multipart upload should not be initiated for small files")
	})

	// Test case for uploading a large file (should use multipart upload)
	t.Run("Large file", func(t *testing.T) {
		mock := &mockS3API{}
		uploader := newTestS3Uploader(mock, 5*1024*1024, 2) // 5MB part size, concurrencyLimit = 2
		largeFile := createTempFile(t, 11*1024*1024) // 11MB file => multiple parts
		err := uploader.Upload(context.Background(), UploadRequest{
			Bucket:   "my-bucket",
			FilePath: largeFile,
			Key:      "my-large-object",
		})
		require.NoError(t, err)
		require.False(t, mock.putObjectCalled, "PutObject should NOT be called for large files")
		require.True(t, mock.createMultipartUploadCalled, "Multipart upload should be initiated for large files")
		require.True(t, mock.completeMultipartCalled, "CompleteMultipartUpload is expected for multipart uploads")
		require.GreaterOrEqual(t, mock.uploadPartCalls, 2, "At least 2 parts should be uploaded")
	})
}
