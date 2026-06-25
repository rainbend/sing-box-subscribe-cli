package subconv

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func clashProxyToOutbound(proxy map[string]any, excluded map[string]struct{}) (outbound, error) {
	typ := normalizedProtocol(stringValue(proxy, "type"))
	if typ == "" {
		return nil, fmt.Errorf("missing type")
	}
	if _, skip := excluded[typ]; skip {
		return nil, fmt.Errorf("protocol %s excluded", typ)
	}

	switch typ {
	case "vmess":
		return clashVMess(proxy)
	case "hysteria2":
		return clashHysteria2(proxy)
	default:
		return nil, fmt.Errorf("unsupported protocol %s", typ)
	}
}

func clashVMess(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	uuid := stringValue(proxy, "uuid")
	if name == "" || server == "" || !ok || uuid == "" {
		return nil, fmt.Errorf("vmess requires name, server, port, and uuid")
	}

	security := stringValue(proxy, "cipher")
	if security == "" || security == "http" || security == "gun" {
		security = "auto"
	}
	alterID, _ := intValue(proxy, "alterId")
	node := outbound{
		"tag":             name,
		"type":            "vmess",
		"server":          server,
		"server_port":     port,
		"uuid":            uuid,
		"security":        security,
		"alter_id":        alterID,
		"packet_encoding": "xudp",
	}

	if boolValue(proxy, "tls") {
		tls := outbound{
			"enabled":  true,
			"insecure": true,
		}
		if raw, exists := proxy["skip-cert-verify"]; exists && !toBool(raw) {
			tls["insecure"] = false
		}
		if sni := firstString(proxy, "servername", "sni"); sni != "" {
			tls["server_name"] = sni
		}
		if fp := stringValue(proxy, "client-fingerprint"); fp != "" {
			tls["utls"] = outbound{"enabled": true, "fingerprint": fp}
		}
		node["tls"] = tls
	}

	switch strings.ToLower(firstString(proxy, "network", "net")) {
	case "ws":
		transport := outbound{
			"type": "ws",
			"path": cleanEarlyDataPath(firstStringFromNested(
				proxy,
				[]string{"ws-opts", "path"},
				[]string{"ws-path"},
			), "/"),
		}
		host := firstStringFromNested(proxy,
			[]string{"ws-opts", "headers", "Host"},
			[]string{"ws-headers", "Host"},
			[]string{"host"},
		)
		if host != "" {
			transport["headers"] = outbound{"Host": host}
		}
		node["transport"] = transport
	case "grpc":
		service := firstStringFromNested(proxy, []string{"grpc-opts", "grpc-service-name"})
		if service == "/" {
			service = ""
		}
		node["transport"] = outbound{"type": "grpc", "service_name": service}
	case "h2", "http":
		transport := outbound{"type": "http"}
		if host := firstStringFromNested(proxy, []string{"h2-opts", "host"}, []string{"http-opts", "headers", "Host"}); host != "" {
			transport["host"] = host
		}
		if path := firstStringFromNested(proxy, []string{"h2-opts", "path"}, []string{"http-opts", "path"}); path != "" {
			transport["path"] = path
		}
		node["transport"] = transport
	case "quic":
		node["transport"] = outbound{"type": "quic"}
	}

	if smux, ok := mapValue(proxy, "smux"); ok && boolValue(smux, "enabled") {
		if mux := clashMultiplex(smux); mux != nil {
			node["multiplex"] = mux
		}
	}

	return node, nil
}

func clashHysteria2(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	password := firstString(proxy, "password", "auth")
	if name == "" || server == "" || !ok || password == "" {
		return nil, fmt.Errorf("hysteria2 requires name, server, port, and password/auth")
	}

	node := outbound{
		"tag":         name,
		"type":        "hysteria2",
		"server":      server,
		"server_port": port,
		"password":    password,
	}
	if up, ok := intValue(proxy, "up"); ok {
		node["up_mbps"] = up
	}
	if down, ok := intValue(proxy, "down"); ok {
		node["down_mbps"] = down
	}
	if ports := stringValue(proxy, "ports"); ports != "" {
		node["server_ports"] = []any{strings.ReplaceAll(ports, "-", ":")}
	}

	tls := outbound{"enabled": true, "insecure": true}
	if raw, exists := proxy["skip-cert-verify"]; exists && !toBool(raw) {
		tls["insecure"] = false
	}
	if sni := stringValue(proxy, "sni"); sni != "" && sni != "None" {
		tls["server_name"] = sni
	}
	tls["alpn"] = stringListOrDefault(proxy["alpn"], []any{"h3"})
	node["tls"] = tls

	if obfs := stringValue(proxy, "obfs"); obfs != "" && obfs != "none" {
		node["obfs"] = outbound{
			"type":     obfs,
			"password": stringValue(proxy, "obfs-password"),
		}
	}
	return node, nil
}

func clashMultiplex(smux map[string]any) outbound {
	protocol := stringValue(smux, "protocol")
	if protocol == "" {
		return nil
	}
	mux := outbound{"enabled": true, "protocol": protocol}
	if maxStreams, ok := intValue(smux, "max-streams"); ok {
		mux["max_streams"] = maxStreams
	} else {
		if maxConnections, ok := intValue(smux, "max-connections"); ok {
			mux["max_connections"] = maxConnections
		}
		if minStreams, ok := intValue(smux, "min-streams"); ok {
			mux["min_streams"] = minStreams
		}
	}
	if boolValue(smux, "padding") {
		mux["padding"] = true
	}
	return mux
}

var earlyDataPattern = regexp.MustCompile(`\?ed=\d+$`)

func cleanEarlyDataPath(path, fallback string) string {
	if path == "" {
		path = fallback
	}
	return earlyDataPattern.ReplaceAllString(path, "")
}

func parseQuery(raw string) url.Values {
	u, err := url.Parse(raw)
	if err != nil {
		return url.Values{}
	}
	return u.Query()
}
