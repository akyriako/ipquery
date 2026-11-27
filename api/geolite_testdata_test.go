package api

import (
	"bytes"
	"encoding/binary"
)

// This file builds a minimal but valid MMDB file in memory so the reload tests
// do not need a multi-megabyte binary fixture in the repository.
//
// Layout: a single search-tree node whose left record points into the data
// section and whose right record is "not found". Any IPv4 address whose first
// bit is 0 (e.g. 1.2.3.4) therefore resolves to the single record.

const mmdbMetadataMarker = "\xab\xcd\xefMaxMind.com"

// buildTestMMDB returns the bytes of an mmdb whose only record is
// {"test": value}.
func buildTestMMDB(value string) []byte {
	const nodeCount = 1

	var tree bytes.Buffer
	// record_size 32: two 4-byte records. Values >= nodeCount+16 are data
	// section offsets (value - (nodeCount + 16)); a value == nodeCount means
	// "not found".
	_ = binary.Write(&tree, binary.BigEndian, uint32(nodeCount+16)) // left -> data offset 0
	_ = binary.Write(&tree, binary.BigEndian, uint32(nodeCount))    // right -> not found

	data := encMap(map[string]any{"test": value})

	metadata := encMap(map[string]any{
		"binary_format_major_version": encUint16(2),
		"binary_format_minor_version": encUint16(0),
		"build_epoch":                 encUint64(1700000000),
		"database_type":               "Test",
		"description":                 encMap(map[string]any{"en": "test database"}),
		"ip_version":                  encUint16(4),
		"languages":                   encArray("en"),
		"node_count":                  encUint32(nodeCount),
		"record_size":                 encUint16(32),
	})

	var out bytes.Buffer
	out.Write(tree.Bytes())
	out.Write(make([]byte, 16)) // data section separator
	out.Write(data)
	out.WriteString(mmdbMetadataMarker)
	out.Write(metadata)
	return out.Bytes()
}

// raw wraps already-encoded bytes so they pass through enc unchanged.
type raw []byte

func enc(v any) []byte {
	switch t := v.(type) {
	case raw:
		return t
	case string:
		return append(ctrl(2, len(t)), t...)
	}
	panic("unsupported test value")
}

// ctrl builds a control byte (plus extended size bytes) for the given type and
// payload size. Only sizes < 29 are needed here.
func ctrl(typ, size int) []byte {
	if size >= 29 {
		panic("test encoder only supports small payloads")
	}
	if typ < 8 {
		return []byte{byte(typ<<5 | size)}
	}
	// Extended type: type field 0, followed by (type - 7).
	return []byte{byte(size), byte(typ - 7)}
}

func encMap(m map[string]any) raw {
	// Key order is irrelevant to the reader, but keep it deterministic.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)

	out := ctrl(7, len(m))
	for _, k := range keys {
		out = append(out, enc(k)...)
		out = append(out, enc(m[k])...)
	}
	return out
}

func encArray(items ...string) raw {
	out := ctrl(11, len(items))
	for _, it := range items {
		out = append(out, enc(it)...)
	}
	return out
}

func encUint16(v uint64) raw { return encUintType(5, v) }
func encUint32(v uint64) raw { return encUintType(6, v) }
func encUint64(v uint64) raw { return encUintType(9, v) }

func encUintType(typ int, v uint64) raw {
	var b []byte
	for i := 7; i >= 0; i-- {
		by := byte(v >> (8 * i))
		if len(b) == 0 && by == 0 {
			continue
		}
		b = append(b, by)
	}
	return append(ctrl(typ, len(b)), b...)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
