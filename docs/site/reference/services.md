# AWS Service Compatibility

CloudMock supports 98 AWS services across two implementation tiers.

## Tier 1: Full Implementation (25 services)

These services have hand-crafted implementations with complete API coverage, realistic behavior, and cross-service integration.

| Service | Key Actions | Notes |
|---------|-------------|-------|
| **IAM** | CreateUser, CreateRole, CreatePolicy, AttachRolePolicy, PutRolePolicy, GetUser, ListUsers, ListRoles | Full policy evaluation engine |
| **STS** | AssumeRole, GetCallerIdentity, GetSessionToken | Returns realistic temporary credentials |
| **S3** | CreateBucket, PutObject, GetObject, DeleteObject, ListObjectsV2, HeadObject, CopyObject, PutBucketPolicy, PutBucketNotification | Multipart uploads, versioning, lifecycle rules, event notifications |
| **DynamoDB** | CreateTable, PutItem, GetItem, Query, Scan, UpdateItem, DeleteItem, BatchWriteItem, BatchGetItem, TransactWriteItems | GSI/LSI, conditional expressions, streams |
| **SQS** | CreateQueue, SendMessage, ReceiveMessage, DeleteMessage, GetQueueAttributes, PurgeQueue, SendMessageBatch | FIFO queues, dead letter queues, visibility timeout |
| **SNS** | CreateTopic, Subscribe, Publish, Unsubscribe, ListTopics, ListSubscriptions | SQS/Lambda/HTTP subscriptions, message filtering |
| **Lambda** | CreateFunction, Invoke, UpdateFunctionCode, UpdateFunctionConfiguration, GetFunction, ListFunctions, DeleteFunction | Node.js/Python runtimes, event source mappings, layers |
| **CloudWatch Logs** | CreateLogGroup, CreateLogStream, PutLogEvents, GetLogEvents, FilterLogEvents, DescribeLogGroups | Log retention, metric filters |
| **RDS** | CreateDBInstance, DescribeDBInstances, DeleteDBInstance, CreateDBCluster, ModifyDBInstance | MySQL/PostgreSQL emulation |
| **CloudFormation** | CreateStack, UpdateStack, DeleteStack, DescribeStacks, ListStacks, GetTemplate | Template validation, resource creation |
| **EC2** | RunInstances, DescribeInstances, TerminateInstances, CreateVpc, CreateSubnet, CreateSecurityGroup, AuthorizeSecurityGroupIngress | VPC networking, security groups, AMI management |
| **ECR** | CreateRepository, DescribeRepositories, GetAuthorizationToken, PutImage, BatchGetImage | Image manifest storage |
| **ECS** | CreateCluster, RegisterTaskDefinition, CreateService, RunTask, DescribeServices, ListTasks | Fargate and EC2 launch types |
| **Secrets Manager** | CreateSecret, GetSecretValue, PutSecretValue, DescribeSecret, DeleteSecret, ListSecrets | Automatic rotation scheduling |
| **SSM** | PutParameter, GetParameter, GetParametersByPath, DeleteParameter | Parameter Store with encryption |
| **Kinesis** | CreateStream, PutRecord, PutRecords, GetRecords, GetShardIterator, DescribeStream | Shard management, enhanced fan-out |
| **Firehose** | CreateDeliveryStream, PutRecord, PutRecordBatch, DescribeDeliveryStream | S3 destination delivery |
| **EventBridge** | PutRule, PutTargets, PutEvents, DescribeRule, ListRules | Rule matching, target invocation |
| **Step Functions** | CreateStateMachine, StartExecution, DescribeExecution, GetExecutionHistory | ASL execution, Choice/Wait/Parallel states |
| **API Gateway** | CreateRestApi, CreateResource, PutMethod, CreateDeployment, GetRestApis | REST API with Lambda integration |
| **KMS** | CreateKey, Encrypt, Decrypt, GenerateDataKey, DescribeKey, ListKeys | Symmetric/asymmetric key support |
| **Cognito** | CreateUserPool, AdminCreateUser, InitiateAuth, SignUp, ConfirmSignUp | User pools, app clients, JWT tokens |
| **SES** | SendEmail, SendRawEmail, VerifyEmailIdentity, ListIdentities | Email capture (viewable in DevTools) |
| **ACM** | RequestCertificate, DescribeCertificate, ListCertificates, DeleteCertificate | Self-signed certificate generation |
| **Route 53** | CreateHostedZone, ChangeResourceRecordSets, ListResourceRecordSets, GetHostedZone | DNS record management |

## Tier 2: Behavioral Emulation (73 services)

These services are generated from AWS API models. They support CRUD operations with correct request/response shapes, ARN generation, and cross-service references. They store resources in-memory and respond with realistic payloads.

### Compute & Containers

| Service | Key Actions |
|---------|-------------|
| Auto Scaling | CreateAutoScalingGroup, DescribeAutoScalingGroups, UpdateAutoScalingGroup, DeleteAutoScalingGroup |
| App Runner | CreateService, DescribeService, ListServices, DeleteService |
| Batch | CreateComputeEnvironment, CreateJobQueue, SubmitJob, DescribeJobs |
| Elastic Beanstalk | CreateApplication, CreateEnvironment, DescribeApplications, DescribeEnvironments |
| EKS | CreateCluster, DescribeCluster, ListClusters, DeleteCluster |
| Lightsail | CreateInstances, GetInstances, DeleteInstance |
| Serverless Repo | CreateApplication, GetApplication, ListApplications |

### Databases & Storage

| Service | Key Actions |
|---------|-------------|
| DocumentDB | CreateDBCluster, DescribeDBClusters, DeleteDBCluster |
| DMS | CreateReplicationInstance, CreateReplicationTask, StartReplicationTask |
| ElastiCache | CreateCacheCluster, DescribeCacheClusters, CreateReplicationGroup, DeleteCacheCluster |
| Glacier | CreateVault, DescribeVault, ListVaults, DeleteVault |
| Keyspaces | CreateKeyspace, CreateTable, GetKeyspace, GetTable |
| Lake Formation | RegisterResource, GrantPermissions, GetDataLakeSettings |
| MemoryDB | CreateCluster, DescribeClusters, DeleteCluster |
| Neptune | CreateDBInstance, DescribeDBInstances, DeleteDBInstance |
| OpenSearch | CreateDomain, DescribeDomain, DeleteDomain |
| Redshift | CreateCluster, DescribeClusters, DeleteCluster |
| S3 Tables | CreateTable, GetTable, ListTables, DeleteTable |
| Timestream Write | CreateDatabase, CreateTable, WriteRecords |

### Networking & Content Delivery

| Service | Key Actions |
|---------|-------------|
| CloudFront | CreateDistribution, GetDistribution, ListDistributions, UpdateDistribution |
| ELB/ALB | CreateLoadBalancer, CreateTargetGroup, DescribeLoadBalancers, DescribeTargetGroups |
| Route 53 Resolver | CreateResolverEndpoint, ListResolverEndpoints |
| Transfer Family | CreateServer, DescribeServer, ListServers |
| VPC Lattice | CreateService, CreateTargetGroup, ListServices |

### Security & Identity

| Service | Key Actions |
|---------|-------------|
| ACM PCA | CreateCertificateAuthority, IssueCertificate, GetCertificate |
| Identity Store | CreateUser, DescribeUser, ListUsers |
| Organizations | CreateOrganization, CreateAccount, ListAccounts |
| RAM | CreateResourceShare, AssociateResourceShare |
| Shield | CreateProtection, DescribeProtection, ListProtections |
| SSO Admin | CreatePermissionSet, AttachManagedPolicyToPermissionSet |
| Verified Permissions | CreatePolicyStore, CreatePolicy, IsAuthorized |
| WAF Regional | CreateWebACL, GetWebACL, ListWebACLs |
| WAFv2 | CreateWebACL, GetWebACL, ListWebACLs, UpdateWebACL |

### Application Integration

| Service | Key Actions |
|---------|-------------|
| AppConfig | CreateApplication, CreateEnvironment, CreateConfigurationProfile, StartDeployment |
| AppSync | CreateGraphqlApi, CreateResolver, ListGraphqlApis |
| EventBridge Pipes | CreatePipe, DescribePipe, ListPipes |
| EventBridge Scheduler | CreateSchedule, GetSchedule, ListSchedules |
| MQ | CreateBroker, DescribeBroker, ListBrokers |
| Service Discovery | CreateService, RegisterInstance, DiscoverInstances |
| SWF | RegisterDomain, RegisterWorkflowType, StartWorkflowExecution |

### Analytics & ML

| Service | Key Actions |
|---------|-------------|
| Athena | StartQueryExecution, GetQueryExecution, GetQueryResults |
| EMR | RunJobFlow, ListClusters, DescribeCluster, TerminateJobFlows |
| Glue | CreateDatabase, CreateTable, GetTable, CreateJob, StartJobRun |
| Kinesis Analytics | CreateApplication, DescribeApplication, StartApplication |
| Managed Blockchain | CreateNetwork, CreateMember, ListNetworks |
| SageMaker | CreateTrainingJob, CreateEndpoint, DescribeTrainingJob |
| Textract | DetectDocumentText, AnalyzeDocument |
| Transcribe | StartTranscriptionJob, GetTranscriptionJob |

### IoT

| Service | Key Actions |
|---------|-------------|
| IoT Core | CreateThing, DescribeThing, ListThings, AttachThingPrincipal |
| IoT Data | Publish, GetThingShadow, UpdateThingShadow |
| IoT Wireless | CreateWirelessDevice, GetWirelessDevice |

### Developer Tools

| Service | Key Actions |
|---------|-------------|
| CodeArtifact | CreateDomain, CreateRepository, ListRepositories |
| CodeBuild | CreateProject, StartBuild, BatchGetBuilds |
| CodeCommit | CreateRepository, GetRepository, ListRepositories |
| CodeConnections | CreateConnection, GetConnection, ListConnections |
| CodeDeploy | CreateApplication, CreateDeploymentGroup, CreateDeployment |
| CodePipeline | CreatePipeline, GetPipeline, ListPipelines, StartPipelineExecution |
| Cloud Control | CreateResource, GetResource, ListResources, UpdateResource |
| FIS | CreateExperimentTemplate, StartExperiment, GetExperiment |

### Management & Governance

| Service | Key Actions |
|---------|-------------|
| Account | GetContactInformation, PutContactInformation |
| Application Auto Scaling | RegisterScalableTarget, PutScalingPolicy |
| Backup | CreateBackupPlan, StartBackupJob, ListBackupPlans |
| CloudTrail | CreateTrail, StartLogging, LookupEvents |
| CloudWatch | PutMetricData, GetMetricData, DescribeAlarms, PutMetricAlarm |
| Config | PutConfigRule, DescribeConfigRules, PutConfigurationRecorder |
| Cost Explorer | GetCostAndUsage, GetCostForecast |
| Resource Groups | CreateGroup, ListGroups, GetGroup |
| Support | DescribeTrustedAdvisorChecks, CreateCase |
| Tagging | TagResources, UntagResources, GetResources |

### Media & Communication

| Service | Key Actions |
|---------|-------------|
| Amplify | CreateApp, CreateBranch, ListApps |
| MediaConvert | CreateJob, GetJob, ListJobs |
| Pinpoint | CreateApp, CreateCampaign, GetApp |

### AI & ML

| Service | Key Actions |
|---------|-------------|
| Bedrock | ListFoundationModels, InvokeModel |
| MWAA (Airflow) | CreateEnvironment, GetEnvironment, ListEnvironments |

## Feature Support by Tier

| Feature | Tier 1 | Tier 2 |
|---------|--------|--------|
| Full API coverage | Yes | Core CRUD only |
| Realistic error responses | Yes | Yes |
| IAM policy enforcement | Yes | Yes |
| Cross-service events | Yes | Basic |
| Resource persistence | Yes | Yes |
| ARN generation | Yes | Yes |
| Request capture in DevTools | Yes | Yes |
| Chaos fault injection | Yes | Yes |
| Cost estimation | Yes | Estimated |
| CloudFormation support | Native | Via Cloud Control |
| Terraform provider | Native | Via schema extraction |
| Pulumi provider | Native | Via schema extraction |
