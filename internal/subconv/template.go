package subconv

import (
	"fmt"
	"regexp"
	"strings"
)

func MergeTemplate(config map[string]any, groups map[string][]outbound) (map[string]any, error) {
	rawOutbounds, ok := config["outbounds"]
	if !ok {
		return nil, fmt.Errorf("template missing outbounds")
	}
	outboundsList, ok := asSlice(rawOutbounds)
	if !ok {
		return nil, fmt.Errorf("template outbounds is not a list")
	}

	directTag := "direct"
	for _, raw := range outboundsList {
		if item, ok := raw.(map[string]any); ok && stringValue(item, "type") == "direct" {
			if tag := stringValue(item, "tag"); tag != "" {
				directTag = tag
			}
			break
		}
	}

	for _, raw := range outboundsList {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rawChoices, exists := item["outbounds"]
		if !exists {
			continue
		}
		choices := stringList(rawChoices)
		if len(choices) == 0 {
			item["outbounds"] = []any{directTag}
			continue
		}
		expanded := expandTemplateChoices(choices, item, groups)
		if len(expanded) == 0 {
			expanded = append(expanded, directTag)
		}
		item["outbounds"] = toAnySlice(expanded)
		delete(item, "filter")
	}

	mergedOutbounds := make([]any, 0, len(outboundsList)+totalNodes(groups))
	mergedOutbounds = append(mergedOutbounds, outboundsList...)
	for _, groupNodes := range groups {
		for _, node := range groupNodes {
			mergedOutbounds = append(mergedOutbounds, node)
		}
	}
	config["outbounds"] = moveWireGuardToEndpoints(mergedOutbounds, config)
	return config, nil
}

func expandTemplateChoices(choices []string, selector outbound, groups map[string][]outbound) []string {
	result := make([]string, 0)
	seenChoice := make(map[string]struct{}, len(choices))
	for _, choice := range choices {
		if _, seen := seenChoice[choice]; seen {
			continue
		}
		seenChoice[choice] = struct{}{}
		if strings.HasPrefix(choice, "{") && strings.HasSuffix(choice, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(choice, "{"), "}")
			if nodes, ok := groups[name]; ok {
				result = append(result, tagsForSelector(nodes, selector, name)...)
				continue
			}
			if name == "all" {
				for group, nodes := range groups {
					result = append(result, tagsForSelector(nodes, selector, group)...)
				}
			}
			continue
		}
		result = append(result, choice)
	}
	return dedupeStrings(result)
}

func tagsForSelector(nodes []outbound, selector outbound, group string) []string {
	filtered := filterNodes(nodes, selector["filter"], group)
	tags := make([]string, 0, len(filtered))
	for _, node := range filtered {
		if tag := stringValue(node, "tag"); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func filterNodes(nodes []outbound, rawFilter any, group string) []outbound {
	filters, ok := asSlice(rawFilter)
	if !ok || len(filters) == 0 {
		return nodes
	}
	current := nodes
	for _, raw := range filters {
		filter, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if rawFor, exists := filter["for"]; exists && !stringListContains(rawFor, group) {
			continue
		}
		action := stringValue(filter, "action")
		keywords := stringList(filter["keywords"])
		current = actionKeywords(current, action, keywords)
	}
	return current
}

func actionKeywords(nodes []outbound, action string, keywords []string) []outbound {
	pattern := strings.Join(keywords, "|")
	if strings.TrimSpace(pattern) == "" {
		return nodes
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nodes
	}
	exclude := action == "exclude"
	filtered := make([]outbound, 0, len(nodes))
	for _, node := range nodes {
		tag := stringValue(node, "tag")
		match := re.MatchString(tag)
		if match != exclude {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func moveWireGuardToEndpoints(outbounds []any, config map[string]any) []any {
	var endpoints []any
	kept := make([]any, 0, len(outbounds))
	for _, raw := range outbounds {
		item, ok := raw.(map[string]any)
		if ok && stringValue(item, "type") == "wireguard" {
			endpoints = append(endpoints, item)
			continue
		}
		kept = append(kept, raw)
	}
	if len(endpoints) > 0 {
		config["endpoints"] = endpoints
	}
	return kept
}

func totalNodes(groups map[string][]outbound) int {
	total := 0
	for _, nodes := range groups {
		total += len(nodes)
	}
	return total
}
