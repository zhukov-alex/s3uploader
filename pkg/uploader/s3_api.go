package uploader

import (
"context"

"github.com/aws/aws-sdk-go-v2/service/s3"
)


// S3API defines an interface for working with S3 (AWS or MinIO)
type S3API interface {
	GetClient() *s3.Client
	CreateBucket(ctx context.Context, in *s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, in *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)
	PutObject(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	CreateMultipartUpload(ctx context.Context, in *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(ctx context.Context, in *s3.UploadPartInput) (*s3.UploadPartOutput, error)
	CompleteMultipartUpload(ctx context.Context, in *s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(ctx context.Context, in *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error)
}

// s3ClientWrapper — implementation of the S3API interface.
type s3ClientWrapper struct {
	*s3.Client
}

func (w *s3ClientWrapper) GetClient() *s3.Client {
	return w.Client
}

func (w *s3ClientWrapper) PutObject(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return w.Client.PutObject(ctx, in)
}

func (w *s3ClientWrapper) CreateMultipartUpload(ctx context.Context, in *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	return w.Client.CreateMultipartUpload(ctx, in)
}

func (w *s3ClientWrapper) UploadPart(ctx context.Context, in *s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	return w.Client.UploadPart(ctx, in)
}

func (w *s3ClientWrapper) CompleteMultipartUpload(ctx context.Context, in *s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error) {
	return w.Client.CompleteMultipartUpload(ctx, in)
}

func (w *s3ClientWrapper) AbortMultipartUpload(ctx context.Context, in *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	return w.Client.AbortMultipartUpload(ctx, in)
}

func (w *s3ClientWrapper) CreateBucket(ctx context.Context, in *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return w.Client.CreateBucket(ctx, in)
}

func (w *s3ClientWrapper) DeleteBucket(ctx context.Context, in *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return w.Client.DeleteBucket(ctx, in)
}
