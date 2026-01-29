package toon

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Writer provides TOON document serialization.
type Writer struct {
	w io.Writer
}

// NewWriter creates a TOON writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteDocument writes a TOON document.
func (w *Writer) WriteDocument(doc *Document) error {
	// Sort block names for consistent output
	names := make([]string, 0, len(doc.Blocks))
	for name := range doc.Blocks {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		if i > 0 {
			fmt.Fprintln(w.w)
		}
		if err := w.WriteBlock(doc.Blocks[name]); err != nil {
			return err
		}
	}
	return nil
}

// WriteBlock writes a single TOON block.
func (w *Writer) WriteBlock(b *Block) error {
	fmt.Fprintf(w.w, "[%s]\n", b.Name)

	// Write fields
	fieldKeys := make([]string, 0, len(b.Fields))
	for k := range b.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)

	for _, k := range fieldKeys {
		fmt.Fprintf(w.w, "%s: %s\n", k, b.Fields[k])
	}

	// Write arrays
	arrayKeys := make([]string, 0, len(b.Arrays))
	for k := range b.Arrays {
		arrayKeys = append(arrayKeys, k)
	}
	sort.Strings(arrayKeys)

	for _, k := range arrayKeys {
		arr := b.Arrays[k]
		if len(arr) == 0 {
			fmt.Fprintf(w.w, "%s[]: \n", k)
			continue
		}
		if len(arr) == 1 {
			fmt.Fprintf(w.w, "%s[]: %s\n", k, arr[0])
			continue
		}
		fmt.Fprintf(w.w, "%s[]:\n", k)
		for _, item := range arr {
			fmt.Fprintf(w.w, "  - %s\n", item)
		}
	}

	return nil
}

// Marshal converts a Document to a TOON string.
func Marshal(doc *Document) string {
	var sb strings.Builder
	w := NewWriter(&sb)
	w.WriteDocument(doc)
	return sb.String()
}

// MarshalBlock removed: zero callers (dead code audit 2026-01-29)
