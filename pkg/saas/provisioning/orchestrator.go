package provisioning

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

// Orchestrator coordinates Fly Machine provisioning and DNS setup
// for tenant lifecycle events.
type Orchestrator struct {
	fly     *FlyClient
	dns     *CloudflareClient
	tenants tenant.Store
}

// NewOrchestrator creates a provisioning orchestrator.
func NewOrchestrator(fly *FlyClient, dns *CloudflareClient, tenants tenant.Store) *Orchestrator {
	return &Orchestrator{
		fly:     fly,
		dns:     dns,
		tenants: tenants,
	}
}

// Provision creates a Fly app, starts a machine, and adds a DNS record
// for the given tenant. It updates the tenant record with the Fly
// machine ID and app name on success.
func (o *Orchestrator) Provision(ctx context.Context, t *tenant.Tenant) error {
	appName := "cm-" + t.Slug

	slog.InfoContext(ctx, "provisioning tenant",
		"tenant_id", t.ID,
		"slug", t.Slug,
		"app_name", appName,
	)

	// 1. Create the Fly application.
	if err := o.fly.CreateApp(ctx, appName); err != nil {
		return fmt.Errorf("provision: create fly app: %w", err)
	}

	// 2. Create the machine with tenant-specific env vars.
	env := map[string]string{
		"CLOUDMOCK_AUTH_ENABLED": "true",
		"CLOUDMOCK_TENANT_ID":   t.ID,
		"CLOUDMOCK_TENANT_SLUG": t.Slug,
	}

	machineID, err := o.fly.CreateMachine(ctx, appName, env)
	if err != nil {
		// Best-effort cleanup: delete the app we just created.
		if cleanupErr := o.fly.DeleteApp(ctx, appName); cleanupErr != nil {
			slog.ErrorContext(ctx, "provision cleanup: failed to delete fly app",
				"app_name", appName,
				"error", cleanupErr,
			)
		}
		return fmt.Errorf("provision: create machine: %w", err)
	}

	// 3. Create the DNS CNAME record.
	dnsName := t.Slug + ".cloudmock.io"
	dnsTarget := appName + ".fly.dev"
	if err := o.dns.AddCNAME(ctx, dnsName, dnsTarget); err != nil {
		// Best-effort cleanup: destroy the machine and app.
		if cleanupErr := o.fly.DestroyMachine(ctx, appName, machineID); cleanupErr != nil {
			slog.ErrorContext(ctx, "provision cleanup: failed to destroy machine",
				"machine_id", machineID,
				"error", cleanupErr,
			)
		}
		if cleanupErr := o.fly.DeleteApp(ctx, appName); cleanupErr != nil {
			slog.ErrorContext(ctx, "provision cleanup: failed to delete fly app",
				"app_name", appName,
				"error", cleanupErr,
			)
		}
		return fmt.Errorf("provision: add DNS: %w", err)
	}

	// 4. Update the tenant record with infrastructure details.
	t.FlyAppName = appName
	t.FlyMachineID = machineID
	if err := o.tenants.Update(ctx, t); err != nil {
		slog.ErrorContext(ctx, "provision: tenant update failed, infrastructure was created",
			"tenant_id", t.ID,
			"app_name", appName,
			"machine_id", machineID,
			"error", err,
		)
		return fmt.Errorf("provision: update tenant: %w", err)
	}

	slog.InfoContext(ctx, "tenant provisioned",
		"tenant_id", t.ID,
		"app_name", appName,
		"machine_id", machineID,
		"dns", dnsName,
	)
	return nil
}

// Deprovision tears down the Fly machine, deletes the app, and removes
// the DNS record for the given tenant. It clears the Fly fields on the
// tenant record.
func (o *Orchestrator) Deprovision(ctx context.Context, t *tenant.Tenant) error {
	appName := t.FlyAppName
	machineID := t.FlyMachineID

	slog.InfoContext(ctx, "deprovisioning tenant",
		"tenant_id", t.ID,
		"slug", t.Slug,
		"app_name", appName,
		"machine_id", machineID,
	)

	var errs []error

	// 1. Destroy the Fly machine (if we have a machine ID).
	if machineID != "" {
		if err := o.fly.DestroyMachine(ctx, appName, machineID); err != nil {
			slog.ErrorContext(ctx, "deprovision: failed to destroy machine",
				"machine_id", machineID,
				"error", err,
			)
			errs = append(errs, fmt.Errorf("destroy machine: %w", err))
		}
	}

	// 2. Delete the Fly application (if we have an app name).
	if appName != "" {
		if err := o.fly.DeleteApp(ctx, appName); err != nil {
			slog.ErrorContext(ctx, "deprovision: failed to delete fly app",
				"app_name", appName,
				"error", err,
			)
			errs = append(errs, fmt.Errorf("delete app: %w", err))
		}
	}

	// 3. Remove the DNS record.
	dnsName := t.Slug + ".cloudmock.io"
	if err := o.dns.RemoveCNAME(ctx, dnsName); err != nil {
		slog.ErrorContext(ctx, "deprovision: failed to remove DNS record",
			"dns_name", dnsName,
			"error", err,
		)
		errs = append(errs, fmt.Errorf("remove DNS: %w", err))
	}

	// 4. Clear infrastructure fields on the tenant.
	t.FlyAppName = ""
	t.FlyMachineID = ""
	if err := o.tenants.Update(ctx, t); err != nil {
		slog.ErrorContext(ctx, "deprovision: tenant update failed",
			"tenant_id", t.ID,
			"error", err,
		)
		errs = append(errs, fmt.Errorf("update tenant: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("deprovision tenant %s: %d errors, first: %w", t.ID, len(errs), errs[0])
	}

	slog.InfoContext(ctx, "tenant deprovisioned",
		"tenant_id", t.ID,
		"slug", t.Slug,
	)
	return nil
}
