package sanitize

import (
	"strings"
	"unicode/utf8"
)

// Recursive cleans lone surrogates and invalid UTF-8 sequences from
// nested map[string]interface{} / []interface{} / string structures.
//
// Lone surrogates (U+D800–U+DFFF) are invalid Unicode scalars that
// break UTF-8 encoding. They can appear in Windows paths or mis-decoded
// environment variables.
func Recursive(obj interface{}) interface{} {
	switch v := obj.(type) {
	case string:
		return cleanString(v)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, val := range v {
			out[cleanString(key)] = Recursive(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = Recursive(val)
		}
		return out
	default:
		return obj
	}
}

func cleanString(s string) string {
	if utf8.ValidString(s) && !hasSurrogate(s) {
		return s
	}
	// Use strings.ToValidUTF8 + manual surrogate removal
	return removeSurrogates(strings.ToValidUTF8(s, "\ufffd"))
}

func hasSurrogate(s string) bool {
	for _, r := range s {
		if r >= 0xD800 && r <= 0xDFFF {
			return true
		}
	}
	return false
}

func removeSurrogates(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0xD800 && r <= 0xDFFF {
			b.WriteRune('\ufffd') // replacement character
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
