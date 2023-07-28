package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"

	"github.com/docker/docker/client"
)

type TracingStackInfo struct {
	NetWorkID        string
	LokiID           string
	VectorID         string
	TrafficScraperID string
	TracingScraperID string
	GrafanaID        string
}

func startTracing(ctx context.Context, conf *TPClashConf) (*TracingStackInfo, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("[tracing] failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	netResp, err := cli.NetworkCreate(ctx, tracingNetworkName, types.NetworkCreate{
		Driver:     "bridge",
		Attachable: true,
	})
	if err != nil {
		return nil, fmt.Errorf("[tracing] failed to create tracing network: %w", err)
	}

	commonLogConfig := container.LogConfig{
		Type: "json",
		Config: map[string]string{
			"max-size": "10m",
			"max-file": "3",
			"labels":   "tpclash",
		},
	}

	commonRestartPolicy := container.RestartPolicy{
		Name:              "on-failure",
		MaximumRetryCount: 3,
	}

	lokiContainerConfig := &container.Config{
		Hostname: lokiHostname,
		User:     "root",
		Tty:      true,
		Image:    lokiImage,
		Labels: map[string]string{
			"io.github.tpclash": "true",
		},
	}
	lokiHostConfig := &container.HostConfig{
		LogConfig:     commonLogConfig,
		NetworkMode:   "default",
		RestartPolicy: commonRestartPolicy,
		Resources:     container.Resources{},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/loki/data"),
				Target: "/loki",
			},
		},
	}
	lokiNetworkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}

	cli.ContainerCreate(ctx)

}
