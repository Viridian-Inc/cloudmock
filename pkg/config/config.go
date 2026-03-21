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
