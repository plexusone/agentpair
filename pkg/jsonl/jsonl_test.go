package jsonl

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type testRecord struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	records := []testRecord{
		{Name: "first", Value: 1},
		{Name: "second", Value: 2},
	}

	for _, r := range records {
		if err := w.Write(r); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	expected := `{"name":"first","value":1}
{"name":"second","value":2}
`
	if buf.String() != expected {
		t.Errorf("unexpected output:\ngot:  %q\nwant: %q", buf.String(), expected)
	}
}

func TestReader(t *testing.T) {
	input := `{"name":"first","value":1}
{"name":"second","value":2}
`
	r := NewReader(bytes.NewBufferString(input))

	var records []testRecord
	for {
		var rec testRecord
		err := r.Decode(&rec)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		records = append(records, rec)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Name != "first" || records[0].Value != 1 {
		t.Errorf("first record mismatch: %+v", records[0])
	}

	if records[1].Name != "second" || records[1].Value != 2 {
		t.Errorf("second record mismatch: %+v", records[1])
	}
}

func TestReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Write some records
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWriter(f)
	w.Write(testRecord{Name: "a", Value: 1})
	w.Write(testRecord{Name: "b", Value: 2})
	f.Close()

	// Read all
	records, err := ReadAll[testRecord](path)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestAppendOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "append.jsonl")

	// Append records one at a time
	if err := AppendOne(path, testRecord{Name: "x", Value: 10}); err != nil {
		t.Fatalf("AppendOne failed: %v", err)
	}
	if err := AppendOne(path, testRecord{Name: "y", Value: 20}); err != nil {
		t.Fatalf("AppendOne failed: %v", err)
	}

	// Verify
	records, err := ReadAll[testRecord](path)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Name != "x" || records[1].Name != "y" {
		t.Errorf("unexpected records: %+v", records)
	}
}

func TestReadAllNonExistent(t *testing.T) {
	records, err := ReadAll[testRecord]("/nonexistent/path.jsonl")
	if err != nil {
		t.Fatalf("ReadAll should return nil for non-existent file: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}
}
