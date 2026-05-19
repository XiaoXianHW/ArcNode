package api

import (
	"io"
)

func copyReader(w io.Writer, r io.Reader) (int64, error) {
	return io.Copy(w, r)
}
