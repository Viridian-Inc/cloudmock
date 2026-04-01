package ce

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ServicePricing holds realistic AWS pricing info for mock cost generation.
type ServicePricing struct {
	Name     string
	BaseCost float64 // Monthly base cost in USD
	Unit     string  // Pricing unit description
	UnitCost float64 // Cost per unit
}

// DefaultServicePricing returns realistic AWS service pricing data.
func DefaultServicePricing() []ServicePricing {
	return []ServicePricing{
		{Name: "Amazon Elastic Compute Cloud - Compute", BaseCost: 245.67, Unit: "Hrs", UnitCost: 0.0116},
		{Name: "Amazon Simple Storage Service", BaseCost: 45.23, Unit: "GB-Mo", UnitCost: 0.023},
		{Name: "Amazon Relational Database Service", BaseCost: 189.45, Unit: "Hrs", UnitCost: 0.017},
		{Name: "AWS Lambda", BaseCost: 12.34, Unit: "Requests", UnitCost: 0.0000002},
		{Name: "Amazon DynamoDB", BaseCost: 34.56, Unit: "ReadRequestUnits", UnitCost: 0.00000025},
		{Name: "Amazon CloudFront", BaseCost: 23.45, Unit: "GB", UnitCost: 0.085},
		{Name: "Amazon Simple Queue Service", BaseCost: 5.67, Unit: "Requests", UnitCost: 0.0000004},
		{Name: "AWS Key Management Service", BaseCost: 3.45, Unit: "Requests", UnitCost: 0.03},
		{Name: "Amazon Route 53", BaseCost: 2.50, Unit: "Queries", UnitCost: 0.0000004},
		{Name: "Amazon Elastic Container Service", BaseCost: 78.90, Unit: "Hrs", UnitCost: 0.04048},
		{Name: "Amazon Kinesis", BaseCost: 15.20, Unit: "ShardHrs", UnitCost: 0.015},
		{Name: "Amazon ElastiCache", BaseCost: 52.30, Unit: "Hrs", UnitCost: 0.017},
		{Name: "Amazon OpenSearch Service", BaseCost: 88.10, Unit: "Hrs", UnitCost: 0.044},
		{Name: "AWS Step Functions", BaseCost: 4.50, Unit: "Transitions", UnitCost: 0.000025},
		{Name: "Amazon SNS", BaseCost: 1.20, Unit: "Requests", UnitCost: 0.0000005},
	}
}

// Store manages Cost Explorer mock data.
type Store struct {
	mu           sync.RWMutex
	accountID    string
	region       string
	rng          *rand.Rand
	serviceCount int // Number of active services (from locator, if known)
}

// NewStore returns a new Cost Explorer Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:    accountID,
		region:       region,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		serviceCount: 0,
	}
}

// SetServiceCount updates the number of active services, which scales cost generation.
func (s *Store) SetServiceCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceCount = count
}

// CostResult represents a cost and usage result for a time period.
type CostResult struct {
	TimePeriod map[string]string
	Total      map[string]map[string]string
	Groups     []CostGroup
}

// CostGroup represents a grouped cost result.
type CostGroup struct {
	Keys    []string
	Metrics map[string]map[string]string
}

// GenerateCostAndUsage generates realistic mock cost data for the given time range.
func (s *Store) GenerateCostAndUsage(start, end string, granularity string, groupBy []map[string]string) []CostResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime, _ := time.Parse("2006-01-02", start)
	endTime, _ := time.Parse("2006-01-02", end)

	pricing := DefaultServicePricing()

	// Scale costs based on active service count if known.
	scaleFactor := 1.0
	if s.serviceCount > 0 {
		scaleFactor = float64(s.serviceCount) / 10.0
		if scaleFactor < 0.5 {
			scaleFactor = 0.5
		}
	}

	var results []CostResult

	current := startTime
	for current.Before(endTime) {
		var nextPeriod time.Time
		switch granularity {
		case "DAILY":
			nextPeriod = current.AddDate(0, 0, 1)
		case "MONTHLY":
			nextPeriod = current.AddDate(0, 1, 0)
		default: // HOURLY
			nextPeriod = current.Add(time.Hour)
		}
		if nextPeriod.After(endTime) {
			nextPeriod = endTime
		}

		totalAmount := 0.0
		var groups []CostGroup

		for _, svc := range pricing {
			variation := 0.8 + s.rng.Float64()*0.4
			cost := svc.BaseCost * variation * scaleFactor
			if granularity == "DAILY" {
				cost /= 30
			} else if granularity == "HOURLY" {
				cost /= 720
			}
			cost = math.Round(cost*100) / 100
			totalAmount += cost

			if len(groupBy) > 0 {
				groups = append(groups, CostGroup{
					Keys: []string{svc.Name},
					Metrics: map[string]map[string]string{
						"UnblendedCost": {
							"Amount": fmt.Sprintf("%.2f", cost),
							"Unit":   "USD",
						},
						"BlendedCost": {
							"Amount": fmt.Sprintf("%.2f", cost*0.95),
							"Unit":   "USD",
						},
						"UsageQuantity": {
							"Amount": fmt.Sprintf("%.4f", cost/svc.UnitCost),
							"Unit":   svc.Unit,
						},
					},
				})
			}
		}

		result := CostResult{
			TimePeriod: map[string]string{
				"Start": current.Format("2006-01-02"),
				"End":   nextPeriod.Format("2006-01-02"),
			},
			Total: map[string]map[string]string{
				"UnblendedCost": {
					"Amount": fmt.Sprintf("%.2f", totalAmount),
					"Unit":   "USD",
				},
				"BlendedCost": {
					"Amount": fmt.Sprintf("%.2f", totalAmount*0.95),
					"Unit":   "USD",
				},
			},
			Groups: groups,
		}

		results = append(results, result)
		current = nextPeriod
	}

	return results
}

// GenerateForecast generates a mock cost forecast, extrapolating from current pricing data.
func (s *Store) GenerateForecast(start, end string, granularity, metric string) ([]CostResult, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime, _ := time.Parse("2006-01-02", start)
	endTime, _ := time.Parse("2006-01-02", end)

	// Use total of default pricing as base for forecast.
	pricing := DefaultServicePricing()
	baseDailyTotal := 0.0
	for _, p := range pricing {
		baseDailyTotal += p.BaseCost / 30
	}

	// Scale with service count.
	scaleFactor := 1.0
	if s.serviceCount > 0 {
		scaleFactor = float64(s.serviceCount) / 10.0
		if scaleFactor < 0.5 {
			scaleFactor = 0.5
		}
	}
	baseDailyTotal *= scaleFactor

	totalForecast := 0.0
	var results []CostResult

	current := startTime
	for current.Before(endTime) {
		var nextPeriod time.Time
		switch granularity {
		case "DAILY":
			nextPeriod = current.AddDate(0, 0, 1)
		case "MONTHLY":
			nextPeriod = current.AddDate(0, 1, 0)
		default:
			nextPeriod = current.Add(time.Hour)
		}
		if nextPeriod.After(endTime) {
			nextPeriod = endTime
		}

		variation := 0.85 + s.rng.Float64()*0.3
		amount := baseDailyTotal * variation
		if granularity == "MONTHLY" {
			amount *= 30
		}
		amount = math.Round(amount*100) / 100
		totalForecast += amount

		results = append(results, CostResult{
			TimePeriod: map[string]string{
				"Start": current.Format("2006-01-02"),
				"End":   nextPeriod.Format("2006-01-02"),
			},
			Total: map[string]map[string]string{
				metric: {
					"Amount": fmt.Sprintf("%.2f", amount),
					"Unit":   "USD",
				},
			},
		})
		current = nextPeriod
	}

	return results, fmt.Sprintf("%.2f", totalForecast)
}

// GetDimensionValues returns mock dimension values.
func (s *Store) GetDimensionValues(dimension string) []string {
	switch dimension {
	case "SERVICE":
		pricing := DefaultServicePricing()
		names := make([]string, 0, len(pricing))
		for _, p := range pricing {
			names = append(names, p.Name)
		}
		return names
	case "REGION":
		return []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	case "LINKED_ACCOUNT":
		return []string{s.accountID}
	case "INSTANCE_TYPE":
		return []string{"t3.micro", "t3.small", "t3.medium", "m5.large", "m5.xlarge", "r5.large"}
	case "USAGE_TYPE":
		return []string{"BoxUsage:t3.micro", "BoxUsage:m5.large", "DataTransfer-Out-Bytes", "Requests-Tier1", "TimedStorage-ByteHrs"}
	default:
		return []string{}
	}
}

// GetTags returns mock tag keys and values.
func (s *Store) GetTags() []string {
	return []string{"Environment", "Project", "Team", "CostCenter", "Owner"}
}
