package tier1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type s3Suite struct{}

func NewS3Suite() harness.Suite { return &s3Suite{} }
func (s *s3Suite) Name() string { return "s3" }
func (s *s3Suite) Tier() int    { return 1 }

func (s *s3Suite) Operations() []harness.Operation {
	bucket := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	key := "bench-object.txt"
	body := []byte("benchmark payload data for testing")

	// sharedClient caches one S3 client per endpoint so that HTTP connections are
	// reused across warm-phase iterations. Creating a new client per Run call causes
	// a TCP handshake on every iteration, masking actual handler latency.
	var (
		clientMu  sync.Mutex
		clientMap = make(map[string]*s3.Client)
	)
	getClient := func(endpoint string) (*s3.Client, error) {
		clientMu.Lock()
		defer clientMu.Unlock()
		if c, ok := clientMap[endpoint]; ok {
			return c, nil
		}
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		c := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
			o.UsePathStyle = true
		})
		clientMap[endpoint] = c
		return c, nil
	}

	createBucket := func(ctx context.Context, client *s3.Client) error {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		return err
	}

	putObject := func(ctx context.Context, client *s3.Client) error {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(body),
		})
		return err
	}

	deleteObject := func(ctx context.Context, client *s3.Client, k string) error {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(k),
		})
		return err
	}

	deleteBucket := func(ctx context.Context, client *s3.Client) error {
		_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateBucket",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.CreateBucket(ctx, &s3.CreateBucketInput{
					Bucket: aws.String(bucket),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateBucketOutput")}
			},
		},
		{
			Name: "PutObject",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				return createBucket(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   bytes.NewReader(body),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := deleteObject(ctx, client, key); err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutObjectOutput")}
			},
		},
		{
			Name: "GetObject",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := createBucket(ctx, client); err != nil {
					return err
				}
				return putObject(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := deleteObject(ctx, client, key); err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				findings := []harness.Finding{harness.CheckNotNil(resp, "GetObjectOutput")}
				if resp == nil {
					return findings
				}
				out, ok := resp.(*s3.GetObjectOutput)
				if !ok || out.Body == nil {
					findings = append(findings, harness.Finding{
						Field:    "Body",
						Expected: string(body),
						Actual:   "<nil or wrong type>",
						Grade:    harness.GradeFail,
					})
					return findings
				}
				data, err := io.ReadAll(out.Body)
				out.Body.Close()
				if err != nil {
					findings = append(findings, harness.Finding{
						Field:    "Body",
						Expected: string(body),
						Actual:   fmt.Sprintf("<read error: %v>", err),
						Grade:    harness.GradeFail,
					})
					return findings
				}
				if string(data) != string(body) {
					findings = append(findings, harness.Finding{
						Field:    "Body",
						Expected: string(body),
						Actual:   string(data),
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Body",
						Expected: string(body),
						Actual:   string(data),
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
		},
		{
			Name: "ListObjects",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := createBucket(ctx, client); err != nil {
					return err
				}
				return putObject(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
					Bucket: aws.String(bucket),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := deleteObject(ctx, client, key); err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				findings := []harness.Finding{harness.CheckNotNil(resp, "ListObjectsV2Output")}
				if resp == nil {
					return findings
				}
				out, ok := resp.(*s3.ListObjectsV2Output)
				if !ok {
					findings = append(findings, harness.Finding{
						Field:    "Contents",
						Expected: ">=1 item",
						Actual:   "<wrong type>",
						Grade:    harness.GradeFail,
					})
					return findings
				}
				if len(out.Contents) >= 1 {
					findings = append(findings, harness.Finding{
						Field:    "Contents",
						Expected: ">=1 item",
						Actual:   fmt.Sprintf("%d items", len(out.Contents)),
						Grade:    harness.GradePass,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Contents",
						Expected: ">=1 item",
						Actual:   "0 items",
						Grade:    harness.GradeFail,
					})
				}
				return findings
			},
		},
		{
			Name: "CopyObject",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := createBucket(ctx, client); err != nil {
					return err
				}
				return putObject(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.CopyObject(ctx, &s3.CopyObjectInput{
					Bucket:     aws.String(bucket),
					Key:        aws.String("copied-" + key),
					CopySource: aws.String(fmt.Sprintf("%s/%s", bucket, key)),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := deleteObject(ctx, client, key); err != nil {
					return err
				}
				if err := deleteObject(ctx, client, "copied-"+key); err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CopyObjectOutput")}
			},
		},
		{
			Name: "DeleteObject",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				if err := createBucket(ctx, client); err != nil {
					return err
				}
				return putObject(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				return deleteBucket(ctx, client)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteObjectOutput")}
			},
		},
		{
			Name: "DeleteBucket",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := getClient(endpoint)
				if err != nil {
					return err
				}
				return createBucket(ctx, client)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := getClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteBucket(ctx, &s3.DeleteBucketInput{
					Bucket: aws.String(bucket),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteBucketOutput")}
			},
		},
	}
}
