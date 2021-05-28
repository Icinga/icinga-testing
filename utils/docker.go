package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

func DockerNetworkName(ctx context.Context, client *client.Client, id string) (string, error) {
	net, err := client.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}
	return net.Name, nil
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

// DockerExec runs a command in a container, forwards its stdin/stdout/stderr to/from the reader/writers passed as
// arguments and waits for the command to complete with exit code 0.
func DockerExec(
	ctx context.Context, client *client.Client, logger *zap.Logger, containerId string,
	cmd []string, stdin io.Reader, stdout io.Writer, stderr io.Writer,
) error {
	logger = logger.With(zap.String("container-id", containerId), zap.Strings("container-exec-cmd", cmd))

	logger.Debug("executing command in container")

	// Always read stdout and stderr to know when the exec exits. If the caller is not interested, just discard it.
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	exec, err := client.ContainerExecCreate(ctx, containerId, types.ExecConfig{
		Cmd:          cmd,
		AttachStdin:  stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       true,
	})
	if err != nil {
		return fmt.Errorf("failed to create container exec: %w", err)
	}
	logger = logger.With(zap.String("container-exec-id", exec.ID))
	logger.Debug("created exec")

	g, gCtx := errgroup.WithContext(ctx)

	attach, err := client.ContainerExecAttach(gCtx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("failed to attach to container exec: %w", err)
	}
	logger.Debug("attached to exec")

	if stdin != nil {
		g.Go(func() error {
			_, err := io.Copy(attach.Conn, stdin)
			if err != nil {
				return err
			}
			return attach.CloseWrite()
		})
	}

	g.Go(func() error {
		for {
			// Docker multiplexes stdout and stderr over the same connection in frames with the following header:
			// [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}
			var header [8]byte
			_, err := io.ReadFull(attach.Reader, header[:])
			if err != nil {
				if err == io.EOF {
					return nil
				} else {
					return err
				}
			}
			streamType := header[0]
			frameLen := binary.BigEndian.Uint32(header[4:8])

			var destWriter io.Writer
			if streamType == 1 {
				destWriter = stdout
			} else if streamType == 2 {
				destWriter = stderr
			} else {
				return fmt.Errorf("received fragment for unknown stream %d", streamType)
			}

			if _, err := io.CopyN(destWriter, attach.Reader, int64(frameLen)); err != nil {
				return err
			}
		}
	})

	if err := g.Wait(); err != nil {
		return err
	}

	inspect, err := client.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return err
	}
	if inspect.Running {
		panic("command should no longer be running")
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", inspect.ExitCode)
	}

	return nil
}

// DockerImagePull pulls an image from the registry. If force=false, no pull is done if the image already exists.
func DockerImagePull(ctx context.Context, logger *zap.Logger, dockerClient *client.Client, image string, force bool) error {
	logger = logger.With(zap.String("docker-image", image))

	if !force {
		_, _, err := dockerClient.ImageInspectWithRaw(ctx, image)
		if err == nil {
			return nil // image exists, no need to pull
		} else if !client.IsErrNotFound(err) {
			return err
		}
	}

	pull, err := dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, err = io.Copy(NewLineWriter(func(line []byte) {
		logger.Debug("docker pull", zap.ByteString("line", line))
	}), pull)
	if err != nil {
		return err
	}

	return pull.Close()
}
