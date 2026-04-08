package target

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var imageMap = map[string]string{
	"cloudmock":      "ghcr.io/viridian-inc/cloudmock:latest",
	"localstack":     "localstack/localstack:latest",
	"localstack-pro": "localstack/localstack-pro:latest",
	"moto":           "motoserver/moto:latest",
}

type DockerTarget struct {
	name        string
	image       string
	port        int
	containerID string
	apiKey      string
	cli         *client.Client
}

func NewDockerTarget(name string, apiKey string) *DockerTarget {
	img := imageMap[name]
	return &DockerTarget{name: name, image: img, port: 4566, apiKey: apiKey}
}

func (d *DockerTarget) Name() string     { return d.name }
func (d *DockerTarget) Image() string    { return d.image }
func (d *DockerTarget) Port() int        { return d.port }
func (d *DockerTarget) Endpoint() string { return fmt.Sprintf("http://localhost:%d", d.port) }

func (d *DockerTarget) Start(ctx context.Context) error {
	var err error
	d.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	reader, err := d.cli.ImagePull(ctx, d.image, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull %s: %w", d.image, err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	env := []string{}
	if d.name == "cloudmock" {
		env = append(env, "CLOUDMOCK_PROFILE=full", "CLOUDMOCK_IAM_MODE=none")
	}
	if d.apiKey != "" && d.name == "localstack-pro" {
		env = append(env, "LOCALSTACK_API_KEY="+d.apiKey)
	}
	if d.name == "moto" {
		d.port = 5000
	}

	hostPort := fmt.Sprintf("%d", d.port)
	containerPort := "4566"
	if d.name == "moto" {
		containerPort = "5000"
	}
	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image: d.image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort + "/tcp"): struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(containerPort + "/tcp"): []nat.PortBinding{{HostPort: hostPort}},
		},
	}, nil, nil, fmt.Sprintf("bench-%s", d.name))
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	d.containerID = resp.ID

	if err := d.cli.ContainerStart(ctx, d.containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	return d.waitReady(ctx, 60*time.Second)
}

func (d *DockerTarget) Stop(ctx context.Context) error {
	if d.containerID == "" {
		return nil
	}
	timeout := 10
	d.cli.ContainerStop(ctx, d.containerID, container.StopOptions{Timeout: &timeout})
	return d.cli.ContainerRemove(ctx, d.containerID, container.RemoveOptions{Force: true})
}

func (d *DockerTarget) ResourceStats(ctx context.Context) (*Stats, error) {
	if d.containerID == "" {
		return nil, fmt.Errorf("container not running")
	}
	resp, err := d.cli.ContainerStatsOneShot(ctx, d.containerID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var statsJSON struct {
		MemoryStats struct {
			Usage uint64 `json:"usage"`
		} `json:"memory_stats"`
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statsJSON); err != nil {
		return nil, err
	}

	memMB := float64(statsJSON.MemoryStats.Usage) / 1024 / 1024
	cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(statsJSON.CPUStats.SystemCPUUsage - statsJSON.PreCPUStats.SystemCPUUsage)
	cpuPct := 0.0
	if sysDelta > 0 {
		cpuPct = (cpuDelta / sysDelta) * 100.0
	}

	return &Stats{MemoryMB: memMB, CPUPct: cpuPct}, nil
}

func (d *DockerTarget) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(d.Endpoint() + "/")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s: not ready after %s", d.name, timeout)
}
