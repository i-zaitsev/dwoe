// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package docker

// Mount describes a bind mount from host to container.
type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// Resources specify CPU and memory limits for a container.
type Resources struct {
	CPUs   float64
	Memory int64
}

// ContainerConfig holds the parameters for creating a new container.
type ContainerConfig struct {
	Image      string
	Name       string
	Env        []string
	Mounts     []Mount
	NetworkID  string
	Resources  Resources
	WorkingDir string
}

// ContainerInfo holds a subset of container inspection data.
type ContainerInfo struct {
	ID       string
	Name     string
	Image    string
	Status   string
	ExitCode int
}
