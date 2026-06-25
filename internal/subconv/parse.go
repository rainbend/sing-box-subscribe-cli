package subconv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseSubscription(content []byte, opts Options) ([]outbound, []string, error) {
	text := strings.TrimPrefix(string(content), "\ufeff")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil, fmt.Errorf("subscription is empty")
	}

	excluded := protocolSet(opts.ExcludeProtocol)

	var jsonDoc any
	if json.Unmarshal([]byte(text), &jsonDoc) == nil {
		return nodesFromDocument(normalize(jsonDoc), excluded)
	}

	var yamlDoc any
	if yaml.Unmarshal([]byte(strings.ReplaceAll(text, "\t", " ")), &yamlDoc) == nil {
		normalized := normalize(yamlDoc)
		if root, ok := normalized.(map[string]any); ok {
			if _, hasProxies := root["proxies"]; hasProxies {
				return nodesFromDocument(root, excluded)
			}
			if _, hasOutbounds := root["outbounds"]; hasOutbounds {
				return nodesFromDocument(root, excluded)
			}
		}
	}

	if decoded, ok := maybeBase64(text); ok {
		return parseLines(decoded, excluded)
	}
	return parseLines(text, excluded)
}

func nodesFromDocument(doc any, excluded map[string]struct{}) ([]outbound, []string, error) {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("subscription document is not an object")
	}
	if rawProxies, ok := root["proxies"]; ok {
		proxies, ok := asSlice(rawProxies)
		if !ok {
			return nil, nil, fmt.Errorf("subscription proxies is not a list")
		}
		nodes := make([]outbound, 0, len(proxies))
		var warnings []string
		for _, raw := range proxies {
			proxy, ok := raw.(map[string]any)
			if !ok {
				warnings = append(warnings, "skip non-object proxy")
				continue
			}
			node, err := clashProxyToOutbound(proxy, excluded)
			if err != nil {
				name := stringValue(proxy, "name")
				if name == "" {
					name = "<unnamed>"
				}
				warnings = append(warnings, fmt.Sprintf("skip %s: %v", name, err))
				continue
			}
			nodes = append(nodes, node)
		}
		return nodes, warnings, nil
	}
	if rawOutbounds, ok := root["outbounds"]; ok {
		items, ok := asSlice(rawOutbounds)
		if !ok {
			return nil, nil, fmt.Errorf("outbounds is not a list")
		}
		nodes := make([]outbound, 0, len(items))
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			typ := normalizedProtocol(stringValue(item, "type"))
			if isTemplateOutbound(typ) {
				continue
			}
			if _, skip := excluded[typ]; skip {
				continue
			}
			nodes = append(nodes, item)
		}
		return nodes, nil, nil
	}
	return nil, nil, fmt.Errorf("document does not contain proxies or outbounds")
}

func parseLines(text string, excluded map[string]struct{}) ([]outbound, []string, error) {
	var nodes []outbound
	var warnings []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		proto := normalizedProtocol(protocolOf(line))
		if proto == "" {
			continue
		}
		if _, skip := excluded[proto]; skip {
			continue
		}
		node, err := parseURI(line, proto)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skip %s URI: %v", proto, err))
			continue
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 && len(warnings) == 0 {
		return nil, nil, fmt.Errorf("no recognizable subscription lines found")
	}
	return nodes, warnings, nil
}

func maybeBase64(text string) (string, bool) {
	candidate := strings.TrimSpace(text)
	candidate = strings.ReplaceAll(candidate, "\n", "")
	candidate = strings.ReplaceAll(candidate, "\r", "")
	decodedURL, err := url.QueryUnescape(candidate)
	if err == nil {
		candidate = decodedURL
	}
	padding := len(candidate) % 4
	if padding != 0 {
		candidate += strings.Repeat("=", 4-padding)
	}
	data, err := base64.URLEncoding.DecodeString(candidate)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(candidate)
	}
	if err != nil {
		return "", false
	}
	decoded := strings.TrimSpace(string(data))
	return decoded, decoded != ""
}
