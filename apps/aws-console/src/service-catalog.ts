/** Represents an AWS service from the /api/services endpoint. */
export interface AWSServiceInfo {
  name: string;
  actions: number;
  healthy: boolean;
}

/**
 * Maps AWS category names to the signing/service names that belong in each.
 * Matches the real AWS Console category grouping.
 */
export const AWS_SERVICE_CATEGORIES: Record<string, string[]> = {
  Compute: [
    'lambda', 'ec2', 'ecs', 'eks', 'batch', 'elasticbeanstalk',
    'apprunner', 'lightsail', 'autoscaling', 'applicationautoscaling',
  ],
  Storage: [
    's3', 's3tables', 'glacier', 'elasticfilesystem', 'backup',
  ],
  Database: [
    'dynamodb', 'rds', 'elasticache', 'neptune', 'docdb', 'redshift',
    'memorydb', 'dax', 'keyspaces', 'qldb', 'timestreamwrite',
  ],
  Networking: [
    'route53', 'route53resolver', 'cloudfront', 'elasticloadbalancing',
    'apigateway', 'appsync', 'globalaccelerator', 'servicediscovery',
  ],
  Security: [
    'iam', 'kms', 'secretsmanager', 'acm', 'acmpca', 'wafv2', 'wafregional',
    'shield', 'guardduty', 'securityhub', 'inspector', 'cognito',
    'verifiedpermissions', 'ssoadmin', 'identitystore', 'ram', 'sts',
  ],
  Integration: [
    'sqs', 'sns', 'eventbridge', 'stepfunctions', 'mq', 'pipes',
    'kinesis', 'firehose', 'kinesisanalytics', 'kafka', 'scheduler',
  ],
  Management: [
    'cloudwatch', 'cloudwatchlogs', 'cloudformation', 'cloudtrail',
    'config', 'ssm', 'organizations', 'account', 'tagging',
    'resourcegroups', 'support', 'ce', 'cloudcontrol', 'servicecatalog',
  ],
  'Developer Tools': [
    'codebuild', 'codepipeline', 'codedeploy', 'codecommit',
    'codeartifact', 'codeconnections', 'xray', 'fis',
  ],
  'AI & ML': [
    'sagemaker', 'bedrock', 'comprehend', 'rekognition', 'textract',
    'transcribe', 'translate', 'polly', 'lex',
  ],
  Analytics: [
    'athena', 'glue', 'elasticmapreduce', 'lakeformation',
    'quicksight', 'opensearch', 'es',
  ],
  'Containers & Serverless': [
    'ecr', 'ecrpublic',
  ],
  IoT: [
    'iot', 'iotdata', 'iotwireless',
  ],
  'Media & Migration': [
    'mediaconvert', 'dms', 'transfer',
  ],
};

/** Reverse map: service name → category */
const serviceToCategoryMap: Record<string, string> = {};
for (const [category, services] of Object.entries(AWS_SERVICE_CATEGORIES)) {
  for (const svc of services) {
    serviceToCategoryMap[svc] = category;
  }
}

/** Get the AWS Console category for a service name. */
export function getCategory(serviceName: string): string {
  return serviceToCategoryMap[serviceName] || 'Other';
}

/**
 * Group services by AWS Console category.
 * Returns a map of category → services in that category.
 * Empty categories are omitted.
 */
export function categorizeServices(
  services: AWSServiceInfo[],
): Record<string, AWSServiceInfo[]> {
  const groups: Record<string, AWSServiceInfo[]> = {};

  for (const svc of services) {
    const category = getCategory(svc.name);
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(svc);
  }

  // Sort services within each category by name
  for (const category of Object.keys(groups)) {
    groups[category].sort((a, b) => a.name.localeCompare(b.name));
  }

  return groups;
}

/**
 * Filter services by search query (case-insensitive substring match on name).
 */
export function filterServices(
  services: AWSServiceInfo[],
  query: string,
): AWSServiceInfo[] {
  if (!query) return services;
  const q = query.toLowerCase();
  return services.filter((s) => s.name.toLowerCase().includes(q));
}

/** Human-friendly display name for a service. */
export function displayName(serviceName: string): string {
  const overrides: Record<string, string> = {
    s3: 'S3',
    dynamodb: 'DynamoDB',
    ec2: 'EC2',
    ecs: 'ECS',
    eks: 'EKS',
    ecr: 'ECR',
    ecrpublic: 'ECR Public',
    iam: 'IAM',
    kms: 'KMS',
    sqs: 'SQS',
    sns: 'SNS',
    ses: 'SES',
    rds: 'RDS',
    ssm: 'SSM',
    sts: 'STS',
    acm: 'ACM',
    acmpca: 'ACM PCA',
    dax: 'DAX',
    dms: 'DMS',
    mq: 'MQ',
    ram: 'RAM',
    fis: 'FIS',
    swf: 'SWF',
    ce: 'Cost Explorer',
    wafv2: 'WAF v2',
    wafregional: 'WAF Regional',
    apigateway: 'API Gateway',
    appsync: 'AppSync',
    apprunner: 'App Runner',
    appconfig: 'AppConfig',
    cloudformation: 'CloudFormation',
    cloudfront: 'CloudFront',
    cloudtrail: 'CloudTrail',
    cloudwatch: 'CloudWatch',
    cloudwatchlogs: 'CloudWatch Logs',
    cloudcontrol: 'Cloud Control',
    codecommit: 'CodeCommit',
    codebuild: 'CodeBuild',
    codepipeline: 'CodePipeline',
    codedeploy: 'CodeDeploy',
    codeartifact: 'CodeArtifact',
    codeconnections: 'CodeConnections',
    cognito: 'Cognito',
    docdb: 'DocumentDB',
    elasticache: 'ElastiCache',
    elasticbeanstalk: 'Elastic Beanstalk',
    elasticfilesystem: 'EFS',
    elasticloadbalancing: 'ELB',
    elasticmapreduce: 'EMR',
    eventbridge: 'EventBridge',
    firehose: 'Firehose',
    guardduty: 'GuardDuty',
    identitystore: 'Identity Store',
    iot: 'IoT Core',
    iotdata: 'IoT Data',
    iotwireless: 'IoT Wireless',
    kafka: 'MSK',
    kinesisanalytics: 'Kinesis Analytics',
    lakeformation: 'Lake Formation',
    managedblockchain: 'Managed Blockchain',
    mediaconvert: 'MediaConvert',
    memorydb: 'MemoryDB',
    neptune: 'Neptune',
    opensearch: 'OpenSearch',
    organizations: 'Organizations',
    pinpoint: 'Pinpoint',
    quicksight: 'QuickSight',
    rekognition: 'Rekognition',
    resourcegroups: 'Resource Groups',
    route53: 'Route 53',
    route53resolver: 'Route 53 Resolver',
    sagemaker: 'SageMaker',
    secretsmanager: 'Secrets Manager',
    securityhub: 'Security Hub',
    serverlessrepo: 'Serverless Repo',
    servicecatalog: 'Service Catalog',
    servicediscovery: 'Cloud Map',
    ssoadmin: 'SSO Admin',
    stepfunctions: 'Step Functions',
    timestreamwrite: 'Timestream',
    verifiedpermissions: 'Verified Permissions',
    globalaccelerator: 'Global Accelerator',
  };
  return overrides[serviceName] || serviceName.charAt(0).toUpperCase() + serviceName.slice(1);
}
