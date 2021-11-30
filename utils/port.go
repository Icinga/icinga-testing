package utils

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"net"
	"strconv"
)

type PortDecision struct {
	dockerHost string
	exposed    string
	port       string
	remote     bool
}

func NewPortDecision(c *client.Client, port string) *PortDecision {
	url, err := client.ParseHostURL(c.DaemonHost())
	if err != nil {
		panic(err)
	}
	p := &PortDecision{port: port}
	if url.Scheme != "unix" {
		portNumber, err := GetFreePort()
		if err != nil {
			panic(err)
		}
		p.dockerHost = url.Hostname()
		p.exposed = strconv.Itoa(portNumber)
		p.remote = true
	} else {
		p.exposed = port
	}

	return p
}

func (p *PortDecision) Address(ctx context.Context, c *client.Client, id string) string {
	if p.remote {
		return p.dockerHost
	}

	return MustString(DockerContainerAddress(ctx, c, id))
}

func (p *PortDecision) Binding(ctx context.Context, c *client.Client, id string) (*PortBinding, error) {
	var port string
	host := p.Address(ctx, c, id)
	if p.remote {
		r, err := c.ContainerInspect(context.Background(), id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to inspect container")
		}

		defaultPort, err := nat.NewPort(nat.SplitProtoPort(p.port))
		if err != nil {
			return nil, errors.Wrap(err, "can't parse default port")
		}

		p, ok := r.NetworkSettings.Ports[defaultPort]
		if !ok {
			return nil, errors.New(fmt.Sprintf("default port %s not exposed", defaultPort))
		}
		port = p[0].HostPort
	} else {
		port = p.port
	}

	return &PortBinding{
		Host: host,
		Port: port,
	}, nil
}

func (p *PortDecision) Map() nat.PortMap {
	if p.remote {
		return nat.PortMap{
			nat.Port(fmt.Sprintf("%s/tcp", p.port)): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: p.exposed,
				},
			},
		}
	}

	return nil
}

func (p *PortDecision) Port() string {
	return p.exposed
}

func (p *PortDecision) Remote() bool {
	return p.remote
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

type PortBinding struct {
	Host string
	Port string
}
