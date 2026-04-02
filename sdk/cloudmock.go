// Package sdk provides an in-process AWS mock that routes AWS SDK v2 calls
// directly to CloudMock's gateway handler without any HTTP/TCP overhead.
//
// Usage:
//
//	cm := sdk.New()
//	defer cm.Close()
//	cfg := cm.Config()
//	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })
package sdk

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	dynamodbsvc "github.com/neureaux/cloudmock/services/dynamodb"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
	stssvc "github.com/neureaux/cloudmock/services/sts"
)

// Option configures a CloudMock instance.
type Option func(*options)

type options struct {
	profile   string
	iamMode   string
	region    string
	accountID string
}

// WithProfile sets the service profile ("minimal" or "standard").
func WithProfile(p string) Option { return func(o *options) { o.profile = p } }

// WithIAMMode sets the IAM enforcement mode ("none", "log", "enforce").
func WithIAMMode(m string) Option { return func(o *options) { o.iamMode = m } }

// WithRegion sets the AWS region for the mock.
func WithRegion(r string) Option { return func(o *options) { o.region = r } }

// WithAccountID sets the AWS account ID for the mock.
func WithAccountID(id string) Option { return func(o *options) { o.accountID = id } }

// CloudMock is an in-process AWS mock. All AWS SDK calls routed through its
// Config() go directly to the gateway handler with zero network overhead.
type CloudMock struct {
	handler   http.Handler
	transport *inProcessTransport
	cfg       aws.Config
}

// New creates a new in-process CloudMock instance. By default it uses a
// minimal profile (S3, STS, DynamoDB, SQS) with IAM disabled for speed.
func New(opts ...Option) *CloudMock {
	o := &options{
		profile:   "minimal",
		iamMode:   "none",
		region:    "us-east-1",
		accountID: "000000000000",
	}
	for _, opt := range opts {
		opt(o)
	}

	// Build a lightweight config — no file I/O, no env parsing.
	c := &config.Config{
		Region:    o.region,
		AccountID: o.accountID,
		Profile:   o.profile,
		IAM: config.IAMConfig{
			Mode:          o.iamMode,
			RootAccessKey: "test",
			RootSecretKey: "test",
		},
	}

	bus := eventbus.NewBus()
	registry := routing.NewRegistry()

	// Register core services — these cover the vast majority of test use cases.
	registry.Register(s3svc.NewWithBus(bus))
	registry.Register(stssvc.New(c.AccountID))
	registry.Register(dynamodbsvc.New(c.AccountID, c.Region))
	registry.Register(sqssvc.New(c.AccountID, c.Region))

	// Build the gateway handler.
	var gw *gateway.Gateway
	if o.iamMode == "none" {
		gw = gateway.New(c, registry)
	} else {
		store := iampkg.NewStore(c.AccountID)
		store.InitRoot(c.IAM.RootAccessKey, c.IAM.RootSecretKey)
		engine := iampkg.NewEngine()
		gw = gateway.NewWithIAM(c, registry, store, engine)
	}
	gw.SetEventBus(bus)

	transport := newInProcessTransport(gw)

	// Build the aws.Config that the caller will pass to SDK clients.
	// Static credentials avoid per-call credential resolution and keep the
	// Authorization header needed for service routing. RetryMaxAttempts=1
	// disables the retry middleware, which otherwise clones the full
	// *http.Request per attempt.
	awsConfig, err := awscfg.LoadDefaultConfig(context.Background(),
		awscfg.WithRegion(o.region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		awscfg.WithHTTPClient(&http.Client{Transport: transport}),
		awscfg.WithRetryMaxAttempts(1),
	)
	if err != nil {
		// Should never happen with static config, but don't swallow it.
		panic("cloudmock/sdk: failed to build aws.Config: " + err.Error())
	}

	// Set a base endpoint so all services route through our transport.
	awsConfig.BaseEndpoint = aws.String("http://cloudmock.local")

	return &CloudMock{
		handler:   gw,
		transport: transport,
		cfg:       awsConfig,
	}
}

// Config returns an aws.Config that routes all SDK calls through CloudMock in-process.
func (cm *CloudMock) Config() aws.Config {
	return cm.cfg
}

// Close releases resources held by this CloudMock instance.
func (cm *CloudMock) Close() {
	// Currently a no-op — reserved for future cleanup (DynamoDB TTL reapers, etc.)
}
