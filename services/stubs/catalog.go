package stubs

import (
	"strings"

	"github.com/neureaux/cloudmock/pkg/stub"
)

// AllModels returns ServiceModels for all 73 Tier 2 AWS services.
func AllModels() []*stub.ServiceModel {
	var models []*stub.ServiceModel

	// ── Query/XML Protocol services ──────────────────────────────────
	models = append(models, queryServices()...)

	// ── JSON Protocol services (X-Amz-Target) ────────────────────────
	models = append(models, jsonServices()...)

	// ── REST-JSON services (path-based routing) ──────────────────────
	models = append(models, restJSONServices()...)

	// ── REST-XML services ────────────────────────────────────────────
	models = append(models, restXMLServices()...)

	return models
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func f(name, typ string, required bool) stub.Field {
	return stub.Field{Name: name, Type: typ, Required: required}
}

func reqStr(name string) stub.Field { return f(name, "string", true) }
func optStr(name string) stub.Field { return f(name, "string", false) }

func createAction(name, resType, idField string, in, out []stub.Field) stub.Action {
	return stub.Action{Name: name, Type: "create", ResourceType: resType, IdField: idField, InputFields: in, OutputFields: out}
}

func describeAction(name, resType, idField string) stub.Action {
	return stub.Action{Name: name, Type: "describe", ResourceType: resType, IdField: idField, InputFields: []stub.Field{reqStr(idField)}}
}

func listAction(name, resType string) stub.Action {
	return stub.Action{Name: name, Type: "list", ResourceType: resType}
}

func deleteAction(name, resType, idField string) stub.Action {
	return stub.Action{Name: name, Type: "delete", ResourceType: resType, IdField: idField, InputFields: []stub.Field{reqStr(idField)}}
}

func updateAction(name, resType, idField string, extra []stub.Field) stub.Action {
	fields := []stub.Field{reqStr(idField)}
	fields = append(fields, extra...)
	return stub.Action{Name: name, Type: "update", ResourceType: resType, IdField: idField, InputFields: fields}
}

func otherAction(name, resType string) stub.Action {
	return stub.Action{Name: name, Type: "other", ResourceType: resType}
}

func rt(name, idField, arnPattern string, fields []stub.Field) stub.ResourceType {
	return stub.ResourceType{Name: name, IdField: idField, ArnPattern: arnPattern, Fields: fields}
}

// ─── Query protocol services ────────────────────────────────────────────────

func queryServices() []*stub.ServiceModel {
	return []*stub.ServiceModel{
		// 1. Auto Scaling
		{
			ServiceName: "autoscaling",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateAutoScalingGroup":    createAction("CreateAutoScalingGroup", "asg", "AutoScalingGroupName", []stub.Field{reqStr("AutoScalingGroupName"), reqStr("MinSize"), reqStr("MaxSize")}, []stub.Field{optStr("AutoScalingGroupName")}),
				"DescribeAutoScalingGroups": listAction("DescribeAutoScalingGroups", "asg"),
				"DeleteAutoScalingGroup":    deleteAction("DeleteAutoScalingGroup", "asg", "AutoScalingGroupName"),
				"UpdateAutoScalingGroup":    updateAction("UpdateAutoScalingGroup", "asg", "AutoScalingGroupName", []stub.Field{optStr("MinSize"), optStr("MaxSize")}),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"asg": rt("AutoScalingGroup", "AutoScalingGroupName", "arn:aws:autoscaling:{region}:{account}:autoScalingGroup/{id}", []stub.Field{optStr("AutoScalingGroupName"), optStr("MinSize"), optStr("MaxSize")}),
			},
		},
		// 2. ELB
		{
			ServiceName: "elasticloadbalancing",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateLoadBalancer":   createAction("CreateLoadBalancer", "lb", "LoadBalancerName", []stub.Field{reqStr("LoadBalancerName")}, []stub.Field{optStr("LoadBalancerName")}),
				"DescribeLoadBalancers": listAction("DescribeLoadBalancers", "lb"),
				"DeleteLoadBalancer":   deleteAction("DeleteLoadBalancer", "lb", "LoadBalancerName"),
				"CreateTargetGroup":    createAction("CreateTargetGroup", "tg", "TargetGroupName", []stub.Field{reqStr("TargetGroupName")}, []stub.Field{optStr("TargetGroupName")}),
				"DescribeTargetGroups": listAction("DescribeTargetGroups", "tg"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"lb": rt("LoadBalancer", "LoadBalancerName", "arn:aws:elasticloadbalancing:{region}:{account}:loadbalancer/{id}", []stub.Field{optStr("LoadBalancerName")}),
				"tg": rt("TargetGroup", "TargetGroupName", "arn:aws:elasticloadbalancing:{region}:{account}:targetgroup/{id}", []stub.Field{optStr("TargetGroupName")}),
			},
		},
		// 3. Elastic Beanstalk
		{
			ServiceName: "elasticbeanstalk",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateApplication":      createAction("CreateApplication", "app", "ApplicationName", []stub.Field{reqStr("ApplicationName")}, []stub.Field{optStr("ApplicationName")}),
				"DescribeApplications":   listAction("DescribeApplications", "app"),
				"DeleteApplication":      deleteAction("DeleteApplication", "app", "ApplicationName"),
				"CreateEnvironment":      createAction("CreateEnvironment", "env", "EnvironmentName", []stub.Field{reqStr("EnvironmentName"), reqStr("ApplicationName")}, []stub.Field{optStr("EnvironmentName")}),
				"DescribeEnvironments":   listAction("DescribeEnvironments", "env"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"app": rt("Application", "ApplicationName", "arn:aws:elasticbeanstalk:{region}:{account}:application/{id}", []stub.Field{optStr("ApplicationName")}),
				"env": rt("Environment", "EnvironmentName", "arn:aws:elasticbeanstalk:{region}:{account}:environment/{id}", []stub.Field{optStr("EnvironmentName"), optStr("ApplicationName")}),
			},
		},
		// 4. ElastiCache
		{
			ServiceName: "elasticache",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateCacheCluster":     createAction("CreateCacheCluster", "cluster", "CacheClusterId", []stub.Field{reqStr("CacheClusterId")}, []stub.Field{optStr("CacheClusterId")}),
				"DescribeCacheClusters":  listAction("DescribeCacheClusters", "cluster"),
				"DeleteCacheCluster":     deleteAction("DeleteCacheCluster", "cluster", "CacheClusterId"),
				"CreateReplicationGroup": createAction("CreateReplicationGroup", "replgroup", "ReplicationGroupId", []stub.Field{reqStr("ReplicationGroupId")}, []stub.Field{optStr("ReplicationGroupId")}),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cluster":   rt("CacheCluster", "CacheClusterId", "arn:aws:elasticache:{region}:{account}:cluster:{id}", []stub.Field{optStr("CacheClusterId")}),
				"replgroup": rt("ReplicationGroup", "ReplicationGroupId", "arn:aws:elasticache:{region}:{account}:replicationgroup:{id}", []stub.Field{optStr("ReplicationGroupId")}),
			},
		},
		// 5. Redshift
		{
			ServiceName: "redshift",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateCluster":    createAction("CreateCluster", "cluster", "ClusterIdentifier", []stub.Field{reqStr("ClusterIdentifier"), reqStr("NodeType")}, []stub.Field{optStr("ClusterIdentifier")}),
				"DescribeClusters": listAction("DescribeClusters", "cluster"),
				"DeleteCluster":    deleteAction("DeleteCluster", "cluster", "ClusterIdentifier"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cluster": rt("Cluster", "ClusterIdentifier", "arn:aws:redshift:{region}:{account}:cluster:{id}", []stub.Field{optStr("ClusterIdentifier"), optStr("NodeType")}),
			},
		},
		// 6. Neptune
		{
			ServiceName: "neptune",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateDBInstance":    createAction("CreateDBInstance", "dbinstance", "DBInstanceIdentifier", []stub.Field{reqStr("DBInstanceIdentifier"), reqStr("DBInstanceClass")}, []stub.Field{optStr("DBInstanceIdentifier")}),
				"DescribeDBInstances": listAction("DescribeDBInstances", "dbinstance"),
				"DeleteDBInstance":    deleteAction("DeleteDBInstance", "dbinstance", "DBInstanceIdentifier"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"dbinstance": rt("DBInstance", "DBInstanceIdentifier", "arn:aws:neptune:{region}:{account}:db:{id}", []stub.Field{optStr("DBInstanceIdentifier"), optStr("DBInstanceClass")}),
			},
		},
		// 7. Elasticsearch
		{
			ServiceName: "es",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateElasticsearchDomain":   createAction("CreateElasticsearchDomain", "domain", "DomainName", []stub.Field{reqStr("DomainName")}, []stub.Field{optStr("DomainName")}),
				"DescribeElasticsearchDomain": describeAction("DescribeElasticsearchDomain", "domain", "DomainName"),
				"DeleteElasticsearchDomain":   deleteAction("DeleteElasticsearchDomain", "domain", "DomainName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"domain": rt("Domain", "DomainName", "arn:aws:es:{region}:{account}:domain/{id}", []stub.Field{optStr("DomainName")}),
			},
		},
		// 8. EMR
		{
			ServiceName: "elasticmapreduce",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"RunJobFlow":        createAction("RunJobFlow", "cluster", "JobFlowId", []stub.Field{reqStr("Name")}, []stub.Field{optStr("Name")}),
				"ListClusters":      listAction("ListClusters", "cluster"),
				"DescribeCluster":   describeAction("DescribeCluster", "cluster", "JobFlowId"),
				"TerminateJobFlows": deleteAction("TerminateJobFlows", "cluster", "JobFlowId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cluster": rt("Cluster", "JobFlowId", "arn:aws:elasticmapreduce:{region}:{account}:cluster/{id}", []stub.Field{optStr("Name")}),
			},
		},
		// 10. Shield (was 10, EC2 moved to Tier 1)
		{
			ServiceName: "shield",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateProtection":   createAction("CreateProtection", "protection", "ProtectionId", []stub.Field{reqStr("Name"), reqStr("ResourceArn")}, []stub.Field{optStr("Name")}),
				"DescribeProtection": describeAction("DescribeProtection", "protection", "ProtectionId"),
				"ListProtections":    listAction("ListProtections", "protection"),
				"DeleteProtection":   deleteAction("DeleteProtection", "protection", "ProtectionId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"protection": rt("Protection", "ProtectionId", "arn:aws:shield::{account}:protection/{id}", []stub.Field{optStr("Name"), optStr("ResourceArn")}),
			},
		},
		// 11. WAF Regional
		{
			ServiceName: "waf-regional",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateWebACL": createAction("CreateWebACL", "webacl", "WebACLId", []stub.Field{reqStr("Name")}, []stub.Field{optStr("Name")}),
				"GetWebACL":    describeAction("GetWebACL", "webacl", "WebACLId"),
				"ListWebACLs":  listAction("ListWebACLs", "webacl"),
				"DeleteWebACL": deleteAction("DeleteWebACL", "webacl", "WebACLId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"webacl": rt("WebACL", "WebACLId", "arn:aws:waf-regional:{region}:{account}:webacl/{id}", []stub.Field{optStr("Name")}),
			},
		},
	}
}

// ─── JSON protocol services ─────────────────────────────────────────────────

func jsonServices() []*stub.ServiceModel {
	return []*stub.ServiceModel{
		// 12. ACM
		{
			ServiceName:  "acm",
			Protocol:     "json",
			TargetPrefix: "CertificateManager",
			Actions: map[string]stub.Action{
				"RequestCertificate":  createAction("RequestCertificate", "certificate", "CertificateArn", []stub.Field{reqStr("DomainName")}, []stub.Field{optStr("DomainName")}),
				"DescribeCertificate": describeAction("DescribeCertificate", "certificate", "CertificateArn"),
				"ListCertificates":    listAction("ListCertificates", "certificate"),
				"DeleteCertificate":   deleteAction("DeleteCertificate", "certificate", "CertificateArn"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"certificate": rt("Certificate", "CertificateArn", "arn:aws:acm:{region}:{account}:certificate/{id}", []stub.Field{optStr("DomainName")}),
			},
		},
		// 13. ACM PCA
		{
			ServiceName:  "acm-pca",
			Protocol:     "json",
			TargetPrefix: "ACMPrivateCA",
			Actions: map[string]stub.Action{
				"CreateCertificateAuthority":    createAction("CreateCertificateAuthority", "ca", "CertificateAuthorityArn", []stub.Field{reqStr("CertificateAuthorityType")}, []stub.Field{optStr("CertificateAuthorityType")}),
				"DescribeCertificateAuthority":  describeAction("DescribeCertificateAuthority", "ca", "CertificateAuthorityArn"),
				"ListCertificateAuthorities":    listAction("ListCertificateAuthorities", "ca"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"ca": rt("CertificateAuthority", "CertificateAuthorityArn", "arn:aws:acm-pca:{region}:{account}:certificate-authority/{id}", []stub.Field{optStr("CertificateAuthorityType")}),
			},
		},
		// 14. AppConfig
		{
			ServiceName:  "appconfig",
			Protocol:     "json",
			TargetPrefix: "AppConfig",
			Actions: map[string]stub.Action{
				"CreateApplication": createAction("CreateApplication", "application", "Id", []stub.Field{reqStr("Name")}, []stub.Field{optStr("Name")}),
				"GetApplication":    describeAction("GetApplication", "application", "Id"),
				"ListApplications":  listAction("ListApplications", "application"),
				"DeleteApplication": deleteAction("DeleteApplication", "application", "Id"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"application": rt("Application", "Id", "arn:aws:appconfig:{region}:{account}:application/{id}", []stub.Field{optStr("Name")}),
			},
		},
		// 15. Application Auto Scaling
		{
			ServiceName:  "application-autoscaling",
			Protocol:     "json",
			TargetPrefix: "AnyScaleFrontendService",
			Actions: map[string]stub.Action{
				"RegisterScalableTarget":   createAction("RegisterScalableTarget", "target", "ResourceId", []stub.Field{reqStr("ServiceNamespace"), reqStr("ResourceId"), reqStr("ScalableDimension")}, []stub.Field{optStr("ResourceId")}),
				"DescribeScalableTargets":  listAction("DescribeScalableTargets", "target"),
				"DeregisterScalableTarget": deleteAction("DeregisterScalableTarget", "target", "ResourceId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"target": rt("ScalableTarget", "ResourceId", "arn:aws:application-autoscaling:{region}:{account}:scalable-target/{id}", []stub.Field{optStr("ServiceNamespace"), optStr("ScalableDimension")}),
			},
		},
		// 17. Athena
		{
			ServiceName:  "athena",
			Protocol:     "json",
			TargetPrefix: "AmazonAthena",
			Actions: map[string]stub.Action{
				"StartQueryExecution": createAction("StartQueryExecution", "query", "QueryExecutionId", []stub.Field{reqStr("QueryString")}, []stub.Field{optStr("QueryString")}),
				"GetQueryExecution":   describeAction("GetQueryExecution", "query", "QueryExecutionId"),
				"ListQueryExecutions": listAction("ListQueryExecutions", "query"),
				"StopQueryExecution":  deleteAction("StopQueryExecution", "query", "QueryExecutionId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"query": rt("QueryExecution", "QueryExecutionId", "arn:aws:athena:{region}:{account}:query/{id}", []stub.Field{optStr("QueryString")}),
			},
		},
		// 18. Backup
		{
			ServiceName:  "backup",
			Protocol:     "json",
			TargetPrefix: "CryoControllerUserManager",
			Actions: map[string]stub.Action{
				"CreateBackupPlan":   createAction("CreateBackupPlan", "plan", "BackupPlanId", []stub.Field{reqStr("BackupPlanName")}, []stub.Field{optStr("BackupPlanName")}),
				"DescribeBackupJob":  describeAction("DescribeBackupJob", "plan", "BackupPlanId"),
				"ListBackupPlans":    listAction("ListBackupPlans", "plan"),
				"DeleteBackupPlan":   deleteAction("DeleteBackupPlan", "plan", "BackupPlanId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"plan": rt("BackupPlan", "BackupPlanId", "arn:aws:backup:{region}:{account}:backup-plan:{id}", []stub.Field{optStr("BackupPlanName")}),
			},
		},
		// 21. CodeBuild
		{
			ServiceName:  "codebuild",
			Protocol:     "json",
			TargetPrefix: "CodeBuild_20161006",
			Actions: map[string]stub.Action{
				"CreateProject":    createAction("CreateProject", "project", "ProjectName", []stub.Field{reqStr("Name"), reqStr("Source")}, []stub.Field{optStr("Name")}),
				"BatchGetProjects": describeAction("BatchGetProjects", "project", "ProjectName"),
				"ListProjects":     listAction("ListProjects", "project"),
				"DeleteProject":    deleteAction("DeleteProject", "project", "ProjectName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"project": rt("Project", "ProjectName", "arn:aws:codebuild:{region}:{account}:project/{id}", []stub.Field{optStr("Name"), optStr("Source")}),
			},
		},
		// 22. CodeCommit
		{
			ServiceName:  "codecommit",
			Protocol:     "json",
			TargetPrefix: "CodeCommit_20150413",
			Actions: map[string]stub.Action{
				"CreateRepository": createAction("CreateRepository", "repository", "RepositoryId", []stub.Field{reqStr("RepositoryName")}, []stub.Field{optStr("RepositoryName")}),
				"GetRepository":    describeAction("GetRepository", "repository", "RepositoryId"),
				"ListRepositories": listAction("ListRepositories", "repository"),
				"DeleteRepository": deleteAction("DeleteRepository", "repository", "RepositoryId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"repository": rt("Repository", "RepositoryId", "arn:aws:codecommit:{region}:{account}:{id}", []stub.Field{optStr("RepositoryName")}),
			},
		},
		// 23. CodeDeploy
		{
			ServiceName:  "codedeploy",
			Protocol:     "json",
			TargetPrefix: "CodeDeploy_20141006",
			Actions: map[string]stub.Action{
				"CreateApplication": createAction("CreateApplication", "application", "ApplicationId", []stub.Field{reqStr("ApplicationName")}, []stub.Field{optStr("ApplicationName")}),
				"GetApplication":    describeAction("GetApplication", "application", "ApplicationId"),
				"ListApplications":  listAction("ListApplications", "application"),
				"DeleteApplication": deleteAction("DeleteApplication", "application", "ApplicationId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"application": rt("Application", "ApplicationId", "arn:aws:codedeploy:{region}:{account}:application:{id}", []stub.Field{optStr("ApplicationName")}),
			},
		},
		// 24. CodePipeline
		{
			ServiceName:  "codepipeline",
			Protocol:     "json",
			TargetPrefix: "CodePipeline_20150709",
			Actions: map[string]stub.Action{
				"CreatePipeline": createAction("CreatePipeline", "pipeline", "PipelineName", []stub.Field{reqStr("PipelineName")}, []stub.Field{optStr("PipelineName")}),
				"GetPipeline":    describeAction("GetPipeline", "pipeline", "PipelineName"),
				"ListPipelines":  listAction("ListPipelines", "pipeline"),
				"DeletePipeline": deleteAction("DeletePipeline", "pipeline", "PipelineName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"pipeline": rt("Pipeline", "PipelineName", "arn:aws:codepipeline:{region}:{account}:{id}", []stub.Field{optStr("PipelineName")}),
			},
		},
		// 26. CodeConnections
		{
			ServiceName:  "codeconnections",
			Protocol:     "json",
			TargetPrefix: "CodeConnections_20231201",
			Actions: map[string]stub.Action{
				"CreateConnection": createAction("CreateConnection", "connection", "ConnectionArn", []stub.Field{reqStr("ConnectionName")}, []stub.Field{optStr("ConnectionName")}),
				"GetConnection":    describeAction("GetConnection", "connection", "ConnectionArn"),
				"ListConnections":  listAction("ListConnections", "connection"),
				"DeleteConnection": deleteAction("DeleteConnection", "connection", "ConnectionArn"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"connection": rt("Connection", "ConnectionArn", "arn:aws:codeconnections:{region}:{account}:connection/{id}", []stub.Field{optStr("ConnectionName")}),
			},
		},
		// 27. Config
		{
			ServiceName:  "config",
			Protocol:     "json",
			TargetPrefix: "StarlingDoveService",
			Actions: map[string]stub.Action{
				"PutConfigRule":       createAction("PutConfigRule", "rule", "ConfigRuleName", []stub.Field{reqStr("ConfigRuleName")}, []stub.Field{optStr("ConfigRuleName")}),
				"DescribeConfigRules": listAction("DescribeConfigRules", "rule"),
				"DeleteConfigRule":    deleteAction("DeleteConfigRule", "rule", "ConfigRuleName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"rule": rt("ConfigRule", "ConfigRuleName", "arn:aws:config:{region}:{account}:config-rule/{id}", []stub.Field{optStr("ConfigRuleName")}),
			},
		},
		// 28. Cost Explorer
		{
			ServiceName:  "ce",
			Protocol:     "json",
			TargetPrefix: "AWSInsightsIndexService",
			Actions: map[string]stub.Action{
				"GetCostAndUsage":        otherAction("GetCostAndUsage", "cost"),
				"GetCostForecast":        otherAction("GetCostForecast", "cost"),
				"ListCostAllocationTags": listAction("ListCostAllocationTags", "cost"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cost": rt("CostReport", "ReportId", "arn:aws:ce:{region}:{account}:report/{id}", nil),
			},
		},
		// 29. DMS
		{
			ServiceName:  "dms",
			Protocol:     "json",
			TargetPrefix: "AmazonDMSv20160101",
			Actions: map[string]stub.Action{
				"CreateReplicationInstance":    createAction("CreateReplicationInstance", "instance", "ReplicationInstanceIdentifier", []stub.Field{reqStr("ReplicationInstanceIdentifier"), reqStr("ReplicationInstanceClass")}, []stub.Field{optStr("ReplicationInstanceIdentifier")}),
				"DescribeReplicationInstances": listAction("DescribeReplicationInstances", "instance"),
				"DeleteReplicationInstance":    deleteAction("DeleteReplicationInstance", "instance", "ReplicationInstanceIdentifier"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"instance": rt("ReplicationInstance", "ReplicationInstanceIdentifier", "arn:aws:dms:{region}:{account}:rep:{id}", []stub.Field{optStr("ReplicationInstanceIdentifier"), optStr("ReplicationInstanceClass")}),
			},
		},
		// 30. DocumentDB (query protocol like RDS)
		{
			ServiceName: "docdb",
			Protocol:    "query",
			Actions: map[string]stub.Action{
				"CreateDBCluster":    createAction("CreateDBCluster", "cluster", "DBClusterIdentifier", []stub.Field{reqStr("DBClusterIdentifier"), reqStr("Engine")}, []stub.Field{optStr("DBClusterIdentifier")}),
				"DescribeDBClusters": listAction("DescribeDBClusters", "cluster"),
				"DeleteDBCluster":    deleteAction("DeleteDBCluster", "cluster", "DBClusterIdentifier"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cluster": rt("DBCluster", "DBClusterIdentifier", "arn:aws:rds:{region}:{account}:cluster:{id}", []stub.Field{optStr("DBClusterIdentifier"), optStr("Engine")}),
			},
		},
		// 33. Glue
		{
			ServiceName:  "glue",
			Protocol:     "json",
			TargetPrefix: "AWSGlue",
			Actions: map[string]stub.Action{
				"CreateDatabase": createAction("CreateDatabase", "database", "DatabaseName", []stub.Field{reqStr("DatabaseName")}, []stub.Field{optStr("DatabaseName")}),
				"GetDatabase":    describeAction("GetDatabase", "database", "DatabaseName"),
				"GetDatabases":   listAction("GetDatabases", "database"),
				"DeleteDatabase": deleteAction("DeleteDatabase", "database", "DatabaseName"),
				"CreateTable":    createAction("CreateTable", "table", "TableName", []stub.Field{reqStr("DatabaseName"), reqStr("TableName")}, []stub.Field{optStr("TableName")}),
				"GetTable":       describeAction("GetTable", "table", "TableName"),
				"GetTables":      listAction("GetTables", "table"),
				"DeleteTable":    deleteAction("DeleteTable", "table", "TableName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"database": rt("Database", "DatabaseName", "arn:aws:glue:{region}:{account}:database/{id}", []stub.Field{optStr("DatabaseName")}),
				"table":    rt("Table", "TableName", "arn:aws:glue:{region}:{account}:table/{id}", []stub.Field{optStr("DatabaseName"), optStr("TableName")}),
			},
		},
		// 34. Identity Store
		{
			ServiceName:  "identitystore",
			Protocol:     "json",
			TargetPrefix: "AWSIdentityStore",
			Actions: map[string]stub.Action{
				"CreateUser":  createAction("CreateUser", "user", "UserId", []stub.Field{reqStr("IdentityStoreId"), reqStr("UserName")}, []stub.Field{optStr("UserName")}),
				"DescribeUser": describeAction("DescribeUser", "user", "UserId"),
				"ListUsers":    listAction("ListUsers", "user"),
				"DeleteUser":   deleteAction("DeleteUser", "user", "UserId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"user": rt("User", "UserId", "arn:aws:identitystore:{region}:{account}:user/{id}", []stub.Field{optStr("UserName"), optStr("IdentityStoreId")}),
			},
		},
		// 38. Lake Formation
		{
			ServiceName:  "lakeformation",
			Protocol:     "json",
			TargetPrefix: "AWSLakeFormation",
			Actions: map[string]stub.Action{
				"RegisterResource":   createAction("RegisterResource", "resource", "ResourceArn", []stub.Field{reqStr("ResourceArn")}, []stub.Field{optStr("ResourceArn")}),
				"DescribeResource":   describeAction("DescribeResource", "resource", "ResourceArn"),
				"ListResources":      listAction("ListResources", "resource"),
				"DeregisterResource": deleteAction("DeregisterResource", "resource", "ResourceArn"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"resource": rt("Resource", "ResourceArn", "arn:aws:lakeformation:{region}:{account}:resource/{id}", []stub.Field{optStr("ResourceArn")}),
			},
		},
		// 42. MemoryDB
		{
			ServiceName:  "memorydb",
			Protocol:     "json",
			TargetPrefix: "AmazonMemoryDB",
			Actions: map[string]stub.Action{
				"CreateCluster":    createAction("CreateCluster", "cluster", "ClusterName", []stub.Field{reqStr("ClusterName"), reqStr("NodeType")}, []stub.Field{optStr("ClusterName")}),
				"DescribeClusters": listAction("DescribeClusters", "cluster"),
				"DeleteCluster":    deleteAction("DeleteCluster", "cluster", "ClusterName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"cluster": rt("Cluster", "ClusterName", "arn:aws:memorydb:{region}:{account}:cluster/{id}", []stub.Field{optStr("ClusterName"), optStr("NodeType")}),
			},
		},
		// 45. Organizations
		{
			ServiceName:  "organizations",
			Protocol:     "json",
			TargetPrefix: "AWSOrganizationsV20161128",
			Actions: map[string]stub.Action{
				"CreateOrganization":   createAction("CreateOrganization", "org", "OrganizationId", nil, nil),
				"DescribeOrganization": describeAction("DescribeOrganization", "org", "OrganizationId"),
				"ListAccounts":         listAction("ListAccounts", "account"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"org":     rt("Organization", "OrganizationId", "arn:aws:organizations::{account}:organization/{id}", nil),
				"account": rt("Account", "AccountId", "arn:aws:organizations::{account}:account/{id}", []stub.Field{optStr("AccountName"), optStr("Email")}),
			},
		},
		// 47. RAM
		{
			ServiceName:  "ram",
			Protocol:     "json",
			TargetPrefix: "AWSRAMShareService",
			Actions: map[string]stub.Action{
				"CreateResourceShare": createAction("CreateResourceShare", "share", "ResourceShareArn", []stub.Field{reqStr("Name")}, []stub.Field{optStr("Name")}),
				"GetResourceShares":   listAction("GetResourceShares", "share"),
				"DeleteResourceShare": deleteAction("DeleteResourceShare", "share", "ResourceShareArn"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"share": rt("ResourceShare", "ResourceShareArn", "arn:aws:ram:{region}:{account}:resource-share/{id}", []stub.Field{optStr("Name")}),
			},
		},
		// 49. Resource Groups Tagging API
		{
			ServiceName:  "tagging",
			Protocol:     "json",
			TargetPrefix: "ResourceGroupsTaggingAPI_20170126",
			Actions: map[string]stub.Action{
				"TagResources":   otherAction("TagResources", "taggedresource"),
				"UntagResources": otherAction("UntagResources", "taggedresource"),
				"GetResources":   listAction("GetResources", "taggedresource"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"taggedresource": rt("TaggedResource", "ResourceARN", "arn:aws:tagging:{region}:{account}:resource/{id}", nil),
			},
		},
		// 50. SageMaker
		{
			ServiceName:  "sagemaker",
			Protocol:     "json",
			TargetPrefix: "SageMaker",
			Actions: map[string]stub.Action{
				"CreateNotebookInstance":   createAction("CreateNotebookInstance", "notebook", "NotebookInstanceName", []stub.Field{reqStr("NotebookInstanceName"), reqStr("InstanceType"), reqStr("RoleArn")}, []stub.Field{optStr("NotebookInstanceName")}),
				"DescribeNotebookInstance": describeAction("DescribeNotebookInstance", "notebook", "NotebookInstanceName"),
				"ListNotebookInstances":    listAction("ListNotebookInstances", "notebook"),
				"DeleteNotebookInstance":   deleteAction("DeleteNotebookInstance", "notebook", "NotebookInstanceName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"notebook": rt("NotebookInstance", "NotebookInstanceName", "arn:aws:sagemaker:{region}:{account}:notebook-instance/{id}", []stub.Field{optStr("NotebookInstanceName"), optStr("InstanceType"), optStr("RoleArn")}),
			},
		},
		// 52. Service Discovery
		{
			ServiceName:  "servicediscovery",
			Protocol:     "json",
			TargetPrefix: "Route53AutoNaming_v20170314",
			Actions: map[string]stub.Action{
				"CreateService": createAction("CreateService", "service", "ServiceId", []stub.Field{reqStr("Name")}, []stub.Field{optStr("Name")}),
				"GetService":    describeAction("GetService", "service", "ServiceId"),
				"ListServices":  listAction("ListServices", "service"),
				"DeleteService": deleteAction("DeleteService", "service", "ServiceId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"service": rt("Service", "ServiceId", "arn:aws:servicediscovery:{region}:{account}:service/{id}", []stub.Field{optStr("Name")}),
			},
		},
		// 53. SWF
		{
			ServiceName:  "swf",
			Protocol:     "json",
			TargetPrefix: "SimpleWorkflowService",
			Actions: map[string]stub.Action{
				"RegisterDomain":  createAction("RegisterDomain", "domain", "DomainName", []stub.Field{reqStr("Name"), reqStr("WorkflowExecutionRetentionPeriodInDays")}, []stub.Field{optStr("Name")}),
				"ListDomains":     listAction("ListDomains", "domain"),
				"DescribeDomain":  describeAction("DescribeDomain", "domain", "DomainName"),
				"DeprecateDomain": deleteAction("DeprecateDomain", "domain", "DomainName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"domain": rt("Domain", "DomainName", "arn:aws:swf:{region}:{account}:domain/{id}", []stub.Field{optStr("Name")}),
			},
		},
		// 54. SSO Admin
		{
			ServiceName:  "sso-admin",
			Protocol:     "json",
			TargetPrefix: "SWBExternalService",
			Actions: map[string]stub.Action{
				"CreatePermissionSet":   createAction("CreatePermissionSet", "permset", "PermissionSetArn", []stub.Field{reqStr("Name"), reqStr("InstanceArn")}, []stub.Field{optStr("Name")}),
				"DescribePermissionSet": describeAction("DescribePermissionSet", "permset", "PermissionSetArn"),
				"ListPermissionSets":    listAction("ListPermissionSets", "permset"),
				"DeletePermissionSet":   deleteAction("DeletePermissionSet", "permset", "PermissionSetArn"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"permset": rt("PermissionSet", "PermissionSetArn", "arn:aws:sso::{account}:permissionSet/{id}", []stub.Field{optStr("Name"), optStr("InstanceArn")}),
			},
		},
		// 55. Support
		{
			ServiceName:  "support",
			Protocol:     "json",
			TargetPrefix: "AWSSupport_20130415",
			Actions: map[string]stub.Action{
				"CreateCase":                     createAction("CreateCase", "case", "CaseId", []stub.Field{reqStr("Subject"), reqStr("CommunicationBody")}, []stub.Field{optStr("Subject")}),
				"DescribeCases":                  listAction("DescribeCases", "case"),
				"DescribeTrustedAdvisorChecks":   otherAction("DescribeTrustedAdvisorChecks", "case"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"case": rt("Case", "CaseId", "arn:aws:support:{region}:{account}:case/{id}", []stub.Field{optStr("Subject"), optStr("CommunicationBody")}),
			},
		},
		// 56. Textract
		{
			ServiceName:  "textract",
			Protocol:     "json",
			TargetPrefix: "Textract",
			Actions: map[string]stub.Action{
				"DetectDocumentText":          otherAction("DetectDocumentText", "document"),
				"AnalyzeDocument":             otherAction("AnalyzeDocument", "document"),
				"StartDocumentTextDetection":  createAction("StartDocumentTextDetection", "job", "JobId", []stub.Field{reqStr("DocumentLocation")}, []stub.Field{optStr("DocumentLocation")}),
				"GetDocumentTextDetection":    describeAction("GetDocumentTextDetection", "job", "JobId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"document": rt("Document", "DocumentId", "arn:aws:textract:{region}:{account}:document/{id}", nil),
				"job":      rt("Job", "JobId", "arn:aws:textract:{region}:{account}:job/{id}", []stub.Field{optStr("DocumentLocation")}),
			},
		},
		// 57. Timestream
		{
			ServiceName:  "timestream-write",
			Protocol:     "json",
			TargetPrefix: "Timestream_20181101",
			Actions: map[string]stub.Action{
				"CreateDatabase":   createAction("CreateDatabase", "database", "DatabaseName", []stub.Field{reqStr("DatabaseName")}, []stub.Field{optStr("DatabaseName")}),
				"DescribeDatabase": describeAction("DescribeDatabase", "database", "DatabaseName"),
				"ListDatabases":    listAction("ListDatabases", "database"),
				"DeleteDatabase":   deleteAction("DeleteDatabase", "database", "DatabaseName"),
				"CreateTable":      createAction("CreateTable", "table", "TableName", []stub.Field{reqStr("DatabaseName"), reqStr("TableName")}, []stub.Field{optStr("TableName")}),
				"DescribeTable":    describeAction("DescribeTable", "table", "TableName"),
				"ListTables":       listAction("ListTables", "table"),
				"DeleteTable":      deleteAction("DeleteTable", "table", "TableName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"database": rt("Database", "DatabaseName", "arn:aws:timestream:{region}:{account}:database/{id}", []stub.Field{optStr("DatabaseName")}),
				"table":    rt("Table", "TableName", "arn:aws:timestream:{region}:{account}:database/db/table/{id}", []stub.Field{optStr("DatabaseName"), optStr("TableName")}),
			},
		},
		// 58. Transcribe
		{
			ServiceName:  "transcribe",
			Protocol:     "json",
			TargetPrefix: "Transcribe",
			Actions: map[string]stub.Action{
				"StartTranscriptionJob":  createAction("StartTranscriptionJob", "job", "TranscriptionJobName", []stub.Field{reqStr("TranscriptionJobName"), reqStr("MediaFileUri")}, []stub.Field{optStr("TranscriptionJobName")}),
				"GetTranscriptionJob":    describeAction("GetTranscriptionJob", "job", "TranscriptionJobName"),
				"ListTranscriptionJobs":  listAction("ListTranscriptionJobs", "job"),
				"DeleteTranscriptionJob": deleteAction("DeleteTranscriptionJob", "job", "TranscriptionJobName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"job": rt("TranscriptionJob", "TranscriptionJobName", "arn:aws:transcribe:{region}:{account}:transcription-job/{id}", []stub.Field{optStr("TranscriptionJobName"), optStr("MediaFileUri")}),
			},
		},
		// 59. Transfer
		{
			ServiceName:  "transfer",
			Protocol:     "json",
			TargetPrefix: "TransferService",
			Actions: map[string]stub.Action{
				"CreateServer":   createAction("CreateServer", "server", "ServerId", nil, nil),
				"DescribeServer": describeAction("DescribeServer", "server", "ServerId"),
				"ListServers":    listAction("ListServers", "server"),
				"DeleteServer":   deleteAction("DeleteServer", "server", "ServerId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"server": rt("Server", "ServerId", "arn:aws:transfer:{region}:{account}:server/{id}", nil),
			},
		},
		// 60. Verified Permissions
		{
			ServiceName:  "verifiedpermissions",
			Protocol:     "json",
			TargetPrefix: "VerifiedPermissions",
			Actions: map[string]stub.Action{
				"CreatePolicyStore": createAction("CreatePolicyStore", "policystore", "PolicyStoreId", nil, nil),
				"GetPolicyStore":    describeAction("GetPolicyStore", "policystore", "PolicyStoreId"),
				"ListPolicyStores":  listAction("ListPolicyStores", "policystore"),
				"DeletePolicyStore": deleteAction("DeletePolicyStore", "policystore", "PolicyStoreId"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"policystore": rt("PolicyStore", "PolicyStoreId", "arn:aws:verifiedpermissions:{region}:{account}:policy-store/{id}", nil),
			},
		},
		// 63. Cloud Control
		{
			ServiceName:  "cloudcontrol",
			Protocol:     "json",
			TargetPrefix: "CloudApiService",
			Actions: map[string]stub.Action{
				"CreateResource": createAction("CreateResource", "resource", "Identifier", []stub.Field{reqStr("TypeName"), reqStr("DesiredState")}, []stub.Field{optStr("TypeName")}),
				"GetResource":    describeAction("GetResource", "resource", "Identifier"),
				"ListResources":  listAction("ListResources", "resource"),
				"DeleteResource": deleteAction("DeleteResource", "resource", "Identifier"),
				"UpdateResource": updateAction("UpdateResource", "resource", "Identifier", []stub.Field{optStr("PatchDocument")}),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"resource": rt("Resource", "Identifier", "arn:aws:cloudcontrol:{region}:{account}:resource/{id}", []stub.Field{optStr("TypeName"), optStr("DesiredState")}),
			},
		},
		// 65. CloudTrail
		{
			ServiceName:  "cloudtrail",
			Protocol:     "json",
			TargetPrefix: "CloudTrail_20131101",
			Actions: map[string]stub.Action{
				"CreateTrail":   createAction("CreateTrail", "trail", "TrailName", []stub.Field{reqStr("Name"), reqStr("S3BucketName")}, []stub.Field{optStr("Name")}),
				"GetTrail":      describeAction("GetTrail", "trail", "TrailName"),
				"DescribeTrails": listAction("DescribeTrails", "trail"),
				"DeleteTrail":   deleteAction("DeleteTrail", "trail", "TrailName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"trail": rt("Trail", "TrailName", "arn:aws:cloudtrail:{region}:{account}:trail/{id}", []stub.Field{optStr("Name"), optStr("S3BucketName")}),
			},
		},
		// 70. Kinesis Analytics (Apache Flink)
		{
			ServiceName:  "kinesisanalytics",
			Protocol:     "json",
			TargetPrefix: "KinesisAnalytics_20180523",
			Actions: map[string]stub.Action{
				"CreateApplication":   createAction("CreateApplication", "application", "ApplicationName", []stub.Field{reqStr("ApplicationName"), reqStr("RuntimeEnvironment")}, []stub.Field{optStr("ApplicationName")}),
				"DescribeApplication": describeAction("DescribeApplication", "application", "ApplicationName"),
				"ListApplications":    listAction("ListApplications", "application"),
				"DeleteApplication":   deleteAction("DeleteApplication", "application", "ApplicationName"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"application": rt("Application", "ApplicationName", "arn:aws:kinesisanalytics:{region}:{account}:application/{id}", []stub.Field{optStr("ApplicationName"), optStr("RuntimeEnvironment")}),
			},
		},
		// 72. X-Ray
		{
			ServiceName:  "xray",
			Protocol:     "json",
			TargetPrefix: "AWSXRay",
			Actions: map[string]stub.Action{
				"PutTraceSegments":   otherAction("PutTraceSegments", "trace"),
				"GetTraceSummaries":  listAction("GetTraceSummaries", "trace"),
				"BatchGetTraces":     otherAction("BatchGetTraces", "trace"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"trace": rt("TraceSummary", "TraceId", "arn:aws:xray:{region}:{account}:trace/{id}", nil),
			},
		},
	}
}

// ─── REST-JSON services ─────────────────────────────────────────────────────

func restJSONServices() []*stub.ServiceModel {
	rj := func(serviceName, resName, idField string) *stub.ServiceModel {
		resKey := "resource"
		return &stub.ServiceModel{
			ServiceName: serviceName,
			Protocol:    "rest-json",
			Actions: map[string]stub.Action{
				"Create" + resName: createAction("Create"+resName, resKey, idField, []stub.Field{optStr("Name")}, []stub.Field{optStr("Name")}),
				"List" + resName + "s": listAction("List"+resName+"s", resKey),
				"Get" + resName:    describeAction("Get"+resName, resKey, idField),
				"Delete" + resName: deleteAction("Delete"+resName, resKey, idField),
			},
			ResourceTypes: map[string]stub.ResourceType{
				resKey: rt(resName, idField, "arn:aws:"+serviceName+":{region}:{account}:"+strings.ToLower(resName)+"/{id}", []stub.Field{optStr("Name")}),
			},
		}
	}

	return []*stub.ServiceModel{
		// 16. AppSync
		rj("appsync", "GraphqlApi", "ApiId"),
		// 19. Batch
		rj("batch", "JobQueue", "JobQueueName"),
		// 20. Bedrock
		rj("bedrock", "ModelCustomizationJob", "JobArn"),
		// 25. CodeArtifact
		rj("codeartifact", "Repository", "RepositoryName"),
		// 31. EKS
		rj("eks", "Cluster", "ClusterName"),
		// 32. FIS
		rj("fis", "ExperimentTemplate", "Id"),
		// 35. IoT
		rj("iot", "Thing", "ThingName"),
		// 36. IoT Data
		rj("iot-data", "ThingShadow", "ThingName"),
		// 37. IoT Wireless
		rj("iot-wireless", "WirelessDevice", "Id"),
		// 39. Managed Blockchain
		rj("managedblockchain", "Network", "NetworkId"),
		// 40. MSK (Kafka)
		rj("kafka", "Cluster", "ClusterArn"),
		// 41. MWAA (Airflow)
		rj("airflow", "Environment", "Name"),
		// 43. MQ
		rj("mq", "Broker", "BrokerId"),
		// 44. OpenSearch
		rj("opensearch", "Domain", "DomainName"),
		// 46. Pinpoint
		rj("pinpoint", "App", "ApplicationId"),
		// 48. Resource Groups
		rj("resource-groups", "Group", "GroupName"),
		// 51. Serverless App Repo
		rj("serverlessrepo", "Application", "ApplicationId"),
		// 61. Amplify
		rj("amplify", "App", "AppId"),
		// 62. Account Management
		rj("account", "ContactInformation", "AccountId"),
		// 66. Glacier
		rj("glacier", "Vault", "VaultName"),
		// 67. MediaConvert
		rj("mediaconvert", "Job", "Id"),
		// 68. EventBridge Pipes
		rj("pipes", "Pipe", "Name"),
		// 69. EventBridge Scheduler
		rj("scheduler", "Schedule", "Name"),
		// 71. S3 Tables
		rj("s3tables", "TableBucket", "TableBucketARN"),
		// 73. WAFv2
		rj("wafv2", "WebACL", "WebACLId"),
		// 74. Route 53 Resolver
		rj("route53resolver", "ResolverEndpoint", "Id"),
		// 75. App Runner
		rj("apprunner", "Service", "ServiceId"),
		// 76. App Mesh
		rj("appmesh", "Mesh", "MeshName"),
		// 77. CloudMap (alias for servicediscovery REST)
		rj("cloud9", "Environment", "EnvironmentId"),
		// 78. CodeStar Connections
		rj("codestar-connections", "Connection", "ConnectionArn"),
		// 79. DataSync
		rj("datasync", "Task", "TaskArn"),
		// 80. Device Farm
		rj("devicefarm", "Project", "Arn"),
		// 81. EventBridge (REST-JSON endpoint)
		rj("events", "EventBus", "EventBusName"),
		// 82. FinSpace
		rj("finspace", "Environment", "EnvironmentId"),
		// 83. Forecast
		rj("forecast", "DatasetGroup", "DatasetGroupArn"),
		// 84. GroundStation
		rj("groundstation", "Config", "ConfigArn"),
		// 85. HealthLake
		rj("healthlake", "FHIRDatastore", "DatastoreId"),
		// 86. Inspector2
		rj("inspector2", "Filter", "FilterArn"),
		// 87. Keyspaces (Managed Cassandra)
		rj("cassandra", "Keyspace", "keyspaceName"),
		// 88. Location Service
		rj("location", "Map", "MapName"),
		// 89. Lookout for Metrics
		rj("lookoutmetrics", "AnomalyDetector", "AnomalyDetectorArn"),
		// 90. Macie
		rj("macie2", "FindingsFilter", "Id"),
		// 91. Migration Hub
		rj("mgh", "ProgressUpdateStream", "ProgressUpdateStreamName"),
		// 92. Nimble Studio
		rj("nimble", "Studio", "StudioId"),
		// 93. Outposts
		rj("outposts", "Outpost", "OutpostId"),
		// 94. Panorama
		rj("panorama", "Device", "DeviceId"),
		// 95. Private 5G
		rj("private-networks", "Network", "NetworkArn"),
		// 96. Proton
		rj("proton", "EnvironmentTemplate", "TemplateName"),
		// 97. Rekognition
		rj("rekognition", "Collection", "CollectionId"),
		// 98. Robomaker
		rj("robomaker", "SimulationApplication", "Arn"),
		// 99. Security Hub
		rj("securityhub", "Hub", "HubArn"),
	}
}

// ─── REST-XML services ──────────────────────────────────────────────────────

func restXMLServices() []*stub.ServiceModel {
	return []*stub.ServiceModel{
		// 64. CloudFront
		{
			ServiceName: "cloudfront",
			Protocol:    "rest-xml",
			Actions: map[string]stub.Action{
				"CreateDistribution":  createAction("CreateDistribution", "distribution", "Id", []stub.Field{optStr("CallerReference")}, []stub.Field{optStr("CallerReference")}),
				"GetDistribution":     describeAction("GetDistribution", "distribution", "Id"),
				"ListDistributions":   listAction("ListDistributions", "distribution"),
				"DeleteDistribution":  deleteAction("DeleteDistribution", "distribution", "Id"),
			},
			ResourceTypes: map[string]stub.ResourceType{
				"distribution": rt("Distribution", "Id", "arn:aws:cloudfront::{account}:distribution/{id}", []stub.Field{optStr("CallerReference"), optStr("DomainName")}),
			},
		},
	}
}
