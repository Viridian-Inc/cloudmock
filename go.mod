module github.com/neureaux/cloudmock

go 1.26.1

require (
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.32.12
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.37
	github.com/aws/aws-sdk-go-v2/service/account v1.30.5
	github.com/aws/aws-sdk-go-v2/service/acm v1.38.1
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.46.12
	github.com/aws/aws-sdk-go-v2/service/amplify v1.38.14
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.39.1
	github.com/aws/aws-sdk-go-v2/service/appconfig v1.43.13
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.41.14
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.39.14
	github.com/aws/aws-sdk-go-v2/service/appsync v1.53.5
	github.com/aws/aws-sdk-go-v2/service/athena v1.57.4
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.65.0
	github.com/aws/aws-sdk-go-v2/service/backup v1.54.11
	github.com/aws/aws-sdk-go-v2/service/batch v1.63.2
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.58.0
	github.com/aws/aws-sdk-go-v2/service/cloudcontrol v1.29.13
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.9
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.61.0
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.55.9
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.55.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.66.0
	github.com/aws/aws-sdk-go-v2/service/codeartifact v1.38.21
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.13
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.33.12
	github.com/aws/aws-sdk-go-v2/service/codeconnections v1.10.20
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.35.13
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.21
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.59.3
	github.com/aws/aws-sdk-go-v2/service/configservice v1.62.1
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.63.6
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.62.0
	github.com/aws/aws-sdk-go-v2/service/datasync v1.58.2
	github.com/aws/aws-sdk-go-v2/service/dax v1.29.16
	github.com/aws/aws-sdk-go-v2/service/docdb v1.48.13
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.296.2
	github.com/aws/aws-sdk-go-v2/service/ecr v1.56.2
	github.com/aws/aws-sdk-go-v2/service/ecs v1.76.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.81.2
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.52.0
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.34.2
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.23
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.40.0
	github.com/aws/aws-sdk-go-v2/service/emr v1.59.0
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.45.23
	github.com/aws/aws-sdk-go-v2/service/firehose v1.42.13
	github.com/aws/aws-sdk-go-v2/service/fis v1.37.20
	github.com/aws/aws-sdk-go-v2/service/glacier v1.32.6
	github.com/aws/aws-sdk-go-v2/service/glue v1.139.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.7
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.36.5
	github.com/aws/aws-sdk-go-v2/service/iot v1.72.5
	github.com/aws/aws-sdk-go-v2/service/iotdataplane v1.32.21
	github.com/aws/aws-sdk-go-v2/service/iotwireless v1.54.9
	github.com/aws/aws-sdk-go-v2/service/kafka v1.49.2
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.43.5
	github.com/aws/aws-sdk-go-v2/service/kinesisanalytics v1.30.23
	github.com/aws/aws-sdk-go-v2/service/kms v1.50.4
	github.com/aws/aws-sdk-go-v2/service/lakeformation v1.47.6
	github.com/aws/aws-sdk-go-v2/service/lambda v1.88.5
	github.com/aws/aws-sdk-go-v2/service/managedblockchain v1.31.21
	github.com/aws/aws-sdk-go-v2/service/mediaconvert v1.89.1
	github.com/aws/aws-sdk-go-v2/service/memorydb v1.33.14
	github.com/aws/aws-sdk-go-v2/service/mq v1.34.19
	github.com/aws/aws-sdk-go-v2/service/mwaa v1.39.22
	github.com/aws/aws-sdk-go-v2/service/neptune v1.44.3
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.64.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.0
	github.com/aws/aws-sdk-go-v2/service/pinpoint v1.39.21
	github.com/aws/aws-sdk-go-v2/service/pipes v1.23.20
	github.com/aws/aws-sdk-go-v2/service/ram v1.36.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.117.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.62.5
	github.com/aws/aws-sdk-go-v2/service/rekognition v1.51.21
	github.com/aws/aws-sdk-go-v2/service/resourcegroups v1.33.24
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.31.10
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.5
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.42.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.97.1
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.238.0
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.17.22
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.5
	github.com/aws/aws-sdk-go-v2/service/serverlessapplicationrepository v1.30.12
	github.com/aws/aws-sdk-go-v2/service/servicediscovery v1.39.26
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.22
	github.com/aws/aws-sdk-go-v2/service/sfn v1.40.10
	github.com/aws/aws-sdk-go-v2/service/shield v1.34.21
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.15
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.24
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.4
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.37.6
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.9
	github.com/aws/aws-sdk-go-v2/service/support v1.31.21
	github.com/aws/aws-sdk-go-v2/service/swf v1.33.16
	github.com/aws/aws-sdk-go-v2/service/textract v1.40.20
	github.com/aws/aws-sdk-go-v2/service/timestreamwrite v1.35.20
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.54.4
	github.com/aws/aws-sdk-go-v2/service/transfer v1.69.5
	github.com/aws/aws-sdk-go-v2/service/verifiedpermissions v1.32.1
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.30.21
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.71.3
	github.com/aws/aws-sdk-go-v2/service/xray v1.36.21
	github.com/docker/docker v28.5.2+incompatible
	github.com/docker/go-connections v0.6.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/pprof v0.0.0-20260302011040-a15ffb7f9dcc
	github.com/google/uuid v1.6.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.40.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/marcboeker/go-duckdb v1.8.5
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.67.5
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.41.0
	github.com/tidwall/btree v1.8.1
	go.opentelemetry.io/otel v1.42.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.42.0
	go.opentelemetry.io/otel/sdk v1.42.0
	golang.org/x/crypto v0.48.0
	google.golang.org/grpc v1.79.2
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/apache/arrow-go/v18 v18.1.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.32.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.17 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v25.1.24+incompatible // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/hc-install v0.9.3 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.25.0 // indirect
	github.com/hashicorp/terraform-json v0.27.2 // indirect
	github.com/hashicorp/terraform-plugin-go v0.31.0 // indirect
	github.com/hashicorp/terraform-plugin-log v0.10.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.4.0 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/shirou/gopsutil/v4 v4.26.2 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/testcontainers/testcontainers-go/modules/postgres v0.41.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zclconf/go-cty v1.17.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/exp v0.0.0-20250128182459-e0ece0dbea4c // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260109210033-bd525da824e2 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
)
