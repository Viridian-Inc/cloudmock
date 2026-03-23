package dataplane

import (
	"context"
	"time"

	"github.com/neureaux/cloudmock/pkg/config"
)

type DeployEvent struct {
	ID          string
	Service     string
	Version     string
	CommitSHA   string
	Author      string
	Description string
	DeployedAt  time.Time
	Metadata    map[string]string
}

type DeployFilter struct {
	Service string
	Limit   int
}

type SavedView struct {
	ID        string
	Name      string
	Filters   map[string]interface{}
	CreatedBy string
	CreatedAt time.Time
}

type ServiceEntry struct {
	Name        string
	ServiceType string
	GroupName   string
	Description string
	Owner       string
	RepoURL     string
}

type ConfigStore interface {
	GetConfig(ctx context.Context) (*config.Config, error)
	SetConfig(ctx context.Context, cfg *config.Config) error
	ListDeploys(ctx context.Context, filter DeployFilter) ([]DeployEvent, error)
	AddDeploy(ctx context.Context, deploy DeployEvent) error
	ListViews(ctx context.Context) ([]SavedView, error)
	SaveView(ctx context.Context, view SavedView) error
	DeleteView(ctx context.Context, id string) error
	ListServices(ctx context.Context) ([]ServiceEntry, error)
	UpsertService(ctx context.Context, svc ServiceEntry) error
}
