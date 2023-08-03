package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docker/docker/errdefs"

	"github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types/strslice"

	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"

	"github.com/docker/docker/client"
)

type TracingConfig struct {
	NetworkConfig   *network.NetworkingConfig
	HostConfig      *container.HostConfig
	ContainerConfig *container.Config
}

func newLokiConfig(logConfig container.LogConfig, restartPolicy container.RestartPolicy) (*TracingConfig, error) {
	lokiDataDir := filepath.Join(conf.ClashHome, "tracing/loki/data")
	stat, err := os.Stat(lokiDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(lokiDataDir, 0755); err != nil {
				return nil, fmt.Errorf("[tracing] failed to create loki data dir: %w", err)
			}
		}
	}
	if stat != nil && !stat.IsDir() {
		return nil, errors.New("[tracing] the loki data directory location already exists, but is not a directory")
	}

	return &TracingConfig{
		ContainerConfig: &container.Config{
			User:     "root",
			Image:    lokiImage,
			Hostname: lokiContainerName,
		},
		HostConfig: &container.HostConfig{
			LogConfig:     logConfig,
			NetworkMode:   "host",
			RestartPolicy: restartPolicy,
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: lokiDataDir,
					Target: "/var/lib/loki",
				},
				{
					Type:   mount.TypeBind,
					Source: filepath.Join(conf.ClashHome, "tracing/loki/config.yaml"),
					Target: "/etc/loki/local-config.yaml",
				},
			},
		},
	}, nil
}

func newVectorConfig(logConfig container.LogConfig, restartPolicy container.RestartPolicy) (*TracingConfig, error) {
	return &TracingConfig{
		ContainerConfig: &container.Config{
			Image:    vectorImage,
			Hostname: vectorContainerName,
		},
		HostConfig: &container.HostConfig{
			LogConfig:     logConfig,
			NetworkMode:   "host",
			RestartPolicy: restartPolicy,
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: filepath.Join(conf.ClashHome, "tracing/vector/vector.toml"),
					Target: "/etc/vector/vector.toml",
				},
			},
		},
	}, nil
}

func newTrafficScraperConfig(logConfig container.LogConfig, restartPolicy container.RestartPolicy, apiHost, apiPort, apiSecret string) (*TracingConfig, error) {
	return &TracingConfig{
		ContainerConfig: &container.Config{
			Image:    trafficScraperImage,
			Hostname: trafficScraperContainerName,
			Cmd: strslice.StrSlice{
				"-v",
				"--autoreconnect-delay-millis",
				"15000",
				fmt.Sprintf("autoreconnect:ws://%s:%s/traffic?token=%s", apiHost, apiPort, apiSecret),
				fmt.Sprintf("autoreconnect:tcp:%s:9000", apiHost),
			},
		},
		HostConfig: &container.HostConfig{
			LogConfig:     logConfig,
			NetworkMode:   "host",
			RestartPolicy: restartPolicy,
		},
	}, nil
}

func newTracingScraperConfig(logConfig container.LogConfig, restartPolicy container.RestartPolicy, apiHost, apiPort, apiSecret string) (*TracingConfig, error) {
	return &TracingConfig{
		ContainerConfig: &container.Config{
			Image:    tracingScraperImage,
			Hostname: tracingScraperContainerName,
			Cmd: strslice.StrSlice{
				"-v",
				"--autoreconnect-delay-millis",
				"15000",
				fmt.Sprintf("autoreconnect:ws://%s:%s/profile/tracing?token=%s", apiHost, apiPort, apiSecret),
				fmt.Sprintf("autoreconnect:tcp:%s:9000", apiHost),
			},
		},
		HostConfig: &container.HostConfig{
			LogConfig:     logConfig,
			NetworkMode:   "host",
			RestartPolicy: restartPolicy,
		},
	}, nil
}

func newGrafanaConfig(logConfig container.LogConfig, restartPolicy container.RestartPolicy) (*TracingConfig, error) {
	grafanaDataDir := filepath.Join(conf.ClashHome, "tracing/grafana/data")
	stat, err := os.Stat(grafanaDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(grafanaDataDir, 0755); err != nil {
				return nil, fmt.Errorf("[tracing] failed to create grafana data dir: %w", err)
			}
		}
	}
	if stat != nil && !stat.IsDir() {
		return nil, errors.New("[tracing] the grafana data directory location already exists, but is not a directory")
	}

	return &TracingConfig{
		ContainerConfig: &container.Config{
			User:     "root",
			Image:    grafanaImage,
			Hostname: grafanaContainerName,
		},
		HostConfig: &container.HostConfig{
			LogConfig:     logConfig,
			NetworkMode:   "host",
			RestartPolicy: restartPolicy,
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: grafanaDataDir,
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
			},
		},
	}, nil
}

func startTracing(ctx context.Context, conf TPClashConf, cc *ClashConf) error {

	apiHost := "127.0.0.1"
	apiPort := "9090"
	apiSecret := cc.Secret
	if cc.ExternalController != "" {
		_, port, err := net.SplitHostPort(cc.ExternalController)
		if err != nil {
			return fmt.Errorf("[tracing] failed to parse clash api address(external-controller): %w", err)
		}

		iport, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("[tracing] failed to parse clash api address(external-controller): %w", err)
		}
		if iport != 9090 {
			apiPort = port
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("[tracing] failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	logConfig := container.LogConfig{
		Type: "json-file",
		Config: map[string]string{
			"max-size": "5m",
			"max-file": "2",
			"labels":   "tpclash",
		},
	}

	restartPolicy := container.RestartPolicy{
		Name:              "on-failure",
		MaximumRetryCount: 3,
	}

	lokiConf, err := newLokiConfig(logConfig, restartPolicy)
	if err != nil {
		return err
	}
	vectorConf, err := newVectorConfig(logConfig, restartPolicy)
	if err != nil {
		return err
	}
	trafficScraperConf, err := newTrafficScraperConfig(logConfig, restartPolicy, apiHost, apiPort, apiSecret)
	if err != nil {
		return err
	}
	tracingScraperConf, err := newTracingScraperConfig(logConfig, restartPolicy, apiHost, apiPort, apiSecret)
	if err != nil {
		return err
	}
	grafanaConf, err := newGrafanaConfig(logConfig, restartPolicy)
	if err != nil {
		return err
	}

	for _, c := range []*TracingConfig{lokiConf, vectorConf, trafficScraperConf, tracingScraperConf, grafanaConf} {
		logrus.Debugf("[tracing] pulling docker image %s: %s", c.ContainerConfig.Hostname, c.ContainerConfig.Image)
		pullResp, err := cli.ImagePull(ctx, c.ContainerConfig.Image, types.ImagePullOptions{})
		if err != nil {
			return fmt.Errorf("[tracing] failed to pull container image: %s: %w", c.ContainerConfig.Hostname, err)
		}
		if conf.Debug {
			_, _ = io.Copy(os.Stdout, pullResp)
		} else {
			_, _ = io.Copy(io.Discard, pullResp)
		}

		logrus.Debugf("[tracing] creating docker container: %s", c.ContainerConfig.Hostname)
		createResp, err := cli.ContainerCreate(ctx, c.ContainerConfig, c.HostConfig, c.NetworkConfig, nil, c.ContainerConfig.Hostname)
		if err != nil {
			return fmt.Errorf("[tracing] failed to create container: %s: %w", c.ContainerConfig.Hostname, err)
		}

		logrus.Debugf("[tracing] staring docker container: %s", c.ContainerConfig.Hostname)
		err = cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
		if err != nil {
			return fmt.Errorf("[tracing] failed to start container: %s: %w", c.ContainerConfig.Hostname, err)
		}
	}

	return nil
}

func stopTracing(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("[tracing] failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	for _, name := range []string{grafanaContainerName, lokiContainerName, vectorContainerName, tracingScraperContainerName, trafficScraperContainerName} {
		logrus.Debugf("[tracing] remove docker containers: %s", name)
		err = cli.ContainerRemove(ctx, name, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			if _, ok := err.(errdefs.ErrNotFound); ok {
				continue
			}
			return fmt.Errorf("[tracing] failed to remove container: %s: %w", name, err)
		}
	}

	return nil
}
