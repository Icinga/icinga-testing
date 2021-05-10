package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"log"
)

type LogWriter struct {
	prefix string
	buf    []byte
}

func NewLogWriter(prefix string) *LogWriter {
	return &LogWriter{prefix: prefix}
}

func (l *LogWriter) Write(b []byte) (int, error) {
	n := len(b)

	for {
		pos := bytes.IndexByte(b, '\n')
		if pos < 0 {
			break
		}

		log.Printf("[%s] %s%s", l.prefix, string(l.buf), string(b[:pos]))
		b = b[1+pos:]
		l.buf = nil
	}

	if len(b) > 0 {
		newBuf := make([]byte, len(b))
		copy(newBuf, b)
		l.buf = newBuf
	}

	return n, nil
}

func (l *LogWriter) Close() error {
	if len(l.buf) > 0 {
		log.Printf("[%s] %s", l.prefix, string(l.buf))
	}
	l.buf = nil
	return nil
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
