package cost

import "time"

// LambdaPricing holds per-invocation pricing for AWS Lambda.
type LambdaPricing struct {
	PerGBSecond     float64 `json:"perGBSecond" yaml:"perGBSecond"`
	DefaultMemoryMB float64 `json:"defaultMemoryMB" yaml:"defaultMemoryMB"`
}

// DynamoDBPricing holds per-operation pricing for DynamoDB.
type DynamoDBPricing struct {
	PerRCU float64 `json:"perRCU" yaml:"perRCU"`
	PerWCU float64 `json:"perWCU" yaml:"perWCU"`
}

// S3Pricing holds per-request pricing for S3.
type S3Pricing struct {
	PerGET float64 `json:"perGET" yaml:"perGET"`
	PerPUT float64 `json:"perPUT" yaml:"perPUT"`
}

// SQSPricing holds per-request pricing for SQS.
type SQSPricing struct {
	PerRequest float64 `json:"perRequest" yaml:"perRequest"`
}

// TransferPricing holds data transfer pricing.
type TransferPricing struct {
	PerKB float64 `json:"perKB" yaml:"perKB"`
}

// PricingConfig holds all service pricing configurations.
type PricingConfig struct {
	Lambda       LambdaPricing   `json:"lambda" yaml:"lambda"`
	DynamoDB     DynamoDBPricing `json:"dynamodb" yaml:"dynamodb"`
	S3           S3Pricing       `json:"s3" yaml:"s3"`
	SQS          SQSPricing      `json:"sqs" yaml:"sqs"`
	DataTransfer TransferPricing `json:"dataTransfer" yaml:"dataTransfer"`
}

// DefaultPricingConfig returns a PricingConfig with standard AWS pricing.
func DefaultPricingConfig() PricingConfig {
	return PricingConfig{
		Lambda: LambdaPricing{
			PerGBSecond:     0.0000166667,
			DefaultMemoryMB: 128,
		},
		DynamoDB: DynamoDBPricing{
			PerRCU: 0.00000025,
			PerWCU: 0.00000125,
		},
		S3: S3Pricing{
			PerGET: 0.0000004,
			PerPUT: 0.000005,
		},
		SQS: SQSPricing{
			PerRequest: 0.0000004,
		},
		DataTransfer: TransferPricing{
			PerKB: 0.00000009,
		},
	}
}

// ServiceCost represents the aggregated cost for a single AWS service.
type ServiceCost struct {
	Service      string  `json:"service"`
	RequestCount int64   `json:"requestCount"`
	TotalCost    float64 `json:"totalCost"`
	AvgCost      float64 `json:"avgCost"`
}

// RouteCost represents the aggregated cost for a specific route (service + method + path).
type RouteCost struct {
	Service      string  `json:"service"`
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	RequestCount int64   `json:"requestCount"`
	TotalCost    float64 `json:"totalCost"`
	AvgCost      float64 `json:"avgCost"`
}

// TenantCost represents the aggregated cost for a specific tenant.
type TenantCost struct {
	TenantID     string  `json:"tenantId"`
	RequestCount int64   `json:"requestCount"`
	TotalCost    float64 `json:"totalCost"`
	AvgCost      float64 `json:"avgCost"`
}

// TimeBucket represents aggregated cost within a time window.
type TimeBucket struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	TotalCost    float64   `json:"totalCost"`
	RequestCount int64     `json:"requestCount"`
}
