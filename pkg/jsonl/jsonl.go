// Package jsonl provides utilities for reading and writing JSONL (JSON Lines) files.
package jsonl

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
)

// Writer writes JSON objects as newline-delimited JSON.
type Writer struct {
	mu      sync.Mutex
	w       io.Writer
	encoder *json.Encoder
}

// NewWriter creates a new JSONL writer.
func NewWriter(w io.Writer) *Writer {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return &Writer{
		w:       w,
		encoder: encoder,
	}
}

// Write encodes and writes a single JSON object followed by a newline.
func (w *Writer) Write(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.encoder.Encode(v)
}

// Reader reads JSON objects from a newline-delimited JSON stream.
type Reader struct {
	scanner *bufio.Scanner
	err     error
}

// NewReader creates a new JSONL reader.
func NewReader(r io.Reader) *Reader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line size
	return &Reader{
		scanner: scanner,
	}
}

// Next returns the next line as raw JSON bytes. Returns nil when done or on error.
func (r *Reader) Next() []byte {
	if r.scanner.Scan() {
		return r.scanner.Bytes()
	}
	r.err = r.scanner.Err()
	return nil
}

// Decode decodes the next JSON object into v. Returns io.EOF when done.
func (r *Reader) Decode(v any) error {
	line := r.Next()
	if line == nil {
		if r.err != nil {
			return r.err
		}
		return io.EOF
	}
	return json.Unmarshal(line, v)
}

// Err returns any error encountered during scanning.
func (r *Reader) Err() error {
	return r.err
}

// FileWriter writes JSONL to a file with append support.
type FileWriter struct {
	*Writer
	file *os.File
}

// OpenFile opens or creates a JSONL file for appending.
func OpenFile(path string) (*FileWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileWriter{
		Writer: NewWriter(f),
		file:   f,
	}, nil
}

// Close closes the underlying file.
func (fw *FileWriter) Close() error {
	return fw.file.Close()
}

// Sync flushes the file to disk.
func (fw *FileWriter) Sync() error {
	return fw.file.Sync()
}

// ReadAll reads all objects from a JSONL file into a slice.
// The factory function creates new instances for each line.
func ReadAll[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var results []T
	reader := NewReader(f)
	for {
		var item T
		err := reader.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			return results, err
		}
		results = append(results, item)
	}
	return results, nil
}

// AppendOne appends a single JSON object to a JSONL file.
func AppendOne(path string, v any) error {
	fw, err := OpenFile(path)
	if err != nil {
		return err
	}
	defer fw.Close()

	if err := fw.Write(v); err != nil {
		return err
	}
	return fw.Sync()
}
