// DynamoStore provides DynamoDB-backed persistence for admin API state
// (dashboards, saved views, deploy events) as an alternative to file-backed
// persistence via SetPersistDir.
package admin

import (
	"context"
	"log/slog"

	ds "github.com/neureaux/cloudmock/pkg/dynamostore"
)

const (
	featureDashboard = "DASHBOARD"
	featureView      = "VIEW"
	featureDeploy    = "DEPLOY"
)

// DynamoStore persists admin state to DynamoDB.
type DynamoStore struct {
	db *ds.Store
}

// NewDynamoStore creates a DynamoDB-backed admin store.
func NewDynamoStore(db *ds.Store) *DynamoStore {
	return &DynamoStore{db: db}
}

// LoadDashboards returns all dashboards from DynamoDB.
func (d *DynamoStore) LoadDashboards(ctx context.Context) []Dashboard {
	var dashboards []Dashboard
	if err := d.db.List(ctx, featureDashboard, &dashboards); err != nil {
		slog.Error("dynamostore: load dashboards", "error", err)
		return nil
	}
	return dashboards
}

// SaveDashboard persists a single dashboard.
func (d *DynamoStore) SaveDashboard(ctx context.Context, dash Dashboard) error {
	return d.db.Put(ctx, featureDashboard, dash.ID, dash)
}

// DeleteDashboard removes a dashboard.
func (d *DynamoStore) DeleteDashboard(ctx context.Context, id string) error {
	return d.db.Delete(ctx, featureDashboard, id)
}

// LoadViews returns all saved views from DynamoDB.
func (d *DynamoStore) LoadViews(ctx context.Context) []SavedView {
	var views []SavedView
	if err := d.db.List(ctx, featureView, &views); err != nil {
		slog.Error("dynamostore: load views", "error", err)
		return nil
	}
	return views
}

// SaveView persists a single saved view.
func (d *DynamoStore) SaveView(ctx context.Context, view SavedView) error {
	return d.db.Put(ctx, featureView, view.ID, view)
}

// DeleteView removes a saved view.
func (d *DynamoStore) DeleteView(ctx context.Context, id string) error {
	return d.db.Delete(ctx, featureView, id)
}

// LoadDeploys returns all deploy events from DynamoDB.
func (d *DynamoStore) LoadDeploys(ctx context.Context) []DeployEvent {
	var deploys []DeployEvent
	if err := d.db.List(ctx, featureDeploy, &deploys); err != nil {
		slog.Error("dynamostore: load deploys", "error", err)
		return nil
	}
	return deploys
}

// SaveDeploy persists a single deploy event.
func (d *DynamoStore) SaveDeploy(ctx context.Context, deploy DeployEvent) error {
	return d.db.Put(ctx, featureDeploy, deploy.ID, deploy)
}
