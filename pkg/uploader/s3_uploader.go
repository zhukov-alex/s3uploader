package uploader

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"
)

type S3Uploader struct {
	s3api      S3API
	bufferPool sync.Pool
	partSize         int
	concurrencyLimit int
}

func NewS3Uploader(cfg S3Config, partSize int, concurrencyLimit int) (*S3Uploader, error) {
	client := s3.NewFromConfig(aws.Config{Region: cfg.Region}, func(o *s3.Options) {
		if cfg.Url != "" {
			o.BaseEndpoint = aws.String(cfg.Url)
		}
		o.Credentials = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	})

	return &S3Uploader{
		s3api:            &s3ClientWrapper{client},
		partSize:         partSize,
		concurrencyLimit: concurrencyLimit,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, partSize)
			},
		},
	}, nil
}

func (u *S3Uploader) GetClient() *s3.Client {
	return u.s3api.GetClient()
}

func (u *S3Uploader) CreateBucket(ctx context.Context, in *s3.CreateBucketInput) error {
	_, err := u.s3api.CreateBucket(ctx, in)
	return err
}

func (u *S3Uploader) DeleteBucket(ctx context.Context, in *s3.DeleteBucketInput) error {
	_, err := u.s3api.DeleteBucket(ctx, in)
	return err
}

// Upload determines whether to use a simple or multipart upload based on file size
func (u *S3Uploader) Upload(ctx context.Context, req UploadRequest) error {
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() <= int64(u.partSize) {
		return u.simpleUpload(ctx, req)
	}
	return u.multipartUpload(ctx, req, fileInfo.Size())
}

// simpleUpload handles the upload of small files in a single request
func (u *S3Uploader) simpleUpload(ctx context.Context, req UploadRequest) error {
	file, err := os.Open(req.FilePath)
	if err != nil {
		return fmt.Errorf("simpleUpload: %w", err)
	}
	defer file.Close()

	_, err = u.s3api.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(req.Bucket),
		Key:    aws.String(req.Key),
		Body:   file,
	})

	if err != nil {
		log.Printf("Couldn't upload file. %v\n", err)
	} else {
		err = s3.NewObjectExistsWaiter(u.s3api.GetClient()).Wait(
			ctx, &s3.HeadObjectInput{Bucket: aws.String(req.Bucket), Key: aws.String(req.Key)}, time.Minute)
		if err != nil {
			log.Printf("Failed attempt to wait for object %s to exist.\n", req.Key)
		}
	}

	return err
}

// multipartUpload handles multi-part upload for large files to S3
func (u *S3Uploader) multipartUpload(ctx context.Context, req UploadRequest, size int64) error {
	// Open the file for reading
	file, err := os.Open(req.FilePath)
	if err != nil {
		return fmt.Errorf("multipartUpload: %w", err)
	}
	defer file.Close()

	// Initiate a multi-part upload session
	resp, err := u.s3api.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &req.Bucket,
		Key:    &req.Key,
	})
	if err != nil {
		return fmt.Errorf("create multipart: %w", err)
	}

	// Calculate the total number of parts required
	totalParts := int((size + int64(u.partSize) - 1) / int64(u.partSize))
	completedParts := make([]s3types.CompletedPart, totalParts)

	// Use an errgroup to upload parts concurrently
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(u.concurrencyLimit)

	for i := 0; i < totalParts; i++ {
		partNum := int32(i + 1)
		offset := int64(u.partSize) * int64(i)

		eg.Go(func() error {
			// Get a buffer from the pool
			buf := u.bufferPool.Get().([]byte)
			defer u.bufferPool.Put(buf)

			// Read part of the file into the buffer
			read, err := file.ReadAt(buf, offset)
			if err != nil && err != io.EOF {
				return fmt.Errorf("part %d: %w", partNum, err)
			}

			// Upload the part to S3
			partResp, err := u.s3api.UploadPart(egCtx, &s3.UploadPartInput{
				Bucket:     &req.Bucket,
				Key:        &req.Key,
				UploadId:   resp.UploadId,
				PartNumber: &partNum,
				Body:       bytes.NewReader(buf[:read]),
			})
			if err != nil {
				return fmt.Errorf("uploadPart #%d: %w", partNum, err)
			}

			// Store part metadata for final completion request
			completedParts[partNum-1] = s3types.CompletedPart{
				ETag:       partResp.ETag,
				PartNumber: &partNum,
			}
			return nil
		})
	}

	// Wait for all parts to be uploaded
	if err := eg.Wait(); err != nil {
		_, err := u.s3api.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   &req.Bucket,
			Key:      &req.Key,
			UploadId: resp.UploadId,
		})
		if err != nil {
			return err
		}
		return fmt.Errorf("multipartUpload: %w", err)
	}

	// Complete the multipart upload
	_, err = u.s3api.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: &req.Bucket,
		Key:    &req.Key,
		UploadId: resp.UploadId,
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	return err
}
