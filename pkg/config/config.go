package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// IAMConfig holds IAM-related configuration.
type IAMConfig struct {
	Mode          string `yaml:"mode"`
	RootAccessKey string `yaml:"root_access_key"`
	RootSecretKey string `yaml:"root_secret_key"`
	SeedFile      string `yaml:"seed_file"`
}

// PersistenceConfig holds persistence-related configuration.
type PersistenceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// GatewayConfig holds gateway-related configuration.
type GatewayConfig struct {
	Port int `yaml:"port"`
}

// DashboardConfig holds dashboard-related configuration.
type DashboardConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

// AdminConfig holds admin API configuration.
type AdminConfig struct {
	Port int `yaml:"port"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ServiceConfig holds per-service configuration.
type ServiceConfig struct {
	Enabled  *bool    `yaml:"enabled"`
	Port     int      `yaml:"port"`
	Runtimes []string `yaml:"runtimes"`
}

// SLORule defines a latency SLO for a service/action.
type SLORule struct {
	Service   string  `yaml:"service" json:"service"`     // e.g. "dynamodb", "*" for all
	Action    string  `yaml:"action" json:"action"`       // e.g. "Query", "*" for all
	P50Ms     float64 `yaml:"p50_ms" json:"p50_ms"`       // target P50 latency
	P95Ms     float64 `yaml:"p95_ms" json:"p95_ms"`       // target P95 latency
	P99Ms     float64 `yaml:"p99_ms" json:"p99_ms"`       // target P99 latency
	ErrorRate float64 `yaml:"error_rate" json:"error_rate"` // max acceptable error rate (0.01 = 1%)
}

// SLOConfig holds SLO configuration.
type SLOConfig struct {
	Enabled bool      `yaml:"enabled" json:"enabled"`
	Rules   []SLORule `yaml:"rules" json:"rules"`
}

// AdminAuthConfig holds admin API authentication configuration.
type AdminAuthConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	APIKey  string `yaml:"api_key" json:"-"` // never serialize the key
}

// Config is the top-level configuration for cloudmock.
type Config struct {
	Region      string                   `yaml:"region"`
	AccountID   string                   `yaml:"account_id"`
	Profile     string                   `yaml:"profile"`
	IAM         IAMConfig                `yaml:"iam"`
	Persistence PersistenceConfig        `yaml:"persistence"`
	Gateway     GatewayConfig            `yaml:"gateway"`
	Dashboard   DashboardConfig          `yaml:"dashboard"`
	Admin       AdminConfig              `yaml:"admin"`
	Logging     LoggingConfig            `yaml:"logging"`
	SLO         SLOConfig                `yaml:"slo"`
	AdminAuth   AdminAuthConfig          `yaml:"admin_auth"`
	Services    map[string]ServiceConfig `yaml:"services"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		Region:    "us-east-1",
		AccountID: "000000000000",
		Profile:   "minimal",
		IAM: IAMConfig{
			Mode:          "enforce",
			RootAccessKey: "test",
			RootSecretKey: "test",
		},
		Persistence: PersistenceConfig{
			Enabled: false,
		},
		Gateway: GatewayConfig{
			Port: 4566,
		},
		Dashboard: DashboardConfig{
			Enabled: true,
			Port:    4500,
		},
		Admin: AdminConfig{
			Port: 4599,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		SLO: SLOConfig{
			Enabled: true,
			Rules: []SLORule{
				{Service: "*", Action: "*", P50Ms: 50, P95Ms: 200, P99Ms: 500, ErrorRate: 0.01},
			},
		},
	}
}

// LoadFromFile loads a Config from a YAML file, using Default() as the base.
func LoadFromFile(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ApplyEnv applies environment variable overrides to the Config.
func (c *Config) ApplyEnv() {
	if v := os.Getenv("CLOUDMOCK_REGION"); v != "" {
		c.Region = v
	}
	if v := os.Getenv("CLOUDMOCK_IAM_MODE"); v != "" {
		c.IAM.Mode = v
	}
	if v := os.Getenv("CLOUDMOCK_PERSIST"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Persistence.Enabled = b
		}
	}
	if v := os.Getenv("CLOUDMOCK_PERSIST_PATH"); v != "" {
		c.Persistence.Path = v
	}
	if v := os.Getenv("CLOUDMOCK_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("CLOUDMOCK_SERVICES"); v != "" {
		// Comma-separated list of services to enable
		if c.Services == nil {
			c.Services = make(map[string]ServiceConfig)
		}
		for _, svc := range strings.Split(v, ",") {
			svc = strings.TrimSpace(svc)
			if svc != "" {
				enabled := true
				c.Services[svc] = ServiceConfig{Enabled: &enabled}
			}
		}
	}
}

// minimalServices are enabled for the "minimal" profile.
var minimalServices = []string{
	"iam", "sts", "s3", "dynamodb", "sqs", "sns", "lambda", "cloudwatch-logs",
}

// standardServices are enabled for the "standard" profile (all tier 1).
var standardServices = []string{
	"iam", "sts", "s3", "dynamodb", "sqs", "sns", "lambda", "cloudwatch-logs",
	"rds", "cloudformation", "ec2", "ecr", "ecs", "secretsmanager", "ssm",
	"kinesis", "firehose", "events", "stepfunctions", "apigateway",
}

// EnabledServices returns the list of services enabled for the current profile.
// Returns nil for the "full" profile, meaning all services are enabled.
func (c *Config) EnabledServices() []string {
	switch c.Profile {
	case "minimal":
		return minimalServices
	case "standard":
		return standardServices
	case "full":
		return nil
	case "custom":
		var services []string
		for name, svc := range c.Services {
			if svc.Enabled == nil || *svc.Enabled {
				services = append(services, name)
			}
		}
		return services
	default:
		return minimalServices
	}
}
