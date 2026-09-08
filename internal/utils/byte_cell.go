package utils

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

const byteCellPrefix = "\x00lazysql-byte:"

func EncodeByteCell(value []byte) string {
	return byteCellPrefix + base64.StdEncoding.EncodeToString(value) + "\x00" + string(value)
}

// DisplayCell renders a byte column's value. When rawBytes is false (the
// default), it only shows the naive text cast if the underlying bytes are
// actually valid UTF-8 - genuinely binary data (protobuf/avro-framed
// payloads, etc) instead shows a plain "<binary, N bytes>" placeholder
// rather than the garbled mix of readable-and-garbage characters you get
// from printing arbitrary bytes as if they were text. rawBytes=true always
// shows the explicit numeric byte array regardless, for either case.
func DisplayCell(value string, rawBytes bool) string {
	raw, text, ok := decodeByteCell(value)
	if !ok {
		return value
	}
	if !rawBytes {
		if !utf8.Valid(raw) {
			return fmt.Sprintf("<binary, %d bytes>", len(raw))
		}
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
