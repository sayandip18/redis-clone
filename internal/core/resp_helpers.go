package core

import (
	"fmt"
	"io"

	"github.com/sayandip18/redis-clone/internal/resp"
)

func writeBulkArray(w io.Writer, values []string) error {
	var b []byte
	b = append(b, []byte(fmt.Sprintf("*%d\r\n", len(values)))...)
	for _, v := range values {
		b = append(b, resp.Encode(v, false)...)
	}
	_, err := w.Write(b)
	return err
}

func writeInt(w io.Writer, n int64) error {
	_, err := w.Write(resp.Encode(n, false))
	return err
}

func writeOK(w io.Writer) error {
	_, err := w.Write([]byte("+OK\r\n"))
	return err
}

func writeBulkOrNil(w io.Writer, v string, ok bool) error {
	if !ok {
		_, err := w.Write(RESP_NIL)
		return err
	}
	_, err := w.Write(resp.Encode(v, false))
	return err
}
