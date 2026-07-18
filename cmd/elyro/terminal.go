package main

import (
	"io"
	"os"
)

func stdinFile(reader io.Reader) *os.File {
	file, _ := reader.(*os.File)
	return file
}

func stdoutFile(writer io.Writer) *os.File {
	file, _ := writer.(*os.File)
	return file
}

func isTerminalFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
