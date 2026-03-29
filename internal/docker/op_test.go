// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build integration

package docker

import (
	"context"
	"fmt"
	"time"
)

type State struct {
	ContainerID string
	Info        ContainerInfo
	ExitCode    int
	NetworkID   string
}

type Op struct {
	Name string
	Fn   func(context.Context, *Client, *State) error
}

func Run(ctx context.Context, c *Client, ops ...Op) (*State, error) {
	state := &State{}
	for _, op := range ops {
		if err := op.Fn(ctx, c, state); err != nil {
			return state, fmt.Errorf("%s: %w", op.Name, err)
		}
	}
	return state, nil
}

func Ping() Op {
	return Op{"ping", func(ctx context.Context, c *Client, s *State) error {
		return c.Ping(ctx)
	}}
}

func CreateContainer(cfg *ContainerConfig) Op {
	return Op{"create", func(ctx context.Context, c *Client, s *State) error {
		id, err := c.CreateContainer(ctx, cfg)
		s.ContainerID = id
		return err
	}}
}

func StartContainer() Op {
	return Op{"start", func(ctx context.Context, c *Client, s *State) error {
		return c.StartContainer(ctx, s.ContainerID)
	}}
}

func StopContainer(timeout time.Duration) Op {
	return Op{"stop", func(ctx context.Context, c *Client, s *State) error {
		return c.StopContainer(ctx, s.ContainerID, timeout)
	}}
}

func RemoveContainer(force bool) Op {
	return Op{"remove", func(ctx context.Context, c *Client, s *State) error {
		return c.RemoveContainer(ctx, s.ContainerID, force)
	}}
}

func InspectContainer() Op {
	return Op{"inspect", func(ctx context.Context, c *Client, s *State) error {
		info, err := c.InspectContainer(ctx, s.ContainerID)
		s.Info = info
		return err
	}}
}

func ContainerLogs(follow bool) Op {
	return Op{"logs", func(ctx context.Context, c *Client, s *State) error {
		logs, err := c.ContainerLogs(ctx, s.ContainerID, follow)
		if err != nil {
			return err
		}
		return logs.Close()
	}}
}

func WaitContainer() Op {
	return Op{"wait", func(ctx context.Context, c *Client, s *State) error {
		code, err := c.WaitContainer(ctx, s.ContainerID)
		s.ExitCode = code
		return err
	}}
}

func CreateNetwork(cfg *NetworkConfig) Op {
	return Op{"create-network", func(ctx context.Context, c *Client, s *State) error {
		id, err := c.CreateNetwork(ctx, cfg)
		s.NetworkID = id
		return err
	}}
}

func RemoveNetwork() Op {
	return Op{"remove-network", func(ctx context.Context, c *Client, s *State) error {
		return c.RemoveNetwork(ctx, s.NetworkID)
	}}
}

func GetNetworkID(name string) Op {
	return Op{"get-network-id", func(ctx context.Context, c *Client, s *State) error {
		id, err := c.GetNetworkID(ctx, name)
		if err != nil {
			return err
		}
		if id != s.NetworkID {
			return fmt.Errorf("network ID mismatch: got %s, want %s", id, s.NetworkID)
		}
		return nil
	}}
}

func ConnectContainerToNetwork(ip string) Op {
	return Op{"connect-container", func(ctx context.Context, c *Client, s *State) error {
		return c.ConnectContainer(ctx, s.NetworkID, s.ContainerID, ip)
	}}
}
