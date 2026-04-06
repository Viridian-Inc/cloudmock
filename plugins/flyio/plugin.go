package flyio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/plugin"
)

// Plugin implements the CloudMock plugin interface for Fly.io Machines API emulation.
// Supports apps, machines, volumes, and secrets management.
type Plugin struct {
	mu       sync.RWMutex
	apps     map[string]*App
	machines map[string]*Machine
	volumes  map[string]*Volume
	secrets  map[string]map[string]string // appName -> key -> value
	logger   *slog.Logger
}

type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Org       string    `json:"organization"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Machine struct {
	ID        string            `json:"id"`
	AppName   string            `json:"app_name"`
	Name      string            `json:"name"`
	State     string            `json:"state"`
	Region    string            `json:"region"`
	ImageRef  string            `json:"image_ref"`
	Env       map[string]string `json:"env,omitempty"`
	CPUs      int               `json:"guest_cpus"`
	MemoryMB  int               `json:"guest_memory_mb"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Volume struct {
	ID        string    `json:"id"`
	AppName   string    `json:"app_name"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	SizeGB    int       `json:"size_gb"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

func New() *Plugin {
	return &Plugin{
		apps:     make(map[string]*App),
		machines: make(map[string]*Machine),
		volumes:  make(map[string]*Volume),
		secrets:  make(map[string]map[string]string),
		logger:   slog.Default(),
	}
}

func (p *Plugin) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil
}

func (p *Plugin) Shutdown(_ context.Context) error {
	return nil
}

func (p *Plugin) HealthCheck(_ context.Context) (plugin.HealthStatus, string, error) {
	return plugin.HealthHealthy, "fly.io machines api emulation is healthy", nil
}

func (p *Plugin) Describe(_ context.Context) (*plugin.Descriptor, error) {
	return &plugin.Descriptor{
		Name:     "flyio",
		Version:  "1.0.0",
		Protocol: "fly-machines-api",
		Actions: []string{
			"ListApps", "CreateApp", "GetApp", "DeleteApp",
			"ListMachines", "CreateMachine", "GetMachine", "UpdateMachine",
			"StartMachine", "StopMachine", "DestroyMachine",
			"ListVolumes", "CreateVolume", "DeleteVolume",
			"ListSecrets", "SetSecrets", "UnsetSecrets",
		},
		APIPaths: []string{
			"/fly/v1/",
		},
		Metadata: map[string]string{
			"description": "Fly.io Machines API emulation for CloudMock",
			"docs":        "https://fly.io/docs/machines/api/",
		},
	}, nil
}

func (p *Plugin) HandleRequest(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	path := strings.TrimPrefix(req.Path, "/fly/v1")
	method := req.Method

	switch {
	// Apps
	case method == "GET" && path == "/apps":
		return p.listApps()
	case method == "POST" && path == "/apps":
		return p.createApp(req.Body)
	case method == "GET" && strings.HasPrefix(path, "/apps/") && !strings.Contains(path[6:], "/"):
		return p.getApp(strings.TrimPrefix(path, "/apps/"))
	case method == "DELETE" && strings.HasPrefix(path, "/apps/") && !strings.Contains(path[6:], "/"):
		return p.deleteApp(strings.TrimPrefix(path, "/apps/"))

	// Machines — /apps/{appName}/machines[/{machineID}[/start|stop]]
	case method == "GET" && matchPath(path, "/apps/*/machines"):
		return p.listMachines(extractSegment(path, 1))
	case method == "POST" && matchPath(path, "/apps/*/machines"):
		return p.createMachine(extractSegment(path, 1), req.Body)
	case method == "GET" && matchPath(path, "/apps/*/machines/*"):
		return p.getMachine(extractSegment(path, 3))
	case method == "POST" && matchPath(path, "/apps/*/machines/*/start"):
		return p.setMachineState(extractSegment(path, 3), "started")
	case method == "POST" && matchPath(path, "/apps/*/machines/*/stop"):
		return p.setMachineState(extractSegment(path, 3), "stopped")
	case method == "DELETE" && matchPath(path, "/apps/*/machines/*"):
		return p.destroyMachine(extractSegment(path, 3))

	// Volumes — /apps/{appName}/volumes[/{volumeID}]
	case method == "GET" && matchPath(path, "/apps/*/volumes"):
		return p.listVolumes(extractSegment(path, 1))
	case method == "POST" && matchPath(path, "/apps/*/volumes"):
		return p.createVolume(extractSegment(path, 1), req.Body)
	case method == "DELETE" && matchPath(path, "/apps/*/volumes/*"):
		return p.deleteVolume(extractSegment(path, 3))

	// Secrets — /apps/{appName}/secrets
	case method == "GET" && matchPath(path, "/apps/*/secrets"):
		return p.listSecrets(extractSegment(path, 1))
	case method == "POST" && matchPath(path, "/apps/*/secrets"):
		return p.setSecrets(extractSegment(path, 1), req.Body)

	default:
		return jsonResp(http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// --- Apps ---

func (p *Plugin) listApps() (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	apps := make([]*App, 0, len(p.apps))
	for _, a := range p.apps {
		apps = append(apps, a)
	}
	return jsonResp(http.StatusOK, apps)
}

func (p *Plugin) createApp(body []byte) (*plugin.Response, error) {
	var req struct {
		AppName string `json:"app_name"`
		OrgSlug string `json:"org_slug"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.AppName == "" {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "app_name required"})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.apps[req.AppName]; exists {
		return jsonResp(http.StatusConflict, map[string]string{"error": "app already exists"})
	}
	app := &App{
		ID:        fmt.Sprintf("fly-%s", randID(8)),
		Name:      req.AppName,
		Org:       req.OrgSlug,
		Status:    "deployed",
		CreatedAt: time.Now().UTC(),
	}
	p.apps[req.AppName] = app
	p.secrets[req.AppName] = make(map[string]string)
	return jsonResp(http.StatusCreated, app)
}

func (p *Plugin) getApp(name string) (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	app, ok := p.apps[name]
	if !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	return jsonResp(http.StatusOK, app)
}

func (p *Plugin) deleteApp(name string) (*plugin.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.apps[name]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	delete(p.apps, name)
	delete(p.secrets, name)
	// Remove associated machines and volumes
	for id, m := range p.machines {
		if m.AppName == name {
			delete(p.machines, id)
		}
	}
	for id, v := range p.volumes {
		if v.AppName == name {
			delete(p.volumes, id)
		}
	}
	return jsonResp(http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Machines ---

func (p *Plugin) listMachines(appName string) (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var machines []*Machine
	for _, m := range p.machines {
		if m.AppName == appName {
			machines = append(machines, m)
		}
	}
	if machines == nil {
		machines = []*Machine{}
	}
	return jsonResp(http.StatusOK, machines)
}

func (p *Plugin) createMachine(appName string, body []byte) (*plugin.Response, error) {
	var req struct {
		Name   string            `json:"name"`
		Region string            `json:"region"`
		Config struct {
			Image string            `json:"image"`
			Env   map[string]string `json:"env"`
			Guest struct {
				CPUs     int `json:"cpus"`
				MemoryMB int `json:"memory_mb"`
			} `json:"guest"`
		} `json:"config"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.apps[appName]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	if req.Region == "" {
		req.Region = "iad"
	}
	if req.Config.Guest.CPUs == 0 {
		req.Config.Guest.CPUs = 1
	}
	if req.Config.Guest.MemoryMB == 0 {
		req.Config.Guest.MemoryMB = 256
	}
	now := time.Now().UTC()
	m := &Machine{
		ID:        fmt.Sprintf("mach_%s", randID(12)),
		AppName:   appName,
		Name:      req.Name,
		State:     "started",
		Region:    req.Region,
		ImageRef:  req.Config.Image,
		Env:       req.Config.Env,
		CPUs:      req.Config.Guest.CPUs,
		MemoryMB:  req.Config.Guest.MemoryMB,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p.machines[m.ID] = m
	return jsonResp(http.StatusCreated, m)
}

func (p *Plugin) getMachine(id string) (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m, ok := p.machines[id]
	if !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "machine not found"})
	}
	return jsonResp(http.StatusOK, m)
}

func (p *Plugin) setMachineState(id, state string) (*plugin.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.machines[id]
	if !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "machine not found"})
	}
	m.State = state
	m.UpdatedAt = time.Now().UTC()
	return jsonResp(http.StatusOK, m)
}

func (p *Plugin) destroyMachine(id string) (*plugin.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.machines[id]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "machine not found"})
	}
	delete(p.machines, id)
	return jsonResp(http.StatusOK, map[string]string{"status": "destroyed"})
}

// --- Volumes ---

func (p *Plugin) listVolumes(appName string) (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var vols []*Volume
	for _, v := range p.volumes {
		if v.AppName == appName {
			vols = append(vols, v)
		}
	}
	if vols == nil {
		vols = []*Volume{}
	}
	return jsonResp(http.StatusOK, vols)
}

func (p *Plugin) createVolume(appName string, body []byte) (*plugin.Response, error) {
	var req struct {
		Name   string `json:"name"`
		Region string `json:"region"`
		SizeGB int    `json:"size_gb"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.apps[appName]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	if req.SizeGB == 0 {
		req.SizeGB = 1
	}
	vol := &Volume{
		ID:        fmt.Sprintf("vol_%s", randID(12)),
		AppName:   appName,
		Name:      req.Name,
		Region:    req.Region,
		SizeGB:    req.SizeGB,
		State:     "created",
		CreatedAt: time.Now().UTC(),
	}
	p.volumes[vol.ID] = vol
	return jsonResp(http.StatusCreated, vol)
}

func (p *Plugin) deleteVolume(id string) (*plugin.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.volumes[id]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "volume not found"})
	}
	delete(p.volumes, id)
	return jsonResp(http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Secrets ---

func (p *Plugin) listSecrets(appName string) (*plugin.Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	secs, ok := p.secrets[appName]
	if !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	// Return key names only, not values
	keys := make([]map[string]string, 0, len(secs))
	for k := range secs {
		keys = append(keys, map[string]string{"name": k, "digest": "sha256:***"})
	}
	return jsonResp(http.StatusOK, keys)
}

func (p *Plugin) setSecrets(appName string, body []byte) (*plugin.Response, error) {
	var secrets map[string]string
	if err := json.Unmarshal(body, &secrets); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.secrets[appName]; !ok {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "app not found"})
	}
	for k, v := range secrets {
		p.secrets[appName][k] = v
	}
	return jsonResp(http.StatusOK, map[string]string{"status": "secrets set"})
}

// --- Helpers ---

func jsonResp(code int, data any) (*plugin.Response, error) {
	body, _ := json.Marshal(data)
	return &plugin.Response{
		StatusCode: code,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func matchPath(path, pattern string) bool {
	pp := strings.Split(strings.Trim(path, "/"), "/")
	tp := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(pp) != len(tp) {
		return false
	}
	for i := range tp {
		if tp[i] != "*" && tp[i] != pp[i] {
			return false
		}
	}
	return true
}

func extractSegment(path string, index int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func randID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
