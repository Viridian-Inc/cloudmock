import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const config = new pulumi.Config();
const domain = config.require("domain");
const containerImage = config.require("containerImage");
const dbInstanceClass = config.get("dbInstanceClass") ?? "db.t4g.micro";
const dbAllocatedStorage = config.getNumber("dbAllocatedStorage") ?? 20;
const desiredCount = config.getNumber("desiredCount") ?? 1;
const cpu = config.get("cpu") ?? "256";
const memory = config.get("memory") ?? "512";

const stack = pulumi.getStack();
const prefix = `cm-cloud-${stack}`;

// ---------------------------------------------------------------------------
// Networking
// ---------------------------------------------------------------------------

const vpc = new aws.ec2.Vpc(`${prefix}-vpc`, {
    cidrBlock: "10.1.0.0/16",
    enableDnsHostnames: true,
    enableDnsSupport: true,
    tags: { Name: `${prefix}-vpc` },
});

const publicSubnetA = new aws.ec2.Subnet(`${prefix}-pub-a`, {
    vpcId: vpc.id,
    cidrBlock: "10.1.1.0/24",
    availabilityZone: "us-east-1a",
    mapPublicIpOnLaunch: true,
    tags: { Name: `${prefix}-pub-a` },
});

const publicSubnetB = new aws.ec2.Subnet(`${prefix}-pub-b`, {
    vpcId: vpc.id,
    cidrBlock: "10.1.2.0/24",
    availabilityZone: "us-east-1b",
    mapPublicIpOnLaunch: true,
    tags: { Name: `${prefix}-pub-b` },
});

const privateSubnetA = new aws.ec2.Subnet(`${prefix}-priv-a`, {
    vpcId: vpc.id,
    cidrBlock: "10.1.10.0/24",
    availabilityZone: "us-east-1a",
    tags: { Name: `${prefix}-priv-a` },
});

const privateSubnetB = new aws.ec2.Subnet(`${prefix}-priv-b`, {
    vpcId: vpc.id,
    cidrBlock: "10.1.11.0/24",
    availabilityZone: "us-east-1b",
    tags: { Name: `${prefix}-priv-b` },
});

const igw = new aws.ec2.InternetGateway(`${prefix}-igw`, {
    vpcId: vpc.id,
    tags: { Name: `${prefix}-igw` },
});

const publicRt = new aws.ec2.RouteTable(`${prefix}-pub-rt`, {
    vpcId: vpc.id,
    routes: [{ cidrBlock: "0.0.0.0/0", gatewayId: igw.id }],
    tags: { Name: `${prefix}-pub-rt` },
});

new aws.ec2.RouteTableAssociation(`${prefix}-rta-pub-a`, {
    subnetId: publicSubnetA.id,
    routeTableId: publicRt.id,
});
new aws.ec2.RouteTableAssociation(`${prefix}-rta-pub-b`, {
    subnetId: publicSubnetB.id,
    routeTableId: publicRt.id,
});

// ---------------------------------------------------------------------------
// Security Groups
// ---------------------------------------------------------------------------

const albSg = new aws.ec2.SecurityGroup(`${prefix}-alb-sg`, {
    vpcId: vpc.id,
    description: "ALB: HTTP/HTTPS inbound from anywhere",
    ingress: [
        { protocol: "tcp", fromPort: 80, toPort: 80, cidrBlocks: ["0.0.0.0/0"] },
        { protocol: "tcp", fromPort: 443, toPort: 443, cidrBlocks: ["0.0.0.0/0"] },
    ],
    egress: [
        { protocol: "-1", fromPort: 0, toPort: 0, cidrBlocks: ["0.0.0.0/0"] },
    ],
    tags: { Name: `${prefix}-alb-sg` },
});

const ecsSg = new aws.ec2.SecurityGroup(`${prefix}-ecs-sg`, {
    vpcId: vpc.id,
    description: "ECS ingest tasks: traffic from ALB only",
    ingress: [
        { protocol: "tcp", fromPort: 8080, toPort: 8080, securityGroups: [albSg.id] },
    ],
    egress: [
        { protocol: "-1", fromPort: 0, toPort: 0, cidrBlocks: ["0.0.0.0/0"] },
    ],
    tags: { Name: `${prefix}-ecs-sg` },
});

const dbSg = new aws.ec2.SecurityGroup(`${prefix}-db-sg`, {
    vpcId: vpc.id,
    description: "RDS: PostgreSQL from ECS tasks only",
    ingress: [
        { protocol: "tcp", fromPort: 5432, toPort: 5432, securityGroups: [ecsSg.id] },
    ],
    egress: [
        { protocol: "-1", fromPort: 0, toPort: 0, cidrBlocks: ["0.0.0.0/0"] },
    ],
    tags: { Name: `${prefix}-db-sg` },
});

// ---------------------------------------------------------------------------
// RDS PostgreSQL (TimescaleDB-compatible)
// ---------------------------------------------------------------------------

const dbSubnetGroup = new aws.rds.SubnetGroup(`${prefix}-db-subnets`, {
    subnetIds: [privateSubnetA.id, privateSubnetB.id],
    tags: { Name: `${prefix}-db-subnets` },
});

const dbPassword = config.requireSecret("dbPassword");

const db = new aws.rds.Instance(`${prefix}-db`, {
    engine: "postgres",
    engineVersion: "16.4",
    instanceClass: dbInstanceClass,
    allocatedStorage: dbAllocatedStorage,
    dbName: "cloudmock",
    username: "cloudmock",
    password: dbPassword,
    dbSubnetGroupName: dbSubnetGroup.name,
    vpcSecurityGroupIds: [dbSg.id],
    publiclyAccessible: false,
    skipFinalSnapshot: true, // dogfood — no retention needed
    storageEncrypted: true,
    backupRetentionPeriod: 1,
    tags: { Name: `${prefix}-db` },
});

const databaseUrl = pulumi.interpolate`postgres://cloudmock:${dbPassword}@${db.address}:5432/cloudmock?sslmode=require`;

// ---------------------------------------------------------------------------
// ECR Repository
// ---------------------------------------------------------------------------

const repo = new aws.ecr.Repository(`${prefix}-repo`, {
    name: "cloudmock-cloud",
    imageTagMutability: "MUTABLE",
    imageScanningConfiguration: { scanOnPush: true },
    tags: { Name: `${prefix}-repo` },
});

// ---------------------------------------------------------------------------
// ALB
// ---------------------------------------------------------------------------

const alb = new aws.lb.LoadBalancer(`${prefix}-alb`, {
    internal: false,
    loadBalancerType: "application",
    securityGroups: [albSg.id],
    subnets: [publicSubnetA.id, publicSubnetB.id],
    tags: { Name: `${prefix}-alb` },
});

const tg = new aws.lb.TargetGroup(`${prefix}-tg`, {
    port: 8080,
    protocol: "HTTP",
    targetType: "ip",
    vpcId: vpc.id,
    healthCheck: {
        path: "/healthz",
        port: "8080",
        protocol: "HTTP",
        healthyThreshold: 2,
        unhealthyThreshold: 3,
        interval: 15,
        timeout: 5,
    },
    tags: { Name: `${prefix}-tg` },
});

new aws.lb.Listener(`${prefix}-http`, {
    loadBalancerArn: alb.arn,
    port: 80,
    protocol: "HTTP",
    defaultActions: [{ type: "forward", targetGroupArn: tg.arn }],
});

// ---------------------------------------------------------------------------
// IAM
// ---------------------------------------------------------------------------

const execRole = new aws.iam.Role(`${prefix}-exec-role`, {
    assumeRolePolicy: JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Effect: "Allow",
            Principal: { Service: "ecs-tasks.amazonaws.com" },
            Action: "sts:AssumeRole",
        }],
    }),
    tags: { Name: `${prefix}-exec-role` },
});

new aws.iam.RolePolicyAttachment(`${prefix}-exec-attach`, {
    role: execRole.name,
    policyArn: "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy",
});

const taskRole = new aws.iam.Role(`${prefix}-task-role`, {
    assumeRolePolicy: JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Effect: "Allow",
            Principal: { Service: "ecs-tasks.amazonaws.com" },
            Action: "sts:AssumeRole",
        }],
    }),
    tags: { Name: `${prefix}-task-role` },
});

// ---------------------------------------------------------------------------
// CloudWatch Logs
// ---------------------------------------------------------------------------

const logGroup = new aws.cloudwatch.LogGroup(`${prefix}-logs`, {
    name: `/ecs/${prefix}`,
    retentionInDays: 14,
    tags: { Name: `${prefix}-logs` },
});

// ---------------------------------------------------------------------------
// ECS Cluster + Service
// ---------------------------------------------------------------------------

const cluster = new aws.ecs.Cluster(`${prefix}-cluster`, {
    name: `${prefix}-cluster`,
    tags: { Name: `${prefix}-cluster` },
});

const taskDef = new aws.ecs.TaskDefinition(`${prefix}-task`, {
    family: `${prefix}-task`,
    networkMode: "awsvpc",
    requiresCompatibilities: ["FARGATE"],
    cpu: cpu,
    memory: memory,
    executionRoleArn: execRole.arn,
    taskRoleArn: taskRole.arn,
    containerDefinitions: pulumi.all([databaseUrl, logGroup.name]).apply(([dbUrl, lgName]) =>
        JSON.stringify([{
            name: "ingest",
            image: containerImage,
            essential: true,
            portMappings: [{ containerPort: 8080, protocol: "tcp" }],
            environment: [
                { name: "DATABASE_URL", value: dbUrl },
                { name: "ADDR", value: ":8080" },
                { name: "MIGRATIONS_PATH", value: "/migrations" },
            ],
            logConfiguration: {
                logDriver: "awslogs",
                options: {
                    "awslogs-group": lgName,
                    "awslogs-region": "us-east-1",
                    "awslogs-stream-prefix": "ingest",
                },
            },
            healthCheck: {
                command: ["CMD-SHELL", "wget -qO- http://localhost:8080/healthz || exit 1"],
                interval: 15,
                timeout: 5,
                retries: 3,
                startPeriod: 30,
            },
        }]),
    ),
    tags: { Name: `${prefix}-task` },
});

const svc = new aws.ecs.Service(`${prefix}-svc`, {
    name: `${prefix}-svc`,
    cluster: cluster.arn,
    taskDefinition: taskDef.arn,
    desiredCount: desiredCount,
    launchType: "FARGATE",
    networkConfiguration: {
        subnets: [publicSubnetA.id, publicSubnetB.id],
        securityGroups: [ecsSg.id],
        assignPublicIp: true,
    },
    loadBalancers: [{
        targetGroupArn: tg.arn,
        containerName: "ingest",
        containerPort: 8080,
    }],
    forceNewDeployment: true,
    tags: { Name: `${prefix}-svc` },
});

// ---------------------------------------------------------------------------
// Exports
// ---------------------------------------------------------------------------

export const vpcId = vpc.id;
export const clusterName = cluster.name;
export const serviceName = svc.name;
export const albDnsName = alb.dnsName;
export const ingestEndpoint = pulumi.interpolate`http://${alb.dnsName}`;
export const ecrRepositoryUrl = repo.repositoryUrl;
export const dbEndpoint = db.address;
export const logGroupName = logGroup.name;
