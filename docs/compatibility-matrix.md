# Compatibility Matrix

All 98 services supported by cloudmock, grouped by tier.

---

## Tier 1 — Full Emulation (25 services)

| Service | AWS Service Name | Protocol | Actions |
|---------|-----------------|----------|---------|
| S3 | `s3` | REST-XML | ListBuckets, CreateBucket, DeleteBucket, HeadBucket, PutObject, GetObject, DeleteObject, HeadObject, ListObjectsV2, CopyObject |
| DynamoDB | `dynamodb` | JSON | CreateTable, DeleteTable, DescribeTable, ListTables, PutItem, GetItem, DeleteItem, UpdateItem, Query, Scan, BatchGetItem, BatchWriteItem |
| SQS | `sqs` | Query | CreateQueue, DeleteQueue, ListQueues, GetQueueUrl, GetQueueAttributes, SetQueueAttributes, SendMessage, ReceiveMessage, DeleteMessage, PurgeQueue, ChangeMessageVisibility, SendMessageBatch, DeleteMessageBatch |
| SNS | `sns` | Query | CreateTopic, DeleteTopic, ListTopics, GetTopicAttributes, SetTopicAttributes, Subscribe, Unsubscribe, ListSubscriptions, ListSubscriptionsByTopic, Publish, TagResource, UntagResource |
| STS | `sts` | Query | GetCallerIdentity, AssumeRole, GetSessionToken |
| KMS | `kms` | JSON | CreateKey, DescribeKey, ListKeys, Encrypt, Decrypt, CreateAlias, ListAliases, EnableKey, DisableKey, ScheduleKeyDeletion |
| Secrets Manager | `secretsmanager` | JSON | CreateSecret, GetSecretValue, PutSecretValue, UpdateSecret, DeleteSecret, RestoreSecret, DescribeSecret, ListSecrets, TagResource, UntagResource |
| SSM | `ssm` | JSON | PutParameter, GetParameter, GetParameters, GetParametersByPath, DeleteParameter, DeleteParameters, DescribeParameters |
| CloudWatch | `monitoring` | Query | PutMetricData, GetMetricData, ListMetrics, PutMetricAlarm, DescribeAlarms, DeleteAlarms, SetAlarmState, DescribeAlarmsForMetric, TagResource, UntagResource, ListTagsForResource |
| CloudWatch Logs | `logs` | JSON | CreateLogGroup, DeleteLogGroup, DescribeLogGroups, CreateLogStream, DeleteLogStream, DescribeLogStreams, PutLogEvents, GetLogEvents, FilterLogEvents, PutRetentionPolicy, DeleteRetentionPolicy, TagLogGroup, UntagLogGroup, ListTagsLogGroup |
| EventBridge | `events` | JSON | CreateEventBus, DeleteEventBus, DescribeEventBus, ListEventBuses, PutRule, DeleteRule, DescribeRule, ListRules, PutTargets, RemoveTargets, ListTargetsByRule, PutEvents, EnableRule, DisableRule, TagResource, UntagResource, ListTagsForResource |
| Cognito | `cognito-idp` | JSON | CreateUserPool, DeleteUserPool, DescribeUserPool, ListUserPools, CreateUserPoolClient, DescribeUserPoolClient, ListUserPoolClients, AdminCreateUser, AdminGetUser, AdminDeleteUser, AdminSetUserPassword, SignUp, InitiateAuth, AdminConfirmSignUp |
| API Gateway | `apigateway` | REST-JSON | CreateRestApi, GetRestApis, GetRestApi, DeleteRestApi, CreateResource, GetResources, DeleteResource, PutMethod, GetMethod, PutIntegration, CreateDeployment, GetDeployments, CreateStage, GetStages |
| Step Functions | `states` | JSON | CreateStateMachine, DeleteStateMachine, DescribeStateMachine, ListStateMachines, UpdateStateMachine, StartExecution, DescribeExecution, StopExecution, ListExecutions, GetExecutionHistory, TagResource, UntagResource, ListTagsForResource |
| Route 53 | `route53` | REST-XML | CreateHostedZone, ListHostedZones, GetHostedZone, DeleteHostedZone, ChangeResourceRecordSets, ListResourceRecordSets |
| RDS | `rds` | Query | CreateDBInstance, DeleteDBInstance, DescribeDBInstances, ModifyDBInstance, CreateDBCluster, DeleteDBCluster, DescribeDBClusters, CreateDBSnapshot, DeleteDBSnapshot, DescribeDBSnapshots, CreateDBSubnetGroup, DescribeDBSubnetGroups, DeleteDBSubnetGroup, AddTagsToResource, RemoveTagsFromResource, ListTagsForResource |
| ECR | `ecr` | JSON | CreateRepository, DeleteRepository, DescribeRepositories, ListImages, BatchGetImage, PutImage, BatchDeleteImage, GetAuthorizationToken, DescribeImageScanFindings, TagResource, UntagResource, ListTagsForResource |
| ECS | `ecs` | JSON | CreateCluster, DeleteCluster, DescribeClusters, ListClusters, RegisterTaskDefinition, DeregisterTaskDefinition, DescribeTaskDefinition, ListTaskDefinitions, CreateService, DeleteService, DescribeServices, ListServices, UpdateService, RunTask, StopTask, DescribeTasks, ListTasks, TagResource, UntagResource, ListTagsForResource |
| SES | `email` | Query | SendEmail, SendRawEmail, VerifyEmailIdentity, ListIdentities, DeleteIdentity, GetIdentityVerificationAttributes, ListVerifiedEmailAddresses |
| Kinesis | `kinesis` | JSON | CreateStream, DeleteStream, DescribeStream, ListStreams, PutRecord, PutRecords, GetShardIterator, GetRecords, IncreaseStreamRetentionPeriod, DecreaseStreamRetentionPeriod, AddTagsToStream, RemoveTagsFromStream, ListTagsForStream |
| Data Firehose | `firehose` | JSON | CreateDeliveryStream, DeleteDeliveryStream, DescribeDeliveryStream, ListDeliveryStreams, PutRecord, PutRecordBatch, UpdateDestination, TagDeliveryStream, UntagDeliveryStream, ListTagsForDeliveryStream |
| CloudFormation | `cloudformation` | Query | CreateStack, DeleteStack, DescribeStacks, ListStacks, DescribeStackResources, DescribeStackEvents, GetTemplate, ValidateTemplate, ListExports, CreateChangeSet, DescribeChangeSet, ExecuteChangeSet, DeleteChangeSet |
| IAM | `iam` | Embedded | CreateUser, GetUser, CreateAccessKey, AttachUserPolicy, GetUserPolicies (via seed file / admin API) |
| AppSync | `appsync` | REST-JSON | CreateGraphqlApi, GetGraphqlApi, ListGraphqlApis, UpdateGraphqlApi, DeleteGraphqlApi, CreateDataSource, GetDataSource, ListDataSources, UpdateDataSource, DeleteDataSource, CreateResolver, GetResolver, ListResolvers, UpdateResolver, DeleteResolver, CreateFunction, GetFunction, ListFunctions, UpdateFunction, DeleteFunction, CreateApiKey, ListApiKeys, UpdateApiKey, DeleteApiKey, TagResource, UntagResource, ListTagsForResource |
| Lambda | `lambda` | REST-JSON | CreateFunction, DeleteFunction, GetFunction, ListFunctions, UpdateFunctionCode, UpdateFunctionConfiguration, InvokeFunction (stub), AddPermission, RemovePermission, CreateEventSourceMapping, ListEventSourceMappings, TagResource, UntagResource |

---

## Tier 2 — CRUD Stubs (73 services)

Tier 2 services support basic CRUD: create, describe/get, list, delete. Some include update. All resources are stored in memory; no business logic is executed.

### Query Protocol

| # | Service | AWS Service Name | Actions |
|---|---------|-----------------|---------|
| 1 | Auto Scaling | `autoscaling` | CreateAutoScalingGroup, DescribeAutoScalingGroups, DeleteAutoScalingGroup, UpdateAutoScalingGroup |
| 2 | ELB / ALB | `elasticloadbalancing` | CreateLoadBalancer, DescribeLoadBalancers, DeleteLoadBalancer, CreateTargetGroup, DescribeTargetGroups |
| 3 | Elastic Beanstalk | `elasticbeanstalk` | CreateApplication, DescribeApplications, DeleteApplication, CreateEnvironment, DescribeEnvironments |
| 4 | ElastiCache | `elasticache` | CreateCacheCluster, DescribeCacheClusters, DeleteCacheCluster, CreateReplicationGroup |
| 5 | Redshift | `redshift` | CreateCluster, DescribeClusters, DeleteCluster |
| 6 | Neptune | `neptune` | CreateDBInstance, DescribeDBInstances, DeleteDBInstance |
| 7 | Elasticsearch | `es` | CreateElasticsearchDomain, DescribeElasticsearchDomain, DeleteElasticsearchDomain |
| 8 | EMR | `elasticmapreduce` | RunJobFlow, ListClusters, DescribeCluster, TerminateJobFlows |
| 9 | EC2 | `ec2` | RunInstances, DescribeInstances, TerminateInstances, CreateVpc, DescribeVpcs, CreateSecurityGroup, DescribeSecurityGroups, CreateSubnet, DescribeSubnets |
| 10 | Shield | `shield` | CreateProtection, DescribeProtection, ListProtections, DeleteProtection |
| 11 | WAF Regional | `waf-regional` | CreateWebACL, GetWebACL, ListWebACLs, DeleteWebACL |
| 30 | DocumentDB | `docdb` | CreateDBCluster, DescribeDBClusters, DeleteDBCluster |

### JSON Protocol

| # | Service | AWS Service Name | Actions |
|---|---------|-----------------|---------|
| 12 | ACM | `acm` | RequestCertificate, DescribeCertificate, ListCertificates, DeleteCertificate |
| 13 | ACM PCA | `acm-pca` | CreateCertificateAuthority, DescribeCertificateAuthority, ListCertificateAuthorities |
| 14 | AppConfig | `appconfig` | CreateApplication, GetApplication, ListApplications, DeleteApplication |
| 15 | Application Auto Scaling | `application-autoscaling` | RegisterScalableTarget, DescribeScalableTargets, DeregisterScalableTarget |
| 17 | Athena | `athena` | StartQueryExecution, GetQueryExecution, ListQueryExecutions, StopQueryExecution |
| 18 | Backup | `backup` | CreateBackupPlan, DescribeBackupJob, ListBackupPlans, DeleteBackupPlan |
| 21 | CodeBuild | `codebuild` | CreateProject, BatchGetProjects, ListProjects, DeleteProject |
| 22 | CodeCommit | `codecommit` | CreateRepository, GetRepository, ListRepositories, DeleteRepository |
| 23 | CodeDeploy | `codedeploy` | CreateApplication, GetApplication, ListApplications, DeleteApplication |
| 24 | CodePipeline | `codepipeline` | CreatePipeline, GetPipeline, ListPipelines, DeletePipeline |
| 26 | CodeConnections | `codeconnections` | CreateConnection, GetConnection, ListConnections, DeleteConnection |
| 27 | Config | `config` | PutConfigRule, DescribeConfigRules, DeleteConfigRule |
| 28 | Cost Explorer | `ce` | GetCostAndUsage |
| 29 | DMS | `dms` | CreateReplicationInstance, DescribeReplicationInstances, DeleteReplicationInstance |
| 33 | Glue | `glue` | CreateDatabase, GetDatabase, GetDatabases, DeleteDatabase, CreateTable, GetTable, GetTables, DeleteTable |
| 34 | Identity Store | `identitystore` | CreateUser, DescribeUser, ListUsers, DeleteUser |
| 38 | Lake Formation | `lakeformation` | RegisterResource, DescribeResource, ListResources, DeregisterResource |
| 42 | MemoryDB | `memorydb` | CreateCluster, DescribeClusters, DeleteCluster |
| 45 | Organizations | `organizations` | CreateOrganization, DescribeOrganization, ListAccounts |
| 47 | RAM | `ram` | CreateResourceShare, GetResourceShares, DeleteResourceShare |
| 49 | Resource Groups Tagging API | `tagging` | TagResources, UntagResources, GetResources |
| 50 | SageMaker | `sagemaker` | CreateNotebookInstance, DescribeNotebookInstance, ListNotebookInstances, DeleteNotebookInstance |
| 52 | Service Discovery | `servicediscovery` | CreateService, GetService, ListServices, DeleteService |
| 53 | SWF | `swf` | RegisterDomain, ListDomains, DescribeDomain, DeprecateDomain |
| 54 | SSO Admin | `sso-admin` | CreatePermissionSet, DescribePermissionSet, ListPermissionSets, DeletePermissionSet |
| 55 | Support | `support` | CreateCase, DescribeCases, DescribeTrustedAdvisorChecks |
| 56 | Textract | `textract` | DetectDocumentText, AnalyzeDocument, StartDocumentTextDetection, GetDocumentTextDetection |
| 57 | Timestream | `timestream-write` | CreateDatabase, DescribeDatabase, ListDatabases, DeleteDatabase, CreateTable, DescribeTable, ListTables, DeleteTable |
| 58 | Transcribe | `transcribe` | StartTranscriptionJob, GetTranscriptionJob, ListTranscriptionJobs, DeleteTranscriptionJob |
| 59 | Transfer | `transfer` | CreateServer, DescribeServer, ListServers, DeleteServer |
| 60 | Verified Permissions | `verifiedpermissions` | CreatePolicyStore, GetPolicyStore, ListPolicyStores, DeletePolicyStore |
| 63 | Cloud Control | `cloudcontrol` | CreateResource, GetResource, ListResources, DeleteResource, UpdateResource |
| 65 | CloudTrail | `cloudtrail` | CreateTrail, GetTrail, DescribeTrails, DeleteTrail |
| 70 | Kinesis Analytics (Flink) | `kinesisanalytics` | CreateApplication, DescribeApplication, ListApplications, DeleteApplication |
| 72 | X-Ray | `xray` | PutTraceSegments, GetTraceSummaries, BatchGetTraces |

### REST-JSON Protocol

| # | Service | AWS Service Name | Actions |
|---|---------|-----------------|---------|
| 19 | Batch | `batch` | CreateJobQueue, ListJobQueues, GetJobQueue, DeleteJobQueue |
| 20 | Bedrock | `bedrock` | CreateModelCustomizationJob, ListModelCustomizationJobs, GetModelCustomizationJob, DeleteModelCustomizationJob |
| 25 | CodeArtifact | `codeartifact` | CreateRepository, ListRepositories, GetRepository, DeleteRepository |
| 31 | EKS | `eks` | CreateCluster, ListClusters, GetCluster, DeleteCluster |
| 32 | FIS | `fis` | CreateExperimentTemplate, ListExperimentTemplates, GetExperimentTemplate, DeleteExperimentTemplate |
| 35 | IoT | `iot` | CreateThing, ListThings, GetThing, DeleteThing |
| 36 | IoT Data | `iot-data` | CreateThingShadow, ListThingShadows, GetThingShadow, DeleteThingShadow |
| 37 | IoT Wireless | `iot-wireless` | CreateWirelessDevice, ListWirelessDevices, GetWirelessDevice, DeleteWirelessDevice |
| 39 | Managed Blockchain | `managedblockchain` | CreateNetwork, ListNetworks, GetNetwork, DeleteNetwork |
| 40 | MSK (Kafka) | `kafka` | CreateCluster, ListClusters, GetCluster, DeleteCluster |
| 41 | MWAA (Airflow) | `airflow` | CreateEnvironment, ListEnvironments, GetEnvironment, DeleteEnvironment |
| 43 | MQ | `mq` | CreateBroker, ListBrokers, GetBroker, DeleteBroker |
| 44 | OpenSearch | `opensearch` | CreateDomain, ListDomains, GetDomain, DeleteDomain |
| 46 | Pinpoint | `pinpoint` | CreateApp, ListApps, GetApp, DeleteApp |
| 48 | Resource Groups | `resource-groups` | CreateGroup, ListGroups, GetGroup, DeleteGroup |
| 51 | Serverless App Repo | `serverlessrepo` | CreateApplication, ListApplications, GetApplication, DeleteApplication |
| 61 | Amplify | `amplify` | CreateApp, ListApps, GetApp, DeleteApp |
| 62 | Account Management | `account` | CreateContactInformation, ListContactInformations, GetContactInformation, DeleteContactInformation |
| 66 | Glacier | `glacier` | CreateVault, ListVaults, GetVault, DeleteVault |
| 67 | MediaConvert | `mediaconvert` | CreateJob, ListJobs, GetJob, DeleteJob |
| 68 | EventBridge Pipes | `pipes` | CreatePipe, ListPipes, GetPipe, DeletePipe |
| 69 | EventBridge Scheduler | `scheduler` | CreateSchedule, ListSchedules, GetSchedule, DeleteSchedule |
| 71 | S3 Tables | `s3tables` | CreateTableBucket, ListTableBuckets, GetTableBucket, DeleteTableBucket |
| 73 | WAFv2 | `wafv2` | CreateWebACL, ListWebACLs, GetWebACL, DeleteWebACL |
| 74 | Route 53 Resolver | `route53resolver` | CreateResolverEndpoint, ListResolverEndpoints, GetResolverEndpoint, DeleteResolverEndpoint |

### REST-XML Protocol

| # | Service | AWS Service Name | Actions |
|---|---------|-----------------|---------|
| 64 | CloudFront | `cloudfront` | CreateDistribution, GetDistribution, ListDistributions, DeleteDistribution |

---

## Summary

| Tier | Count | Description |
|------|-------|-------------|
| Tier 1 | 25 | Full emulation with business logic |
| Tier 2 | 73 | CRUD stubs with in-memory resource storage |
| **Total** | **98** | |
