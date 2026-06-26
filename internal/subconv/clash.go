package subconv

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func clashProxyToOutbounds(proxy map[string]any, excluded map[string]struct{}) ([]outbound, error) {
	typ := normalizedProtocol(stringValue(proxy, "type"))
	if typ == "" {
		return nil, fmt.Errorf("missing type")
	}
	if _, skip := excluded[typ]; skip {
		return nil, fmt.Errorf("protocol %s excluded", typ)
	}

	switch typ {
	case "vmess":
		return oneOutbound(clashVMess(proxy))
	case "ss":
		return clashShadowsocks(proxy)
	case "ssr":
		return oneOutbound(clashShadowsocksR(proxy))
	case "trojan":
		return oneOutbound(clashTrojan(proxy))
	case "vless":
		return oneOutbound(clashVLess(proxy))
	case "hysteria":
		return oneOutbound(clashHysteria(proxy))
	case "hysteria2":
		return oneOutbound(clashHysteria2(proxy))
	case "tuic":
		return oneOutbound(clashTUIC(proxy))
	case "wg":
		return oneOutbound(clashWireGuard(proxy))
	case "socks":
		return oneOutbound(clashSocks(proxy))
	case "http", "https":
		return oneOutbound(clashHTTP(proxy, typ == "https"))
	case "anytls":
		return oneOutbound(clashAnyTLS(proxy))
	default:
		return nil, fmt.Errorf("unsupported protocol %s", typ)
	}
}

func oneOutbound(node outbound, err error) ([]outbound, error) {
	if err != nil {
		return nil, err
	}
	return []outbound{node}, nil
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
		node["tls"] = clashTLS(proxy, true, true)
	}
	applyTransport(node, proxy, firstString(proxy, "network", "net"))
	applyMultiplex(node, proxy)
	return node, nil
}

func clashShadowsocks(proxy map[string]any) ([]outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	method := firstString(proxy, "cipher", "method")
	password := stringValue(proxy, "password")
	if name == "" || server == "" || !ok || method == "" || password == "" {
		return nil, fmt.Errorf("shadowsocks requires name, server, port, cipher/method, and password")
	}

	node := outbound{
		"tag":         name,
		"type":        "shadowsocks",
		"server":      server,
		"server_port": port,
		"method":      normalizeShadowsocksMethod(method),
		"password":    password,
	}
	if boolValue(proxy, "udp-over-tcp") || boolValue(proxy, "uot") {
		node["udp_over_tcp"] = outbound{"enabled": true, "version": 2}
	}
	applyShadowsocksPlugin(node, proxy)
	applyMultiplex(node, proxy)
	if stringValue(proxy, "plugin") != "shadow-tls" {
		return []outbound{node}, nil
	}

	opts, _ := mapValue(proxy, "plugin-opts")
	detour := name + "_shadowtls"
	node["detour"] = detour
	delete(node, "server")
	delete(node, "server_port")
	shadowTLS := outbound{
		"tag":         detour,
		"type":        "shadowtls",
		"server":      firstNonEmpty(stringValue(opts, "address"), server),
		"server_port": port,
		"version":     intValueOrDefault(opts, "version", 1),
		"password":    stringValue(opts, "password"),
		"tls": outbound{
			"enabled":     true,
			"server_name": stringValue(opts, "host"),
		},
	}
	if overridePort, ok := intValue(opts, "port"); ok {
		shadowTLS["server_port"] = overridePort
	}
	if fp := firstString(proxy, "client-fingerprint"); fp != "" {
		shadowTLS["tls"].(outbound)["utls"] = outbound{"enabled": true, "fingerprint": fp}
	}
	return []outbound{node, shadowTLS}, nil
}

func clashShadowsocksR(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	method := firstString(proxy, "cipher", "method")
	password := stringValue(proxy, "password")
	protocol := stringValue(proxy, "protocol")
	obfs := stringValue(proxy, "obfs")
	if name == "" || server == "" || !ok || method == "" || password == "" || protocol == "" || obfs == "" {
		return nil, fmt.Errorf("shadowsocksr requires name, server, port, protocol, cipher/method, obfs, and password")
	}
	node := outbound{
		"tag":         name,
		"type":        "shadowsocksr",
		"server":      server,
		"server_port": port,
		"protocol":    protocol,
		"method":      method,
		"obfs":        obfs,
		"password":    password,
	}
	if value := firstString(proxy, "obfs-param", "obfs_param"); value != "" {
		node["obfs_param"] = value
	}
	if value := firstString(proxy, "protocol-param", "protocol_param"); value != "" {
		node["protocol_param"] = value
	}
	return node, nil
}

func clashTrojan(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	password := stringValue(proxy, "password")
	if name == "" || server == "" || !ok || password == "" {
		return nil, fmt.Errorf("trojan requires name, server, port, and password")
	}
	node := outbound{
		"tag":         name,
		"type":        "trojan",
		"server":      server,
		"server_port": port,
		"password":    password,
		"tls":         clashTLS(proxy, true, false),
	}
	applyTransport(node, proxy, firstString(proxy, "network", "net"))
	applyMultiplex(node, proxy)
	return node, nil
}

func clashVLess(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	uuid := stringValue(proxy, "uuid")
	if name == "" || server == "" || !ok || uuid == "" {
		return nil, fmt.Errorf("vless requires name, server, port, and uuid")
	}
	node := outbound{
		"tag":             name,
		"type":            "vless",
		"server":          server,
		"server_port":     port,
		"uuid":            uuid,
		"packet_encoding": firstNonEmpty(stringValue(proxy, "packet-encoding"), stringValue(proxy, "packetEncoding"), "xudp"),
	}
	if flow := stringValue(proxy, "flow"); flow != "" {
		node["flow"] = flow
	}
	if shouldEnableVLessTLS(proxy) {
		node["tls"] = clashTLS(proxy, true, false)
	}
	applyTransport(node, proxy, firstString(proxy, "network", "net"))
	applyMultiplex(node, proxy)
	return node, nil
}

func clashHysteria(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	if name == "" || server == "" || !ok {
		return nil, fmt.Errorf("hysteria requires name, server, and port")
	}
	node := outbound{
		"tag":         name,
		"type":        "hysteria",
		"server":      server,
		"server_port": port,
		"tls":         clashTLS(proxy, true, false),
	}
	if auth := firstString(proxy, "auth_str", "auth-str", "auth"); auth != "" {
		node["auth_str"] = auth
	}
	if up, ok := intValue(proxy, "up"); ok {
		node["up_mbps"] = up
	}
	if down, ok := intValue(proxy, "down"); ok {
		node["down_mbps"] = down
	}
	if obfs := stringValue(proxy, "obfs"); obfs != "" && obfs != "none" {
		node["obfs"] = obfs
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

	tls := clashTLS(proxy, true, false)
	if stringValue(tls, "server_name") == "" {
		delete(tls, "server_name")
		tls["insecure"] = true
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

func clashTUIC(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	uuid := stringValue(proxy, "uuid")
	password := stringValue(proxy, "password")
	if name == "" || server == "" || !ok || uuid == "" || password == "" {
		return nil, fmt.Errorf("tuic requires name, server, port, uuid, and password")
	}
	node := outbound{
		"tag":                name,
		"type":               "tuic",
		"server":             server,
		"server_port":        port,
		"uuid":               uuid,
		"password":           password,
		"congestion_control": firstNonEmpty(stringValue(proxy, "congestion-controller"), stringValue(proxy, "congestion_control"), "bbr"),
		"udp_relay_mode":     firstNonEmpty(stringValue(proxy, "udp-relay-mode"), stringValue(proxy, "udp_relay_mode"), "native"),
		"zero_rtt_handshake": false,
		"heartbeat":          "10s",
		"tls":                clashTLS(proxy, true, false),
	}
	node["tls"].(outbound)["alpn"] = stringListOrDefault(proxy["alpn"], []any{"h3"})
	return node, nil
}

func clashWireGuard(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	privateKey := firstString(proxy, "private-key", "privateKey")
	publicKey := firstString(proxy, "public-key", "publicKey")
	address := wireGuardAddressList(firstString(proxy, "ip", "address"), stringValue(proxy, "ipv6"))
	if name == "" || server == "" || !ok || privateKey == "" || publicKey == "" || len(address) == 0 {
		return nil, fmt.Errorf("wireguard requires name, server, port, private-key, public-key, and ip/address")
	}
	peer := outbound{
		"address":                       server,
		"port":                          port,
		"public_key":                    publicKey,
		"allowed_ips":                   []any{"0.0.0.0/0"},
		"persistent_keepalive_interval": 30,
	}
	if psk := firstString(proxy, "pre-shared-key", "presharedKey", "pre_shared_key"); psk != "" {
		peer["pre_shared_key"] = psk
	}
	if reserved, exists := wireGuardReserved(proxy["reserved"]); exists {
		peer["reserved"] = reserved
	}
	node := outbound{
		"tag":         name,
		"type":        "wireguard",
		"private_key": privateKey,
		"address":     address,
		"peers":       []any{peer},
	}
	if mtu, ok := intValue(proxy, "mtu"); ok {
		node["mtu"] = mtu
	}
	return node, nil
}

func clashSocks(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	if name == "" || server == "" || !ok {
		return nil, fmt.Errorf("socks requires name, server, and port")
	}
	node := outbound{
		"tag":         name,
		"type":        "socks",
		"version":     "5",
		"server":      server,
		"server_port": port,
	}
	if username := stringValue(proxy, "username"); username != "" {
		node["username"] = username
	}
	if password := stringValue(proxy, "password"); password != "" {
		node["password"] = password
	}
	return node, nil
}

func clashHTTP(proxy map[string]any, forceTLS bool) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	if name == "" || server == "" || !ok {
		return nil, fmt.Errorf("http requires name, server, and port")
	}
	node := outbound{
		"tag":         name,
		"type":        "http",
		"server":      server,
		"server_port": port,
	}
	if username := stringValue(proxy, "username"); username != "" {
		node["username"] = username
	}
	if password := stringValue(proxy, "password"); password != "" {
		node["password"] = password
	}
	if forceTLS || boolValue(proxy, "tls") || firstString(proxy, "sni", "servername") != "" {
		node["tls"] = clashTLS(proxy, true, true)
	}
	return node, nil
}

func clashAnyTLS(proxy map[string]any) (outbound, error) {
	name := stringValue(proxy, "name")
	server := stringValue(proxy, "server")
	port, ok := intValue(proxy, "port")
	password := firstString(proxy, "password", "auth")
	if name == "" || server == "" || !ok || password == "" {
		return nil, fmt.Errorf("anytls requires name, server, port, and password/auth")
	}
	node := outbound{
		"tag":         name,
		"type":        "anytls",
		"server":      server,
		"server_port": port,
		"password":    password,
		"tls":         clashTLS(proxy, true, false),
	}
	if value := durationSeconds(proxy, "idle-session-check-interval"); value != "" {
		node["idle_session_check_interval"] = value
	}
	if value := durationSeconds(proxy, "idle-session-timeout"); value != "" {
		node["idle_session_timeout"] = value
	}
	if value, ok := intValue(proxy, "min-idle-session"); ok {
		node["min_idle_session"] = value
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

func applyMultiplex(node outbound, proxy map[string]any) {
	if smux, ok := mapValue(proxy, "smux"); ok && boolValue(smux, "enabled") {
		if mux := clashMultiplex(smux); mux != nil {
			node["multiplex"] = mux
		}
	}
}

func applyTransport(node outbound, proxy map[string]any, network string) {
	switch strings.ToLower(network) {
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
}

func clashTLS(proxy map[string]any, enabled bool, defaultInsecure bool) outbound {
	tls := outbound{"enabled": enabled, "insecure": defaultInsecure}
	if raw, exists := proxy["skip-cert-verify"]; exists {
		tls["insecure"] = toBool(raw)
	}
	if raw, exists := proxy["allowInsecure"]; exists {
		tls["insecure"] = toBool(raw)
	}
	if sni := firstString(proxy, "servername", "sni", "peer"); sni != "" && sni != "None" {
		tls["server_name"] = sni
	}
	if fp := stringValue(proxy, "client-fingerprint"); fp != "" {
		tls["utls"] = outbound{"enabled": true, "fingerprint": fp}
	}
	if alpn := stringList(proxy["alpn"]); len(alpn) > 0 {
		tls["alpn"] = toAnySlice(alpn)
	}
	if reality, ok := mapValue(proxy, "reality-opts"); ok {
		if publicKey := firstString(reality, "public-key", "public_key"); publicKey != "" {
			realityConfig := outbound{"enabled": true, "public_key": publicKey}
			if shortID := firstString(reality, "short-id", "short_id"); shortID != "" && shortID != "None" {
				realityConfig["short_id"] = shortID
			}
			tls["reality"] = realityConfig
			utls := outbound{"enabled": true}
			if fp := stringValue(proxy, "client-fingerprint"); fp != "" {
				utls["fingerprint"] = fp
			}
			tls["utls"] = utls
		}
	}
	return tls
}

func shouldEnableVLessTLS(proxy map[string]any) bool {
	if raw, exists := proxy["tls"]; exists && !toBool(raw) {
		return false
	}
	if security := strings.ToLower(stringValue(proxy, "security")); security == "none" {
		return false
	}
	return true
}

func applyShadowsocksPlugin(node outbound, proxy map[string]any) {
	plugin := stringValue(proxy, "plugin")
	if plugin == "" || plugin == "shadow-tls" {
		return
	}
	opts, _ := mapValue(proxy, "plugin-opts")
	switch plugin {
	case "obfs", "obfs-local", "simple-obfs":
		node["plugin"] = "obfs-local"
		parts := []string{"obfs=" + firstString(opts, "mode", "obfs")}
		if host := firstString(opts, "host", "obfs-host"); host != "" {
			parts = append(parts, "obfs-host="+host)
		}
		node["plugin_opts"] = strings.Join(parts, ";") + ";"
	case "v2ray-plugin":
		node["plugin"] = "v2ray-plugin"
		parts := []string{"mode=" + firstString(opts, "mode", "obfs")}
		if host := firstString(opts, "host", "obfs-host"); host != "" {
			parts = append(parts, "host="+host)
		}
		if path := stringValue(opts, "path"); path != "" {
			parts = append(parts, "path="+path)
		}
		if boolValue(opts, "mux") {
			parts = append(parts, "mux=true")
		}
		if boolValue(opts, "tls") {
			parts = append(parts, "tls")
		}
		node["plugin_opts"] = strings.Join(parts, ";") + ";"
	default:
		node["plugin"] = plugin
	}
}

func normalizeShadowsocksMethod(method string) string {
	switch method {
	case "chacha20-poly1305":
		return "chacha20-ietf-poly1305"
	case "xchacha20-poly1305":
		return "xchacha20-ietf-poly1305"
	default:
		return method
	}
}

func intValueOrDefault(m map[string]any, key string, fallback int) int {
	if value, ok := intValue(m, key); ok {
		return value
	}
	return fallback
}

func wireGuardAddressList(ipValue, ipv6Value string) []any {
	var values []string
	if ipValue != "" {
		values = append(values, strings.Split(ipValue, ",")...)
	}
	if ipv6Value != "" {
		values = append(values, ipv6Value)
	}
	result := make([]any, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.Contains(value, "/") {
			if strings.Contains(value, ":") {
				value += "/128"
			} else {
				value += "/32"
			}
		}
		result = append(result, value)
	}
	return result
}

func wireGuardReserved(value any) (any, bool) {
	switch v := value.(type) {
	case nil:
		return nil, false
	case []any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			if n, err := strconv.Atoi(fmt.Sprint(item)); err == nil {
				items = append(items, n)
			}
		}
		return items, len(items) > 0
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, false
		}
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			items := make([]any, 0, len(parts))
			for _, part := range parts {
				if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
					items = append(items, n)
				}
			}
			return items, len(items) > 0
		}
		return v, true
	default:
		return v, true
	}
}

func durationSeconds(m map[string]any, key string) string {
	value := stringValue(m, key)
	if value == "" {
		return ""
	}
	if _, err := strconv.Atoi(value); err == nil {
		return value + "s"
	}
	return value
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
