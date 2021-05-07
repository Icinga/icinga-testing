package utils

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
)

func DockerContainerAddress(ctx context.Context, client *client.Client, id string) (string, error) {
	info, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}

	for _, net := range info.NetworkSettings.Networks {
		if net.IPAddress != "" {
			return net.IPAddress, nil
		}
	}

	return "", fmt.Errorf("no address found for container %s", id)
}
