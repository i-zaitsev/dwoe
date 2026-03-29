// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package docker provides a simple wrapper around the Docker API client to manage containers and networks.
// The wrapper implements the workspace.DockerClient contract, which helps to decouple the specific implementation
// from the subcommands logic and adds some convenience methods on top of the moby client methods.
// This package should stay light-weight and avoid getting too generic and only focus on the subset of Docker API
// necessary to run and manage tasks.
package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// Client is a thin wrapper around the Docker SDK client.
type Client struct {
	cli *client.Client
}

// NewClient creates a Client configured from environment variables.
func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// Ping verifies that the Docker daemon is reachable and negotiates the API version.
func (c *Client) Ping(ctx context.Context) error {
	slog.Info("docker: ping")

	_, err := c.cli.Ping(ctx, client.PingOptions{
		NegotiateAPIVersion: true,
		ForceNegotiate:      false,
	})

	return err
}

// BuildImage builds a Docker image from the given Dockerfile path and tags it.
func (c *Client) BuildImage(ctx context.Context, dockerfile, tag string, out io.Writer) error {
	slog.Info("docker: build-image", "dockerfile", dockerfile, "tag", tag)

	data, err := createDockerfileTar(dockerfile)
	if err != nil {
		return err
	}

	dataReader := bytes.NewReader(data)
	resp, err := c.cli.ImageBuild(
		ctx,
		dataReader,
		client.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Remove:     true,
			Tags:       []string{tag},
		},
	)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	return parseBuildOutput(resp.Body, out)
}

// CreateContainer creates a new container and returns its ID.
func (c *Client) CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	slog.Info("docker: create-container", "name", cfg.Name, "image", cfg.Image)

	opts := client.ContainerCreateOptions{
		Name: cfg.Name,
		Config: &container.Config{
			Image:      cfg.Image,
			Env:        cfg.Env,
			WorkingDir: cfg.WorkingDir,
		},
		HostConfig: &container.HostConfig{
			Binds: mountsToBinds(cfg.Mounts),
			Resources: container.Resources{
				NanoCPUs: int64(cfg.Resources.CPUs * 1e9),
				Memory:   cfg.Resources.Memory,
			},
		},
	}

	if cfg.NetworkID != "" {
		opts.NetworkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cfg.NetworkID: {},
			},
		}
	}

	res, err := c.cli.ContainerCreate(ctx, opts)
	if err != nil {
		return "", err
	}

	return res.ID, nil
}

// StartContainer starts an existing container by ID.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	slog.Info("docker: start-container", "id", containerID)

	_, err := c.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})

	return err
}

// StopContainer stops a running container with the given timeout.
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	slog.Info("docker: stop-container", "id", containerID, "timeout", timeout)

	timeoutSecs := int(timeout.Seconds())
	_, err := c.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{
		Timeout: &timeoutSecs,
	})

	return err
}

// RemoveContainer removes a container by ID, optionally forcing removal.
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	slog.Info("docker: remove-container", "id", containerID, "force", force)

	_, err := c.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force: force,
	})

	return err
}

// ContainerLogs returns a reader with the container's stdout and stderr.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	slog.Info("docker: container-logs", "id", containerID, "follow", follow)

	raw, err := c.cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		// TODO: this assumes TTY=false. The format must be configured or
		//       detected to avoid issues with stream multiplexing.
		defer raw.Close()
		_, copyErr := stdcopy.StdCopy(pw, pw, raw)
		_ = pw.CloseWithError(copyErr)
	}()

	return pr, nil
}

// WaitContainer blocks until the container exits and returns its exit code.
func (c *Client) WaitContainer(ctx context.Context, containerID string) (int, error) {
	slog.Info("docker: wait-container", "id", containerID)

	result := c.cli.ContainerWait(ctx, containerID, client.ContainerWaitOptions{})

	select {
	case resp := <-result.Result:
		code := int(resp.StatusCode)
		slog.Debug("wait: container exited", "code", code)
		return code, nil

	case err := <-result.Error:
		slog.Error("wait: error from Docker", "err", err)
		if err == nil {
			err = fmt.Errorf("wait: unexpected nil error from Docker")
		}
		return 0, err

	case <-ctx.Done():
		slog.Info("wait: context cancelled")
		return 0, ctx.Err()
	}
}

// InspectContainer returns status information about a container.
func (c *Client) InspectContainer(ctx context.Context, containerID string) (ContainerInfo, error) {
	slog.Info("docker: inspect-container", "id", containerID)

	res, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return ContainerInfo{}, err
	}

	return ContainerInfo{
		ID:       res.Container.ID,
		Name:     res.Container.Name,
		Image:    res.Container.Image,
		Status:   string(res.Container.State.Status),
		ExitCode: res.Container.State.ExitCode,
	}, nil
}

// CreateNetwork creates a bridge network. If a network with the same name already exists, its ID is returned.
func (c *Client) CreateNetwork(ctx context.Context, cfg *NetworkConfig) (string, error) {
	slog.Info("docker: create-network", "name", cfg.Name, "subnet", cfg.Subnet)

	opts := client.NetworkCreateOptions{
		Driver: "bridge",
	}

	if cfg.Subnet != "" {
		prefix, errPrefix := netip.ParsePrefix(cfg.Subnet)
		if errPrefix != nil {
			return "", fmt.Errorf("invalid subnet: %s: %w", cfg.Subnet, errPrefix)
		}
		gateway, errGateway := netip.ParseAddr(cfg.Gateway)
		if errGateway != nil {
			return "", fmt.Errorf("invalid gateway: %s: %w", cfg.Gateway, errGateway)
		}
		opts.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{
				{Subnet: prefix, Gateway: gateway},
			},
		}
	}

	resp, err := c.cli.NetworkCreate(ctx, cfg.Name, opts)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return c.GetNetworkID(ctx, cfg.Name)
		}
		return "", err
	}

	return resp.ID, nil
}

// RemoveNetwork removes a Docker network by ID.
func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	slog.Info("docker: remove-network", "id", networkID)

	_, err := c.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})

	return err
}

// GetNetworkID resolves a network name to its ID.
func (c *Client) GetNetworkID(ctx context.Context, name string) (string, error) {
	slog.Info("docker: get-network-id", "name", name)

	resp, err := c.cli.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}

	return resp.Network.ID, nil
}

// ConnectContainer attaches a container to a network with a static IP address.
func (c *Client) ConnectContainer(ctx context.Context, networkID, containerID, ipAddr string) error {
	slog.Info("docker: connect-container", "networkID", networkID, "containerID", containerID, "ipAddr", ipAddr)

	addr, err := netip.ParseAddr(ipAddr)
	if err != nil {
		return fmt.Errorf("cannot parse IP address: %s: %w", ipAddr, err)
	}

	var ipamCfg network.EndpointIPAMConfig
	if addr.Is4() {
		ipamCfg.IPv4Address = addr
	}
	if addr.Is6() {
		ipamCfg.IPv6Address = addr
	}
	_, err = c.cli.NetworkConnect(ctx, networkID, client.NetworkConnectOptions{
		Container: containerID,
		EndpointConfig: &network.EndpointSettings{
			IPAMConfig: &ipamCfg,
		},
	})

	return err
}

// Close releases the underlying Docker client resources.
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// BuildError is returned when a Docker image build fails.
type BuildError struct {
	Msg string
}

func (e *BuildError) Error() string {
	return e.Msg
}

func parseBuildOutput(r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	for {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return &BuildError{Msg: err.Error()}
		}
		if msg.Error != "" {
			return &BuildError{Msg: strings.TrimSpace(msg.Error)}
		}
		if msg.Stream != "" {
			_, _ = fmt.Fprint(w, msg.Stream)
		}
	}
}

func createDockerfileTar(dockerfile string) ([]byte, error) {
	data, err := os.ReadFile(dockerfile)
	if err != nil {
		return nil, fmt.Errorf("read dockerfile: %w", err)
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	success := false

	defer func() {
		if !success {
			_ = tw.Close()
		}
	}()

	if err = tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(data)),
	}); err != nil {
		return nil, err
	}
	if _, err = tw.Write(data); err != nil {
		return nil, err
	}
	if err = tw.Close(); err != nil {
		return nil, err
	}

	success = true

	return buf.Bytes(), nil
}

func mountsToBinds(mounts []Mount) []string {
	binds := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := m.Source + ":" + m.Target
		if m.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}
	return binds
}
