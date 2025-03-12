package logging

import (
	"fmt"
	"os"
	"sync"
)

type SerializedWriter struct {
	mu   *sync.Mutex
	file *os.File
}

func NewSerializedWriter(path string) (*SerializedWriter, error) {
	tmp, err := os.Create(path)

	if err != nil {
		return nil, err
	}

	return &SerializedWriter{
		mu:   &sync.Mutex{},
		file: tmp,
	}, nil
}

func (sw *SerializedWriter) Write(data []byte) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if _, err := sw.file.Write(data); err != nil {
		return err
	}

	return nil
}

func (sw *SerializedWriter) Writeln(data []byte) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if _, err := sw.file.Write(data); err != nil {
		return err
	}

	if _, err := sw.file.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

func (sw *SerializedWriter) Printf(format string, args ...any) error {
	s := fmt.Sprintf(format, args...)

	if _, err := sw.file.Write([]byte(s)); err != nil {
		return err
	}

	return nil
}

func (sw *SerializedWriter) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	return sw.file.Close()
}
