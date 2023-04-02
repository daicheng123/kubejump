package utils

import "io"

func IgnoreErrWriteString(writer io.Writer, s string) {
	_, _ = io.WriteString(writer, s)
}
