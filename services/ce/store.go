package ce

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Store manages Cost Explorer mock data.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string
	rng       *rand.Rand
}

// NewStore returns a new Cost Explorer Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID: accountID,
		region:    region,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
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

	var results []CostResult

	services := []struct {
		name    string
		baseCost float64
	}{
		{"Amazon Elastic Compute Cloud - Compute", 245.67},
		{"Amazon Simple Storage Service", 45.23},
		{"Amazon Relational Database Service", 189.45},
		{"AWS Lambda", 12.34},
		{"Amazon DynamoDB", 34.56},
		{"Amazon CloudFront", 23.45},
		{"Amazon Simple Queue Service", 5.67},
		{"AWS Key Management Service", 3.45},
		{"Amazon Route 53", 2.50},
		{"Amazon Elastic Container Service", 78.90},
	}

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

		for _, svc := range services {
			variation := 0.8 + s.rng.Float64()*0.4
			cost := svc.baseCost * variation
			if granularity == "DAILY" {
				cost /= 30
			} else if granularity == "HOURLY" {
				cost /= 720
			}
			cost = math.Round(cost*100) / 100
			totalAmount += cost

			if len(groupBy) > 0 {
				groups = append(groups, CostGroup{
					Keys: []string{svc.name},
					Metrics: map[string]map[string]string{
						"UnblendedCost": {
							"Amount": fmt.Sprintf("%.2f", cost),
							"Unit":   "USD",
						},
						"BlendedCost": {
							"Amount": fmt.Sprintf("%.2f", cost*0.95),
							"Unit":   "USD",
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

// GenerateForecast generates a mock cost forecast.
func (s *Store) GenerateForecast(start, end string, granularity, metric string) ([]CostResult, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime, _ := time.Parse("2006-01-02", start)
	endTime, _ := time.Parse("2006-01-02", end)

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

		baseDailyTotal := 21.50
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
		return []string{
			"Amazon Elastic Compute Cloud - Compute",
			"Amazon Simple Storage Service",
			"Amazon Relational Database Service",
			"AWS Lambda",
			"Amazon DynamoDB",
			"Amazon CloudFront",
			"Amazon Simple Queue Service",
			"AWS Key Management Service",
			"Amazon Route 53",
			"Amazon Elastic Container Service",
		}
	case "REGION":
		return []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	case "LINKED_ACCOUNT":
		return []string{s.accountID}
	case "INSTANCE_TYPE":
		return []string{"t3.micro", "t3.small", "t3.medium", "m5.large", "m5.xlarge", "r5.large"}
	default:
		return []string{}
	}
}

// GetTags returns mock tag keys and values.
func (s *Store) GetTags() []string {
	return []string{"Environment", "Project", "Team", "CostCenter", "Owner"}
}
