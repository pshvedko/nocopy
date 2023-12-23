package io

import (
	"fmt"
	"io"
)

func WritePartHeader(w io.Writer, offset, length, size int64, boundary, mime string) (err error) {
	_, err = fmt.Fprintln(w)
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "--", boundary, "\n")
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w, "Content-Type:", mime)
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w, "Content-Range:", "bytes", fmt.Sprintf("%d-%d/%d", offset, offset+length-1, size))
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w)
	return
}

func WritePartFooter(w io.Writer, boundary string) (err error) {
	_, err = fmt.Fprintln(w)
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "--", boundary, "--\n")
	return
}
