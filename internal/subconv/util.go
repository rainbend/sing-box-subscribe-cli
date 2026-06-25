package subconv

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func normalize(v any) any {
	switch value := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(value))
		for key, child := range value {
			m[fmt.Sprint(key)] = normalize(child)
		}
		return m
	case map[string]any:
		m := make(map[string]any, len(value))
		for key, child := range value {
			m[key] = normalize(child)
		}
		return m
	case []any:
		items := make([]any, len(value))
		for i, child := range value {
			items[i] = normalize(child)
		}
		return items
	default:
		return value
	}
}

func asSlice(v any) ([]any, bool) {
	items, ok := v.([]any)
	return items, ok
}

func mapValue(m map[string]any, key string) (map[string]any, bool) {
	value, ok := m[key].(map[string]any)
	return value, ok
}

func stringValue(m map[string]any, key string) string {
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func intValue(m map[string]any, key string) (int, bool) {
	value, ok := m[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		return intFromString(v)
	default:
		return intFromString(fmt.Sprint(v))
	}
}

func intFromString(v string) (int, bool) {
	text := numberText(v)
	if text == "" {
		return 0, false
	}
	n, err := strconv.Atoi(text)
	return n, err == nil
}

func numberText(v string) string {
	return regexp.MustCompile(`\d+`).FindString(v)
}

func boolValue(m map[string]any, key string) bool {
	value, ok := m[key]
	return ok && toBool(value)
}

func toBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return truthy(v)
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(m, key); value != "" {
			return value
		}
	}
	return ""
}

func firstStringFromNested(m map[string]any, paths ...[]string) string {
	for _, path := range paths {
		var current any = m
		for _, key := range path {
			next, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = next[key]
		}
		if current == nil {
			continue
		}
		switch v := current.(type) {
		case string:
			if v != "" {
				return v
			}
		case []any:
			if len(v) > 0 {
				return fmt.Sprint(v[0])
			}
		default:
			if text := fmt.Sprint(v); text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringList(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		return append([]string(nil), v...)
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, fmt.Sprint(item))
		}
		return result
	default:
		return []string{fmt.Sprint(v)}
	}
}

func stringListOrDefault(value any, fallback []any) []any {
	items := stringList(value)
	if len(items) == 0 {
		return fallback
	}
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func stringListContains(value any, needle string) bool {
	for _, item := range stringList(value) {
		if item == needle {
			return true
		}
	}
	return false
}

func toAnySlice(items []string) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func splitCSVOrPipe(value string) []string {
	parts := regexp.MustCompile(`[,|]`).Split(value, -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func protocolSet(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, protocol := range splitCSVOrPipe(value) {
		if normalized := normalizedProtocol(protocol); normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func protocolOf(value string) string {
	index := strings.Index(value, "://")
	if index <= 0 {
		return ""
	}
	return value[:index]
}

func normalizedProtocol(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "hy2":
		return "hysteria2"
	case "wireguard":
		return "wg"
	case "http2":
		return "http"
	case "socks5":
		return "socks"
	default:
		return value
	}
}

func isTemplateOutbound(typ string) bool {
	switch typ {
	case "selector", "urltest", "direct", "block", "dns":
		return true
	default:
		return false
	}
}

func randomName(prefix string) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func unescape(value string) string {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
