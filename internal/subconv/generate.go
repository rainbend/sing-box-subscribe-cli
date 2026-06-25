package subconv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func Generate(ctx context.Context, opts Options) (Result, error) {
	if opts.Tag == "" {
		opts.Tag = "tag_1"
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "clashmeta"
	}

	content, err := LoadBytes(ctx, opts.URL, opts.UserAgent, opts.Timeout)
	if err != nil {
		return Result{}, err
	}

	nodes, warnings, err := ParseSubscription(content, opts)
	if err != nil {
		return Result{}, err
	}
	if len(nodes) == 0 {
		return Result{}, fmt.Errorf("no supported nodes found in subscription")
	}

	applyPrefix(nodes, opts.Prefix)
	filterByName(&nodes, opts.ExcludeNodeName)
	deduplicateTags(nodes)

	if opts.OnlyNodes {
		return Result{Config: nodes, NodeCount: len(nodes), Warnings: warnings}, nil
	}

	templateBytes, err := LoadBytes(ctx, opts.Template, opts.UserAgent, opts.Timeout)
	if err != nil {
		return Result{}, err
	}

	var config map[string]any
	if err := json.Unmarshal(templateBytes, &config); err != nil {
		return Result{}, fmt.Errorf("parse template JSON %q: %w", opts.Template, err)
	}

	merged, err := MergeTemplate(config, map[string][]outbound{opts.Tag: nodes})
	if err != nil {
		return Result{}, err
	}
	return Result{Config: merged, NodeCount: len(nodes), Warnings: warnings}, nil
}

func applyPrefix(nodes []outbound, prefix string) {
	if prefix == "" {
		return
	}
	for _, node := range nodes {
		if tag, ok := node["tag"].(string); ok {
			node["tag"] = prefix + tag
		}
		if detour, ok := node["detour"].(string); ok {
			node["detour"] = prefix + detour
		}
	}
}

func filterByName(nodes *[]outbound, patterns string) {
	if strings.TrimSpace(patterns) == "" {
		return
	}
	parts := splitCSVOrPipe(patterns)
	filtered := (*nodes)[:0]
	for _, node := range *nodes {
		tag, _ := node["tag"].(string)
		skip := false
		for _, part := range parts {
			if part != "" && strings.Contains(tag, part) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, node)
		}
	}
	*nodes = filtered
}

func deduplicateTags(nodes []outbound) {
	seen := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		tag, _ := node["tag"].(string)
		if tag == "" {
			tag = randomName("node")
		}
		base := tag
		for index := 2; ; index++ {
			if _, ok := seen[tag]; !ok {
				break
			}
			tag = fmt.Sprintf("%s %d", base, index)
		}
		node["tag"] = tag
		seen[tag] = struct{}{}
	}
}
