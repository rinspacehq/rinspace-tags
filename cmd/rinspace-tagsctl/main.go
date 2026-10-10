package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rinspacehq/rinspace-tags/core/graph"
)

const maxSnapshotBytes = 16 << 20

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "usage: rinspace-tagsctl validate [-file snapshot.json]")
		return errors.New("unknown command")
	}
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	file := flags.String("file", "", "read the graph snapshot from this file instead of stdin")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	reader := stdin
	var closeFile *os.File
	if *file != "" {
		opened, err := os.Open(*file)
		if err != nil {
			return fmt.Errorf("open snapshot: %w", err)
		}
		closeFile = opened
		defer closeFile.Close()
		reader = opened
	}
	limited := io.LimitReader(reader, maxSnapshotBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	if len(data) > maxSnapshotBytes {
		return errors.New("snapshot exceeds 16 MiB")
	}
	var snapshot graph.Snapshot
	decoder := json.NewDecoder(newByteReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("snapshot must contain exactly one JSON value")
	}
	report := graph.Validate(snapshot)
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if !report.Valid {
		return errors.New("Tag relationship graph is invalid")
	}
	return nil
}

type byteReader struct {
	data []byte
	off  int
}

func newByteReader(data []byte) *byteReader { return &byteReader{data: data} }

func (reader *byteReader) Read(target []byte) (int, error) {
	if reader.off >= len(reader.data) {
		return 0, io.EOF
	}
	count := copy(target, reader.data[reader.off:])
	reader.off += count
	return count, nil
}
