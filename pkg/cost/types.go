package cost

import (
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
)

// Type aliases for config pricing types so cost-package callers do not need to
// import config directly, and existing code referencing these types continues
// to compile without change.
type LambdaPricing = config.LambdaPricing
type DynamoDBPricing = config.DynamoDBPricing
type S3Pricing = config.S3Pricing
type SQSPricing = config.SQSPricing
type TransferPricing = config.TransferPricing

// PricingConfig is an alias for config.PricingConfig so callers can use
// cost.PricingConfig without importing the config package directly.
type PricingConfig = config.PricingConfig

// DefaultPricingConfig returns a PricingConfig with standard AWS pricing.
// It delegates to config.DefaultPricingConfig so pricing defaults are defined
// in a single place.
func DefaultPricingConfig() PricingConfig {
	return config.DefaultPricingConfig()
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
