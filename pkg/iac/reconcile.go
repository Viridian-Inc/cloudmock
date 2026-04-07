package iac

import (
	"log/slog"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Reconciler compares a new IaC scan result against what's currently provisioned
// in CloudMock services and adds/removes resources to match the IaC source of truth.
type Reconciler struct {
	logger   *slog.Logger
	registry serviceRegistry
}

// serviceRegistry is the subset of routing.Registry needed by the reconciler.
type serviceRegistry interface {
	Lookup(name string) (service.Service, error)
}

// NewReconciler creates a reconciler that syncs IaC state into CloudMock services.
func NewReconciler(registry serviceRegistry, logger *slog.Logger) *Reconciler {
	return &Reconciler{registry: registry, logger: logger}
}

// Reconcile provisions new resources and removes stale ones based on the IaC scan.
func (r *Reconciler) Reconcile(result *IaCImportResult) {
	r.reconcileDynamo(result.Tables)
	r.reconcileLambda(result.Lambdas)
	r.reconcileSQS(result.SQSQueues)
	r.reconcileSNS(result.SNSTopics)
	r.reconcileS3(result.S3Buckets)
	r.reconcileCognito(result.CognitoPools)
	r.reconcileAPIGateway(result.APIGateways)
}

func (r *Reconciler) reconcileDynamo(desired []DynamoTableDef) {
	svc, err := r.registry.Lookup("dynamodb")
	if err != nil {
		return
	}
	lister, ok := svc.(interface{ GetTableNames() []string })
	if !ok {
		return
	}
	deleter, ok := svc.(interface {
		DeleteTable(name string) (interface{}, interface{})
	})
	if !ok {
		return
	}

	desiredSet := make(map[string]bool, len(desired))
	for _, t := range desired {
		desiredSet[t.Name] = true
	}

	// Remove tables not in IaC
	for _, name := range lister.GetTableNames() {
		if !desiredSet[name] {
			deleter.DeleteTable(name)
			r.logger.Info("reconcile: removed stale DynamoDB table", "table", name)
		}
	}

	// Provision new tables
	ProvisionDynamoTables(desired, svc, r.logger)
}

func (r *Reconciler) reconcileLambda(desired []LambdaDef) {
	svc, err := r.registry.Lookup("lambda")
	if err != nil {
		return
	}
	lister, ok := svc.(interface{ GetFunctionNames() []string })
	if !ok {
		return
	}
	deleter, ok := svc.(interface{ Delete(name string) bool })
	if !ok {
		return
	}

	desiredSet := make(map[string]bool, len(desired))
	for _, l := range desired {
		desiredSet[l.Name] = true
	}

	for _, name := range lister.GetFunctionNames() {
		if !desiredSet[name] {
			deleter.Delete(name)
			r.logger.Info("reconcile: removed stale Lambda function", "function", name)
		}
	}

	ProvisionLambdas(desired, svc, "000000000000", "us-east-1", r.logger)
}

func (r *Reconciler) reconcileSQS(desired []SQSQueueDef) {
	svc, err := r.registry.Lookup("sqs")
	if err != nil {
		return
	}

	// SQS provisioning handles idempotent creates
	ProvisionSQSQueues(desired, svc, r.logger)
}

func (r *Reconciler) reconcileSNS(desired []SNSTopicDef) {
	svc, err := r.registry.Lookup("sns")
	if err != nil {
		return
	}

	ProvisionSNSTopics(desired, svc, r.logger)
}

func (r *Reconciler) reconcileS3(desired []S3BucketDef) {
	svc, err := r.registry.Lookup("s3")
	if err != nil {
		return
	}

	ProvisionS3Buckets(desired, svc, r.logger)
}

func (r *Reconciler) reconcileCognito(desired []CognitoDef) {
	svc, err := r.registry.Lookup("cognito-idp")
	if err != nil {
		return
	}

	ProvisionCognitoPools(desired, svc, r.logger)
}

func (r *Reconciler) reconcileAPIGateway(desired []APIGatewayDef) {
	svc, err := r.registry.Lookup("apigateway")
	if err != nil {
		return
	}

	ProvisionAPIGateways(desired, svc, r.logger)
}
