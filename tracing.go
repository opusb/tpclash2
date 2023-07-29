package main

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types/strslice"

	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"

	"github.com/docker/docker/client"
)

func startTracing(ctx context.Context, conf TPClashConf, cc *ClashConf) (map[string]string, error) {

	apiHost := tplMainIP()
	apiPort := 9090
	if cc.ExternalController != "" {
		host, port, err := net.SplitHostPort(cc.ExternalController)
		if err != nil {
			logrus.Fatalf("[tracing] failed to parse clash api address(external-controller): %v", err)
		}
		apiPort, err = strconv.Atoi(port)
		if err != nil {
			logrus.Fatalf("[tracing] failed to parse clash api address(external-controller): %v", err)
		}
		if host == "127.0.0.1" || host != apiHost {

		}
	}

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

	hostConfig := &container.HostConfig{
		LogConfig: container.LogConfig{
			Type: "json-file",
			Config: map[string]string{
				"max-size": "5m",
				"max-file": "2",
				"labels":   "tpclash",
			},
		},
		NetworkMode: "bridge",
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3,
		},
		PortBindings: map[nat.Port][]nat.PortBinding{},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/loki/data"),
				Target: "/var/lib/loki",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/loki/config.yaml"),
				Target: "/etc/loki/local-config.yaml",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/grafana/data"),
				Target: "/var/lib/grafana",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/grafana/panels"),
				Target: "/etc/dashboards",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/grafana/provisioning/dashboards"),
				Target: "/etc/grafana/provisioning/dashboards",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/grafana/provisioning/datasources"),
				Target: "/etc/grafana/provisioning/datasources",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(conf.ClashHome, "tracing/vector/vector.toml"),
				Target: "/etc/vector/vector.toml",
			},
		},
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			tracingNetworkName: {
				NetworkID: netResp.ID,
			},
		},
	}

	stackContainers := []*container.Config{
		{
			Image:    lokiImage,
			User:     "root",
			Hostname: lokiContainerName,
		},
		{
			Image:    vectorImage,
			Hostname: vectorContainerName,
		},
		{
			Image:    trafficScraperImage,
			Hostname: trafficScraperContainerName,
			Cmd: strslice.StrSlice{
				"-v",
				"--autoreconnect-delay-millis",
				"15000",
				fmt.Sprintf("autoreconnect:ws://%s/traffic?token=%s", cc.ExternalController, cc.Secret),
				fmt.Sprintf("autoreconnect:tcp:%s:9000", vectorContainerName),
			},
		},
		{
			Image:    tracingScraperImage,
			Hostname: tracingScraperContainerName,
			Cmd: strslice.StrSlice{
				"-v",
				"--autoreconnect-delay-millis",
				"15000",
				fmt.Sprintf("autoreconnect:ws://%s/profile/tracing?token=%s", cc.ExternalController, cc.Secret),
				fmt.Sprintf("autoreconnect:tcp:%s:9000", vectorContainerName),
			},
		},
		{
			Image:        grafanaImage,
			Hostname:     grafanaContainerName,
			ExposedPorts: map[nat.Port]struct{}{},
		},
	}

	containerMap := make(map[string]string)
	for _, c := range stackContainers {
		createResp, err := cli.ContainerCreate(ctx, c, hostConfig, networkingConfig, nil, c.Hostname)
		if err != nil {
			return nil, fmt.Errorf("[tracing] failed to create container: %s: %w", c.Hostname, err)
		}

		err = cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
		if err != nil {
			return nil, fmt.Errorf("[tracing] failed to start container: %s: %w", c.Hostname, err)
		}
		containerMap[c.Hostname] = createResp.ID
	}

	return containerMap, nil
}

func stopTracing(ctx context.Context, containerMap map[string]string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("[tracing] failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	for k, v := range containerMap {
		err = cli.ContainerRemove(ctx, v, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return fmt.Errorf("[tracing] failed to remove container: %s: %w", k, err)
		}
	}

	return nil
}
