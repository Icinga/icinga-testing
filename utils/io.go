package utils

import (
	"bytes"
)

// LineWriter implements io.WriteCloser and calls the given callback for every line written to it.
type LineWriter struct {
	buf      []byte
	callback func([]byte)
}

func NewLineWriter(callback func([]byte)) *LineWriter {
	return &LineWriter{
		callback: callback,
	}
}

func (l *LineWriter) Write(in []byte) (int, error) {
	n := len(in)

	for {
		pos := bytes.IndexByte(in, '\n')
		if pos < 0 {
			break
		}

		if l.buf != nil {
			// buffered data from a previous call, append remainder of the line to the buffer,
			// call the callback and then clear the buffer
			l.buf = append(l.buf, in[:pos]...)
			l.callback(l.buf)
			l.buf = nil
		} else {
			// no buffered data from a previous call, just call the callback for the current line
			l.callback(in[:pos])
		}

		in = in[1+pos:]
	}

	if len(in) > 0 {
		// the current write did not end with a newline, buffer the beginning
		// of the current line for the next call to Write or Close
		newBuf := make([]byte, len(in))
		copy(newBuf, in)
		l.buf = newBuf
	}

	return n, nil
}

func (l *LineWriter) Close() error {
	if len(l.buf) > 0 {
		// pass the last line to the callback, even if it did not end with a newline
		l.callback(l.buf)
	}
	l.buf = nil
	return nil
}
