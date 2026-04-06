import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();
const domain = config.require("domain");
const containerImage = config.require("containerImage");
const desiredCount = config.getNumber("desiredCount") ?? 2;
const cpu = config.get("cpu") ?? "512";
const memory = config.get("memory") ?? "1024";

const stack = pulumi.getStack();
const prefix = `cloudmock-${stack}`;

// --- Networking ---

const vpc = new aws.ec2.Vpc(`${prefix}-vpc`, {
    cidrBlock: "10.0.0.0/16",
    enableDnsHostnames: true,
    enableDnsSupport: true,
    tags: { Name: `${prefix}-vpc` },
});

const publicSubnetA = new aws.ec2.Subnet(`${prefix}-public-a`, {
    vpcId: vpc.id,
    cidrBlock: "10.0.1.0/24",
    availabilityZone: "us-east-1a",
    mapPublicIpOnLaunch: true,
    tags: { Name: `${prefix}-public-a` },
});

const publicSubnetB = new aws.ec2.Subnet(`${prefix}-public-b`, {
    vpcId: vpc.id,
    cidrBlock: "10.0.2.0/24",
    availabilityZone: "us-east-1b",
    mapPublicIpOnLaunch: true,
    tags: { Name: `${prefix}-public-b` },
});

const igw = new aws.ec2.InternetGateway(`${prefix}-igw`, {
    vpcId: vpc.id,
    tags: { Name: `${prefix}-igw` },
});

const routeTable = new aws.ec2.RouteTable(`${prefix}-rt`, {
    vpcId: vpc.id,
    routes: [{
        cidrBlock: "0.0.0.0/0",
        gatewayId: igw.id,
    }],
    tags: { Name: `${prefix}-rt` },
});

new aws.ec2.RouteTableAssociation(`${prefix}-rta-a`, {
    subnetId: publicSubnetA.id,
    routeTableId: routeTable.id,
});

new aws.ec2.RouteTableAssociation(`${prefix}-rta-b`, {
    subnetId: publicSubnetB.id,
    routeTableId: routeTable.id,
});

// --- Security Groups ---

const albSg = new aws.ec2.SecurityGroup(`${prefix}-alb-sg`, {
    vpcId: vpc.id,
    description: "ALB - allow HTTP/HTTPS inbound",
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
    description: "ECS tasks - allow traffic from ALB",
    ingress: [
        { protocol: "tcp", fromPort: 4500, toPort: 4500, securityGroups: [albSg.id] },
        { protocol: "tcp", fromPort: 4566, toPort: 4566, securityGroups: [albSg.id] },
    ],
    egress: [
        { protocol: "-1", fromPort: 0, toPort: 0, cidrBlocks: ["0.0.0.0/0"] },
    ],
    tags: { Name: `${prefix}-ecs-sg` },
});

// --- ALB ---

const alb = new aws.lb.LoadBalancer(`${prefix}-alb`, {
    internal: false,
    loadBalancerType: "application",
    securityGroups: [albSg.id],
    subnets: [publicSubnetA.id, publicSubnetB.id],
    tags: { Name: `${prefix}-alb` },
});

const dashboardTg = new aws.lb.TargetGroup(`${prefix}-dashboard-tg`, {
    port: 4500,
    protocol: "HTTP",
    targetType: "ip",
    vpcId: vpc.id,
    healthCheck: {
        path: "/api/health",
        port: "4500",
        protocol: "HTTP",
        healthyThreshold: 2,
        unhealthyThreshold: 3,
        interval: 15,
        timeout: 5,
    },
    tags: { Name: `${prefix}-dashboard-tg` },
});

const gatewayTg = new aws.lb.TargetGroup(`${prefix}-gateway-tg`, {
    port: 4566,
    protocol: "HTTP",
    targetType: "ip",
    vpcId: vpc.id,
    healthCheck: {
        path: "/",
        port: "4566",
        protocol: "HTTP",
        healthyThreshold: 2,
        unhealthyThreshold: 3,
        interval: 15,
        timeout: 5,
    },
    tags: { Name: `${prefix}-gateway-tg` },
});

new aws.lb.Listener(`${prefix}-http-listener`, {
    loadBalancerArn: alb.arn,
    port: 80,
    protocol: "HTTP",
    defaultActions: [{
        type: "forward",
        targetGroupArn: dashboardTg.arn,
    }],
});

// --- IAM ---

const taskExecRole = new aws.iam.Role(`${prefix}-exec-role`, {
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

new aws.iam.RolePolicyAttachment(`${prefix}-exec-policy`, {
    role: taskExecRole.name,
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

// --- CloudWatch Logs ---

const logGroup = new aws.cloudwatch.LogGroup(`${prefix}-logs`, {
    name: `/ecs/${prefix}`,
    retentionInDays: 14,
    tags: { Name: `${prefix}-logs` },
});

// --- ECS ---

const cluster = new aws.ecs.Cluster(`${prefix}-cluster`, {
    name: `${prefix}-cluster`,
    tags: { Name: `${prefix}-cluster` },
});

const taskDefinition = new aws.ecs.TaskDefinition(`${prefix}-task`, {
    family: `${prefix}-task`,
    networkMode: "awsvpc",
    requiresCompatibilities: ["FARGATE"],
    cpu: cpu,
    memory: memory,
    executionRoleArn: taskExecRole.arn,
    taskRoleArn: taskRole.arn,
    containerDefinitions: pulumi.jsonStringify([{
        name: "cloudmock",
        image: containerImage,
        essential: true,
        portMappings: [
            { containerPort: 4500, protocol: "tcp" },
            { containerPort: 4566, protocol: "tcp" },
        ],
        environment: [
            { name: "CLOUDMOCK_PROFILE", value: "full" },
            { name: "CLOUDMOCK_SAAS_ENABLED", value: "true" },
            { name: "CLOUDMOCK_AUTH_ENABLED", value: "true" },
        ],
        logConfiguration: {
            logDriver: "awslogs",
            options: {
                "awslogs-group": `/ecs/${prefix}`,
                "awslogs-region": "us-east-1",
                "awslogs-stream-prefix": "cloudmock",
            },
        },
        healthCheck: {
            command: ["CMD-SHELL", "curl -f http://localhost:4500/api/health || exit 1"],
            interval: 15,
            timeout: 5,
            retries: 3,
            startPeriod: 30,
        },
    }]),
    tags: { Name: `${prefix}-task` },
});

const service = new aws.ecs.Service(`${prefix}-service`, {
    name: `${prefix}-service`,
    cluster: cluster.arn,
    taskDefinition: taskDefinition.arn,
    desiredCount: desiredCount,
    launchType: "FARGATE",
    networkConfiguration: {
        subnets: [publicSubnetA.id, publicSubnetB.id],
        securityGroups: [ecsSg.id],
        assignPublicIp: true,
    },
    loadBalancers: [
        {
            targetGroupArn: dashboardTg.arn,
            containerName: "cloudmock",
            containerPort: 4500,
        },
    ],
    deploymentConfiguration: {
        maximumPercent: 200,
        minimumHealthyPercent: 100,
    },
    tags: { Name: `${prefix}-service` },
});

// --- Route 53 ---

const zone = new aws.route53.Zone(`${prefix}-zone`, {
    name: domain,
    tags: { Name: `${prefix}-zone` },
});

new aws.route53.Record(`${prefix}-a-record`, {
    zoneId: zone.zoneId,
    name: domain,
    type: "A",
    aliases: [{
        name: alb.dnsName,
        zoneId: alb.zoneId,
        evaluateTargetHealth: true,
    }],
});

new aws.route53.Record(`${prefix}-wildcard`, {
    zoneId: zone.zoneId,
    name: `*.${domain}`,
    type: "A",
    aliases: [{
        name: alb.dnsName,
        zoneId: alb.zoneId,
        evaluateTargetHealth: true,
    }],
});

// --- CloudWatch Alarms ---

new aws.cloudwatch.MetricAlarm(`${prefix}-high-cpu`, {
    alarmName: `${prefix}-high-cpu`,
    comparisonOperator: "GreaterThanThreshold",
    evaluationPeriods: 2,
    metricName: "CPUUtilization",
    namespace: "AWS/ECS",
    period: 300,
    statistic: "Average",
    threshold: 80,
    dimensions: {
        ClusterName: cluster.name,
        ServiceName: service.name,
    },
    tags: { Name: `${prefix}-high-cpu` },
});

new aws.cloudwatch.MetricAlarm(`${prefix}-unhealthy-hosts`, {
    alarmName: `${prefix}-unhealthy-hosts`,
    comparisonOperator: "GreaterThanThreshold",
    evaluationPeriods: 1,
    metricName: "UnHealthyHostCount",
    namespace: "AWS/ApplicationELB",
    period: 60,
    statistic: "Maximum",
    threshold: 0,
    dimensions: {
        TargetGroup: dashboardTg.arnSuffix,
        LoadBalancer: alb.arnSuffix,
    },
    tags: { Name: `${prefix}-unhealthy-hosts` },
});

// --- Exports ---

export const vpcId = vpc.id;
export const clusterName = cluster.name;
export const serviceName = service.name;
export const albDnsName = alb.dnsName;
export const albArn = alb.arn;
export const dashboardUrl = pulumi.interpolate`http://${alb.dnsName}`;
export const gatewayEndpoint = pulumi.interpolate`http://${alb.dnsName}:4566`;
export const logGroupName = logGroup.name;
export const domainName = domain;
export const nameServers = zone.nameServers;
