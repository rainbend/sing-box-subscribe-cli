package subconv

import "testing"

func TestClashVMessAndHysteria2(t *testing.T) {
	input := []byte(`
proxies:
  - { name: Japan, type: vmess, server: example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000000, alterId: 0, cipher: auto, network: ws, ws-opts: { path: /ws, headers: { Host: edge.example.com } } }
  - { name: SG-HY2, type: hysteria2, server: hy.example.com, port: 1443, sni: hy.example.com, up: 100, down: 200, skip-cert-verify: true, password: secret }
`)
	nodes, warnings, err := ParseSubscription(input, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(nodes))
	}
	if nodes[0]["type"] != "vmess" || nodes[0]["tag"] != "Japan" {
		t.Fatalf("bad vmess node: %#v", nodes[0])
	}
	transport := nodes[0]["transport"].(map[string]any)
	headers := transport["headers"].(map[string]any)
	if headers["Host"] != "edge.example.com" {
		t.Fatalf("Host = %v", headers["Host"])
	}
	if nodes[1]["type"] != "hysteria2" || nodes[1]["up_mbps"] != 100 {
		t.Fatalf("bad hysteria2 node: %#v", nodes[1])
	}
}

func TestMergeTemplateExpandsAllWithFilter(t *testing.T) {
	template := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "Proxy",
				"type":      "selector",
				"outbounds": []any{"{all}"},
			},
			map[string]any{
				"tag":       "Japan",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": []any{
					map[string]any{"action": "include", "keywords": []any{"JP|Japan|日本"}},
				},
			},
			map[string]any{"tag": "direct", "type": "direct"},
		},
	}
	nodes := []outbound{
		{"tag": "Japan 1", "type": "vmess"},
		{"tag": "US 1", "type": "vmess"},
	}
	merged, err := MergeTemplate(template, map[string][]outbound{"tag_1": nodes})
	if err != nil {
		t.Fatal(err)
	}
	outbounds := merged["outbounds"].([]any)
	japan := outbounds[1].(map[string]any)
	choices := japan["outbounds"].([]any)
	if len(choices) != 1 || choices[0] != "Japan 1" {
		t.Fatalf("Japan choices = %#v", choices)
	}
	if _, exists := japan["filter"]; exists {
		t.Fatalf("filter should be removed after expansion")
	}
}
