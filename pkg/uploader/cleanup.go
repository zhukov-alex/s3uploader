package uploader

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func CleanupBucket(ctx context.Context, client *s3.Client, bucket string) error {
	// List of objects in bucket
	listOutput, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
	})
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	// Delete every object in the bucket
	for _, obj := range listOutput.Contents {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", *obj.Key, err)
		}
	}
	return nil
}
