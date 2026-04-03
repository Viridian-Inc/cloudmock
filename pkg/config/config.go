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

// AuthConfig holds JWT-based RBAC authentication configuration.
type AuthConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Secret  string `yaml:"secret" json:"secret"`
}

// DataPlaneConfig holds data plane configuration for request/trace storage.
type DataPlaneConfig struct {
	Mode       string           `yaml:"mode" json:"mode"` // "local", "dynamodb", "production"
	DuckDB     DuckDBConfig     `yaml:"duckdb" json:"duckdb"`
	PostgreSQL PostgreSQLConfig `yaml:"postgresql" json:"postgresql"`
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
	OTel       OTelConfig       `yaml:"otel" json:"otel"`
	DynamoDB   DynamoDBStoreConfig `yaml:"dynamodb" json:"dynamodb"`
}

// DynamoDBStoreConfig holds DynamoDB persistence configuration for the
// single-table multi-tenant data store.
type DynamoDBStoreConfig struct {
	TableName string `yaml:"table_name" json:"table_name"` // default: "cloudmock-data"
	Region    string `yaml:"region" json:"region"`          // default: from AWS env/config
	Endpoint  string `yaml:"endpoint" json:"endpoint"`     // optional: for local DynamoDB
	TenantID  string `yaml:"tenant_id" json:"tenant_id"`   // default tenant for non-SaaS mode
}

// DuckDBConfig holds DuckDB database configuration.
type DuckDBConfig struct {
	Path string `yaml:"path" json:"path"` // default: "cloudmock.duckdb"
}

// PostgreSQLConfig holds PostgreSQL connection configuration.
type PostgreSQLConfig struct {
	URL string `yaml:"url" json:"url"`
}

// PrometheusConfig holds Prometheus connection configuration.
type PrometheusConfig struct {
	URL string `yaml:"url" json:"url"`
}

// OTelConfig holds OpenTelemetry configuration.
type OTelConfig struct {
	CollectorEndpoint string `yaml:"collector_endpoint" json:"collector_endpoint"`
	ServiceName       string `yaml:"service_name" json:"service_name"`
}

// RegressionConfig holds regression detection configuration.
type RegressionConfig struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	ScanInterval string `yaml:"scan_interval" json:"scan_interval"`
	Window       string `yaml:"window" json:"window"`
}

// IncidentConfig holds incident management configuration.
type IncidentConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	GroupWindow string `yaml:"group_window" json:"group_window"`
}

// MonitorConfig holds monitoring and alerting configuration.
type MonitorConfig struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	EvalInterval string `yaml:"eval_interval" json:"eval_interval"` // Go duration (default "30s")
}

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

// ChaosRule defines a fault injection rule loaded from the config file.
type ChaosRule struct {
	Service    string `yaml:"service"`
	Action     string `yaml:"action"`
	Type       string `yaml:"type"`
	ErrorCode  int    `yaml:"error_code"`
	ErrorMsg   string `yaml:"error_msg"`
	LatencyMs  int    `yaml:"latency_ms"`
	Percentage int    `yaml:"percentage"`
}

// ChaosConfig holds chaos/fault injection configuration.
type ChaosConfig struct {
	Rules []ChaosRule `yaml:"rules"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled           bool    `yaml:"enabled" json:"enabled"`
	RequestsPerSecond float64 `yaml:"requests_per_second" json:"requests_per_second"`
	Burst             int     `yaml:"burst" json:"burst"`
}

// CostConfig holds cost intelligence engine configuration.
type CostConfig struct {
	Pricing PricingConfig `yaml:"pricing" json:"pricing"`
}

// SaaSConfig holds hosted SaaS configuration.
type SaaSConfig struct {
	Enabled      bool               `yaml:"enabled"`
	Clerk        ClerkConfig        `yaml:"clerk"`
	Stripe       StripeConfig       `yaml:"stripe"`
	Provisioning ProvisioningConfig `yaml:"provisioning"`
	Cloudflare   CloudflareConfig   `yaml:"cloudflare"`
}

// ClerkConfig holds Clerk authentication configuration.
type ClerkConfig struct {
	SecretKey     string `yaml:"secret_key"`
	WebhookSecret string `yaml:"webhook_secret"`
	Domain        string `yaml:"domain"` // Clerk frontend API domain (e.g. "abc.clerk.accounts.dev")
}

// StripeConfig holds Stripe billing configuration.
type StripeConfig struct {
	SecretKey     string `yaml:"secret_key"`
	WebhookSecret string `yaml:"webhook_secret"`
	ProPriceID    string `yaml:"pro_price_id"`
	TeamPriceID   string `yaml:"team_price_id"`
}

// ProvisioningConfig holds Fly Machines provisioning configuration.
type ProvisioningConfig struct {
	FlyAPIToken        string `yaml:"fly_api_token"`
	FlyOrg             string `yaml:"fly_org"`
	FlyRegion          string `yaml:"fly_region"`
	Image              string `yaml:"image"`
	IdleTimeoutMinutes int    `yaml:"idle_timeout_minutes"`
	DataRetentionDays  int    `yaml:"data_retention_days"`
}

// CloudflareConfig holds Cloudflare DNS configuration.
type CloudflareConfig struct {
	APIToken string `yaml:"api_token"`
	ZoneID   string `yaml:"zone_id"`
}

// RUMConfig holds Real User Monitoring configuration.
type RUMConfig struct {
	Enabled    bool    `yaml:"enabled" json:"enabled"`
	SampleRate float64 `yaml:"sample_rate" json:"sample_rate"` // 0.0–1.0
	MaxEvents  int     `yaml:"max_events" json:"max_events"`   // circular buffer capacity
}

// OTLPConfig holds OTLP ingestion endpoint configuration.
type OTLPConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Port    int  `yaml:"port" json:"port"`
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
	Auth        AuthConfig               `yaml:"auth"`
	DataPlane   DataPlaneConfig          `yaml:"dataplane"`
	Regression  RegressionConfig         `yaml:"regression"`
	Cost        CostConfig               `yaml:"cost" json:"cost"`
	Incidents   IncidentConfig           `yaml:"incidents" json:"incidents"`
	Monitor     MonitorConfig            `yaml:"monitor" json:"monitor"`
	RateLimit   RateLimitConfig          `yaml:"rate_limit" json:"rate_limit"`
	RUM         RUMConfig                `yaml:"rum" json:"rum"`
	OTLP        OTLPConfig               `yaml:"otlp" json:"otlp"`
	Chaos          ChaosConfig              `yaml:"chaos" json:"chaos"`
	IaCDir         string                   `yaml:"iac_dir" json:"iac_dir"` // Path to Pulumi/Terraform project
	IaCEnv         string                   `yaml:"iac_env" json:"iac_env"` // Environment name (dev/stage/prod)
	SaaS           SaaSConfig               `yaml:"saas"`
	Notifications  NotificationsConfig      `yaml:"notifications" json:"notifications"`
	SCM            SCMConfig                `yaml:"scm" json:"scm"`
	Services       map[string]ServiceConfig `yaml:"services"`
}

// SCMConfig holds source code management integration configuration.
type SCMConfig struct {
	Provider string          `yaml:"provider" json:"provider"`
	Token    string          `yaml:"token" json:"-"` // never serialize the token
	Repos    []SCMRepoConfig `yaml:"repos" json:"repos"`
}

// SCMRepoConfig maps a repository to path-strip rules.
type SCMRepoConfig struct {
	Owner      string `yaml:"owner" json:"owner"`
	Repo       string `yaml:"repo" json:"repo"`
	PathPrefix string `yaml:"path_prefix" json:"path_prefix"`
}

// NotificationsConfig holds alert routing configuration.
type NotificationsConfig struct {
	Channels []NotifyChannelConfig `yaml:"channels" json:"channels"`
	Routes   []NotifyRouteConfig   `yaml:"routes" json:"routes"`
}

// NotifyChannelConfig defines a notification channel in config.
type NotifyChannelConfig struct {
	Type       string `yaml:"type" json:"type"`
	Name       string `yaml:"name" json:"name"`
	WebhookURL string `yaml:"webhook_url,omitempty" json:"webhook_url,omitempty"`
	RoutingKey string `yaml:"routing_key,omitempty" json:"routing_key,omitempty"`
	SMTPHost   string `yaml:"smtp_host,omitempty" json:"smtp_host,omitempty"`
	SMTPPort   int    `yaml:"smtp_port,omitempty" json:"smtp_port,omitempty"`
	Username   string `yaml:"username,omitempty" json:"username,omitempty"`
	Password   string `yaml:"password,omitempty" json:"password,omitempty"`
	From       string `yaml:"from,omitempty" json:"from,omitempty"`
	To         string `yaml:"to,omitempty" json:"to,omitempty"` // comma-separated
}

// NotifyRouteConfig defines a routing rule in config.
type NotifyRouteConfig struct {
	Name     string                `yaml:"name" json:"name"`
	Match    NotifyRouteMatchConfig `yaml:"match,omitempty" json:"match,omitempty"`
	Channels []string              `yaml:"channels" json:"channels"` // channel names
}

// NotifyRouteMatchConfig defines match conditions in config.
type NotifyRouteMatchConfig struct {
	Services   []string `yaml:"services,omitempty" json:"services,omitempty"`
	Severities []string `yaml:"severities,omitempty" json:"severities,omitempty"`
	Types      []string `yaml:"types,omitempty" json:"types,omitempty"`
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
		Auth: AuthConfig{
			Enabled: false,
			Secret:  "cloudmock-dev-secret-change-in-production",
		},
		DataPlane: DataPlaneConfig{
			Mode: "local",
		},
		Regression: RegressionConfig{
			Enabled:      true,
			ScanInterval: "5m",
			Window:       "15m",
		},
		Cost: CostConfig{
			Pricing: DefaultPricingConfig(),
		},
		Incidents: IncidentConfig{
			Enabled:     true,
			GroupWindow: "5m",
		},
		Monitor: MonitorConfig{
			Enabled:      true,
			EvalInterval: "30s",
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerSecond: 100,
			Burst:             200,
		},
		RUM: RUMConfig{
			Enabled:    true,
			SampleRate: 1.0,
			MaxEvents:  10000,
		},
		OTLP: OTLPConfig{
			Enabled: true,
			Port:    4318,
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

	if cfg.DataPlane.Mode == "" {
		cfg.DataPlane.Mode = "local"
	}

	return cfg, nil
}

// ApplyEnv applies environment variable overrides to the Config.
func (c *Config) ApplyEnv() {
	if v := os.Getenv("CLOUDMOCK_PROFILE"); v != "" {
		c.Profile = v
	}
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
	if v := os.Getenv("CLOUDMOCK_GATEWAY_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Gateway.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_ADMIN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Admin.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_DASHBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Dashboard.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_DATAPLANE_MODE"); v != "" {
		c.DataPlane.Mode = v
	}
	if v := os.Getenv("CLOUDMOCK_DUCKDB_PATH"); v != "" {
		c.DataPlane.DuckDB.Path = v
	}
	if v := os.Getenv("CLOUDMOCK_POSTGRESQL_URL"); v != "" {
		c.DataPlane.PostgreSQL.URL = v
	}
	if v := os.Getenv("CLOUDMOCK_PROMETHEUS_URL"); v != "" {
		c.DataPlane.Prometheus.URL = v
	}
	if v := os.Getenv("CLOUDMOCK_OTEL_ENDPOINT"); v != "" {
		c.DataPlane.OTel.CollectorEndpoint = v
	}
	if v := os.Getenv("CLOUDMOCK_DYNAMODB_TABLE"); v != "" {
		c.DataPlane.DynamoDB.TableName = v
	}
	if v := os.Getenv("CLOUDMOCK_DYNAMODB_REGION"); v != "" {
		c.DataPlane.DynamoDB.Region = v
	}
	if v := os.Getenv("CLOUDMOCK_DYNAMODB_ENDPOINT"); v != "" {
		c.DataPlane.DynamoDB.Endpoint = v
	}
	if v := os.Getenv("CLOUDMOCK_DYNAMODB_TENANT_ID"); v != "" {
		c.DataPlane.DynamoDB.TenantID = v
	}
	if v := os.Getenv("CLOUDMOCK_OTLP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.OTLP.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_OTLP_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.OTLP.Enabled = b
		}
	}
	if v := os.Getenv("CLOUDMOCK_SAAS_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.SaaS.Enabled = b
		}
	}
	if v := os.Getenv("CLERK_SECRET_KEY"); v != "" {
		c.SaaS.Clerk.SecretKey = v
	}
	if v := os.Getenv("CLERK_WEBHOOK_SECRET"); v != "" {
		c.SaaS.Clerk.WebhookSecret = v
	}
	if v := os.Getenv("CLERK_DOMAIN"); v != "" {
		c.SaaS.Clerk.Domain = v
	}
	if v := os.Getenv("STRIPE_SECRET_KEY"); v != "" {
		c.SaaS.Stripe.SecretKey = v
	}
	if v := os.Getenv("STRIPE_WEBHOOK_SECRET"); v != "" {
		c.SaaS.Stripe.WebhookSecret = v
	}
	if v := os.Getenv("STRIPE_PRO_PRICE_ID"); v != "" {
		c.SaaS.Stripe.ProPriceID = v
	}
	if v := os.Getenv("STRIPE_TEAM_PRICE_ID"); v != "" {
		c.SaaS.Stripe.TeamPriceID = v
	}
	if v := os.Getenv("FLY_API_TOKEN"); v != "" {
		c.SaaS.Provisioning.FlyAPIToken = v
	}
	if v := os.Getenv("FLY_ORG"); v != "" {
		c.SaaS.Provisioning.FlyOrg = v
	}
	if v := os.Getenv("FLY_REGION"); v != "" {
		c.SaaS.Provisioning.FlyRegion = v
	}
	if v := os.Getenv("CLOUDFLARE_API_TOKEN"); v != "" {
		c.SaaS.Cloudflare.APIToken = v
	}
	if v := os.Getenv("CLOUDFLARE_ZONE_ID"); v != "" {
		c.SaaS.Cloudflare.ZoneID = v
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
