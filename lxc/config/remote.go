package config

import (
	"fmt"
	"strings"

	"github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared"
)

// Remote holds details for communication with a remote daemon
type Remote struct {
	Addr     string `yaml:"addr"`
	Public   bool   `yaml:"public"`
	Protocol string `yaml:"protocol,omitempty"`
	Static   bool   `yaml:"-"`
}

// ParseRemote splits remote and object
func (c *Config) ParseRemote(raw string) (string, string, error) {
	result := strings.SplitN(raw, ":", 2)
	if len(result) == 1 {
		return c.DefaultRemote, result[0], nil
	}

	_, ok := c.Remotes[result[0]]
	if !ok {
		return "", "", fmt.Errorf("The remote \"%s\" doesn't exist", result[0])
	}

	return result[0], result[1], nil
}

// GetContainerServer returns a ContainerServer struct for the remote
func (c *Config) GetContainerServer(name string) (lxd.ContainerServer, error) {
	// Get the remote
	remote, ok := c.Remotes[name]
	if !ok {
		return nil, fmt.Errorf("The remote \"%s\" doesn't exist", name)
	}

	// Sanity checks
	if remote.Public || remote.Protocol == "simplestreams" {
		return nil, fmt.Errorf("The remote isn't a private LXD server")
	}

	// Get connection arguments
	args := c.getConnectionArgs(name)

	// Unix socket
	if strings.HasPrefix(remote.Addr, "unix:") {
		d, err := lxd.ConnectLXDUnix(strings.TrimPrefix(strings.TrimPrefix(remote.Addr, "unix:"), "//"), &args)
		if err != nil {
			return nil, err
		}

		return d, nil
	}

	// HTTPs
	if args.TLSClientCert == "" || args.TLSClientKey == "" {
		return nil, fmt.Errorf("Missing TLS client certificate and key")
	}

	d, err := lxd.ConnectLXD(remote.Addr, &args)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// GetImageServer returns a ImageServer struct for the remote
func (c *Config) GetImageServer(name string) (lxd.ImageServer, error) {
	// Get the remote
	remote, ok := c.Remotes[name]
	if !ok {
		return nil, fmt.Errorf("The remote \"%s\" doesn't exist", name)
	}

	// Get connection arguments
	args := c.getConnectionArgs(name)

	// Unix socket
	if strings.HasPrefix(remote.Addr, "unix:") {
		d, err := lxd.ConnectLXDUnix(strings.TrimPrefix(strings.TrimPrefix(remote.Addr, "unix:"), "//"), &args)
		if err != nil {
			return nil, err
		}

		return d, nil
	}

	// HTTPs (simplestreams)
	if remote.Protocol == "simplestreams" {
		d, err := lxd.ConnectSimpleStreams(remote.Addr, &args)
		if err != nil {
			return nil, err
		}

		return d, nil
	}

	// HTTPs (LXD)
	d, err := lxd.ConnectPublicLXD(remote.Addr, &args)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (c *Config) getConnectionArgs(name string) lxd.ConnectionArgs {
	args := lxd.ConnectionArgs{
		UserAgent: c.UserAgent,
	}

	// Client certificate
	if !shared.PathExists(c.ConfigPath("client.crt")) {
		args.TLSClientCert = c.ConfigPath("client.crt")
	}

	// Client key
	if !shared.PathExists(c.ConfigPath("client.key")) {
		args.TLSClientKey = c.ConfigPath("client.key")
	}

	// Client CA
	if shared.PathExists(c.ConfigPath("client.ca")) {
		args.TLSCA = c.ConfigPath("client.ca")
	}

	// Server certificate
	if shared.PathExists(c.ServerCertPath(name)) {
		args.TLSServerCert = c.ServerCertPath(name)
	}

	return args
}
