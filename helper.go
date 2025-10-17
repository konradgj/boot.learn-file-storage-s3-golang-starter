package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	pClient := s3.NewPresignClient(s3Client)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	pObject, err := pClient.PresignGetObject(context.Background(), input, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return pObject.URL, nil
}
