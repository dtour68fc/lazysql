package utils

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const byteCellPrefix = "\x00lazysql-byte:"

func EncodeByteCell(value []byte) string {
	return byteCellPrefix + base64.StdEncoding.EncodeToString(value) + "\x00" + string(value)
}

func DisplayCell(value string, rawBytes bool) string {
	raw, text, ok := decodeByteCell(value)
	if !ok {
		return value
	}
	if !rawBytes {
		return text
	}

	parts := make([]string, len(raw))
	for i, b := range raw {
		parts[i] = fmt.Sprintf("%d", b)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func decodeByteCell(value string) ([]byte, string, bool) {
	if !strings.HasPrefix(value, byteCellPrefix) {
		return nil, "", false
	}

	rest := strings.TrimPrefix(value, byteCellPrefix)
	encoded, text, found := strings.Cut(rest, "\x00")
	if !found {
		return nil, "", false
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", false
	}

	return raw, text, true
}
