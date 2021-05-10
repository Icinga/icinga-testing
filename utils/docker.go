package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
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

// ForwardDockerContainerOutput attaches to a docker container and forwards all its output to a writer.
func ForwardDockerContainerOutput(
	ctx context.Context, client *client.Client, containerId string, logs bool, w io.Writer,
) error {
	stream, err := client.ContainerAttach(ctx, containerId, types.ContainerAttachOptions{
		Logs:   logs,
		Stderr: true,
		Stdout: true,
		Stream: true,
	})
	if err != nil {
		return err
	}

	go func() {
		r := stream.Reader
		for {
			// Docker multiplexes stdout and stderr over the same connection in frames with the following header:
			// [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}
			var header [8]byte
			_, err := io.ReadFull(r, header[:])
			if err != nil {
				if err == io.EOF {
					break
				} else {
					panic(err)
				}
			}
			frameLen := binary.BigEndian.Uint32(header[4:8])
			if _, err := io.CopyN(w, r, int64(frameLen)); err != nil {
				panic(err)
			}
		}
	}()

	return nil
}
