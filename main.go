package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func AWSConfig(ctx context.Context, accessKeyID, secretAccessKey, region, endpoint string) (aws.Config, error) {
	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
		if len(endpoint) > 0 {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
				SigningRegion:     region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	optFns := []func(*config.LoadOptions) error{config.WithCredentialsProvider(creds)}
	if len(endpoint) > 0 {
		optFns = append(
			optFns,
			config.WithDefaultRegion(region),
			config.WithEndpointResolverWithOptions(resolver),
		)
	}

	awsConfig, err := config.LoadDefaultConfig(
		ctx,
		optFns...,
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("aws config error: %w", err)
	}
	return awsConfig, err
}

func main() {
	err := exec.Command("docker", "compose", "-f", "compose", "up", "--build", "-d").Run()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	ctx := context.Background()

	cfg, _ := AWSConfig(ctx, "dummy", "dummy", "ap-northeast-1", "http://localhost:4566")

	s3client := s3.NewFromConfig(cfg)

	bucket := "dummy-bucket"
	contentType := "text/plain"

	uploader := manager.NewUploader(s3client)

	copyMethod := func(objectKey, dstObjectKey string) error {
		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(objectKey),
			Body:        bytes.NewBuffer([]byte("test")),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			log.Fatalf("error s3client upload: %+v\n", err)
		}

		copySource := url.QueryEscape(fmt.Sprintf("%s/%s", bucket, objectKey))
		_, err = s3client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String(copySource),
			Key:        aws.String(dstObjectKey),
		})
		if err != nil {
			return err
		}
		log.Printf("success copy object %s to %s\n", objectKey, dstObjectKey)
		return nil
	}

	err = copyMethod("test_original.txt", "test copy.txt")
	if err != nil {
		log.Fatalf("error s3client copy object: %+v\n", err)
		return
	}

	err = copyMethod("test original.txt", "test copy.txt")
	if err != nil {
		log.Fatalf("error s3client copy object: %+v\n", err)
		return
	}
}
