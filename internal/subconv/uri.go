package subconv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func parseURI(raw, proto string) ([]outbound, error) {
	switch proto {
	case "vmess":
		return oneOutbound(parseVMessURI(raw))
	case "ss":
		return parseShadowsocksURI(raw)
	case "ssr":
		return oneOutbound(parseShadowsocksRURI(raw))
	case "trojan":
		return oneOutbound(parseTrojanURI(raw))
	case "vless":
		return oneOutbound(parseVLessURI(raw))
	case "hysteria":
		return oneOutbound(parseHysteriaURI(raw))
	case "hysteria2":
		return oneOutbound(parseHysteria2URI(raw))
	case "tuic":
		return oneOutbound(parseTUICURI(raw))
	case "wg":
		return oneOutbound(parseWireGuardURI(raw))
	case "socks":
		return oneOutbound(parseSocksURI(raw))
	case "http", "https":
		return oneOutbound(parseHTTPProxyURI(raw, proto == "https"))
	case "anytls":
		return oneOutbound(parseAnyTLSURI(raw))
	default:
		return nil, fmt.Errorf("unsupported URI protocol %s", proto)
	}
}

func parseVMessURI(raw string) (outbound, error) {
	payload := strings.TrimPrefix(raw, "vmess://")
	decoded, err := decodeBase64String(payload)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(decoded), &item); err != nil {
		return nil, fmt.Errorf("parse vmess JSON: %w", err)
	}
	proxy := map[string]any{
		"name":    firstString(item, "ps"),
		"type":    "vmess",
		"server":  firstString(item, "add"),
		"port":    firstString(item, "port"),
		"uuid":    firstString(item, "id"),
		"alterId": firstString(item, "aid"),
		"cipher":  firstString(item, "scy"),
		"network": firstString(item, "net"),
	}
	if tls := firstString(item, "tls"); tls != "" && tls != "none" {
		proxy["tls"] = true
	}
	if host := firstString(item, "host"); host != "" {
		proxy["ws-headers"] = map[string]any{"Host": host}
	}
	if path := firstString(item, "path"); path != "" {
		proxy["ws-path"] = path
	}
	if sni := firstString(item, "sni"); sni != "" {
		proxy["sni"] = sni
	}
	if fp := firstString(item, "fp"); fp != "" {
		proxy["client-fingerprint"] = fp
	}
	if proxy["name"] == "" {
		proxy["name"] = randomName("vmess")
	}
	return clashVMess(proxy)
}

func parseShadowsocksURI(raw string) ([]outbound, error) {
	body, fragment := stripSchemeAndFragment(raw)
	body, query := splitBodyQuery(body)
	if amp := strings.Index(body, "&"); amp >= 0 {
		mergeQuery(query, body[amp+1:])
		body = body[:amp]
	}
	name := unescape(fragment)
	if name == "" {
		name = randomName("shadowsocks")
	}

	var method, password, server string
	var port int
	if strings.Contains(body, "@") {
		credential, hostPort, _ := strings.Cut(body, "@")
		decoded, ok := tryDecodeBase64String(credential)
		if !ok {
			decoded = unescape(credential)
		}
		var okCred bool
		method, password, okCred = splitCredential(decoded)
		if !okCred {
			return nil, fmt.Errorf("parse shadowsocks credentials")
		}
		var err error
		server, port, err = splitHostPortLoose(hostPort)
		if err != nil {
			return nil, err
		}
	} else {
		decoded, err := decodeBase64String(body)
		if err != nil {
			return nil, err
		}
		credential, hostPort, ok := strings.Cut(decoded, "@")
		if !ok {
			return nil, fmt.Errorf("parse shadowsocks authority")
		}
		method, password, ok = splitCredential(credential)
		if !ok {
			return nil, fmt.Errorf("parse shadowsocks credentials")
		}
		server, port, err = splitHostPortLoose(hostPort)
		if err != nil {
			return nil, err
		}
	}

	node := outbound{
		"tag":         name,
		"type":        "shadowsocks",
		"server":      server,
		"server_port": port,
		"method":      normalizeShadowsocksMethod(method),
		"password":    password,
	}
	if queryTruthy(query, "uot") || strings.Contains(raw, "uot") {
		node["udp_over_tcp"] = outbound{"enabled": true, "version": 2}
	}
	applyShadowsocksURIPlugin(node, query)
	if mux := multiplexFromQuery(query); mux != nil {
		node["multiplex"] = mux
	}
	if shadowTLS := query.Get("shadow-tls"); shadowTLS != "" {
		return shadowsocksShadowTLSOutbounds(node, shadowTLS, server, port)
	}
	return []outbound{node}, nil
}

func parseShadowsocksRURI(raw string) (outbound, error) {
	payload := strings.TrimPrefix(raw, "ssr://")
	decoded, err := decodeBase64String(payload)
	if err != nil {
		return nil, err
	}
	mainPart, rawQuery, _ := strings.Cut(decoded, "/?")
	parts := strings.SplitN(mainPart, ":", 6)
	if len(parts) != 6 {
		return nil, fmt.Errorf("parse shadowsocksr authority")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	password, err := decodeBase64String(parts[5])
	if err != nil {
		return nil, err
	}
	query, _ := url.ParseQuery(rawQuery)
	node := outbound{
		"tag":         randomName("shadowsocksr"),
		"type":        "shadowsocksr",
		"server":      parts[0],
		"server_port": port,
		"protocol":    parts[2],
		"method":      parts[3],
		"obfs":        parts[4],
		"password":    password,
	}
	if value := decodeQueryBase64(query, "obfsparam"); value != "" {
		node["obfs_param"] = value
	}
	if value := decodeQueryBase64(query, "protoparam"); value != "" {
		node["protocol_param"] = value
	}
	if value := decodeQueryBase64(query, "remarks"); value != "" {
		node["tag"] = value
	}
	return node, nil
}

func parseTrojanURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}
	if password == "" {
		return nil, fmt.Errorf("trojan requires password")
	}
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("trojan")
	}
	query := u.Query()
	node := outbound{
		"tag":         name,
		"type":        "trojan",
		"server":      server,
		"server_port": port,
		"password":    password,
		"tls":         tlsFromQuery(query, true, false),
	}
	applyURITransport(node, query.Get("type"), query)
	if mux := multiplexFromQuery(query); mux != nil {
		node["multiplex"] = mux
	}
	return node, nil
}

func parseVLessURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	uuid := ""
	if u.User != nil {
		uuid = u.User.Username()
	}
	if uuid == "" {
		return nil, fmt.Errorf("vless requires uuid")
	}
	query := u.Query()
	name := firstNonEmpty(query.Get("remarks"), unescape(u.Fragment), randomName("vless"))
	node := outbound{
		"tag":             name,
		"type":            "vless",
		"server":          server,
		"server_port":     port,
		"uuid":            uuid,
		"packet_encoding": firstNonEmpty(query.Get("packetEncoding"), "xudp"),
	}
	if flow := query.Get("flow"); flow != "" {
		node["flow"] = flow
	}
	security := strings.ToLower(query.Get("security"))
	if (security != "" && security != "none") || queryTruthy(query, "tls") {
		node["tls"] = tlsFromQuery(query, true, false)
		if security == "reality" || query.Get("pbk") != "" {
			tls := node["tls"].(outbound)
			reality := outbound{"enabled": true, "public_key": query.Get("pbk")}
			if sid := query.Get("sid"); sid != "" && sid != "None" {
				reality["short_id"] = sid
			}
			tls["reality"] = reality
			utls := outbound{"enabled": true}
			if fp := query.Get("fp"); fp != "" {
				utls["fingerprint"] = fp
			}
			tls["utls"] = utls
		}
	}
	if typ := query.Get("type"); typ != "" {
		applyURITransport(node, typ, query)
	} else if query.Get("obfs") == "websocket" {
		applyURITransport(node, "ws", query)
	}
	if mux := multiplexFromQuery(query); mux != nil {
		node["multiplex"] = mux
	}
	return node, nil
}

func parseHysteriaURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	query := u.Query()
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("hysteria")
	}
	node := outbound{
		"tag":         name,
		"type":        "hysteria",
		"server":      server,
		"server_port": port,
		"up_mbps":     intFromQueryDefault(query, "upmbps", 10),
		"down_mbps":   intFromQueryDefault(query, "downmbps", 100),
		"auth_str":    query.Get("auth"),
		"tls":         tlsFromQuery(query, true, false),
	}
	if alpn := splitList(query.Get("alpn")); len(alpn) > 0 {
		node["tls"].(outbound)["alpn"] = toAnySlice(alpn)
	}
	if obfs := query.Get("obfs"); obfs != "" && obfs != "none" {
		node["obfs"] = obfs
	}
	return node, nil
}

func parseHysteria2URI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := hostWithPath(u)
	server, port, err := splitHostPortLoose(host)
	if err != nil {
		return nil, err
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
	}
	query := u.Query()
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("hysteria2")
	}
	proxy := map[string]any{
		"name":             name,
		"type":             "hysteria2",
		"server":           server,
		"port":             port,
		"password":         firstNonEmpty(query.Get("auth"), user),
		"sni":              firstNonEmpty(query.Get("sni"), query.Get("peer")),
		"skip-cert-verify": queryTruthy(query, "insecure") || queryTruthy(query, "allowInsecure"),
		"obfs":             query.Get("obfs"),
		"obfs-password":    query.Get("obfs-password"),
	}
	if up := numberText(firstNonEmpty(query.Get("upmbps"), "10")); up != "" {
		proxy["up"] = up
	}
	if down := numberText(firstNonEmpty(query.Get("downmbps"), "100")); down != "" {
		proxy["down"] = down
	}
	if ports := portRange(host); ports != "" {
		proxy["ports"] = ports
	}
	if alpn := query.Get("alpn"); alpn != "" {
		proxy["alpn"] = splitList(alpn)
	}
	return clashHysteria2(proxy)
}

func parseTUICURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	uuid := ""
	password := ""
	if u.User != nil {
		uuid = u.User.Username()
		password, _ = u.User.Password()
	}
	query := u.Query()
	password = firstNonEmpty(password, query.Get("password"))
	if uuid == "" || password == "" {
		return nil, fmt.Errorf("tuic requires uuid and password")
	}
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("tuic")
	}
	node := outbound{
		"tag":                name,
		"type":               "tuic",
		"server":             server,
		"server_port":        port,
		"uuid":               uuid,
		"password":           password,
		"congestion_control": firstNonEmpty(query.Get("congestion_control"), "bbr"),
		"udp_relay_mode":     query.Get("udp_relay_mode"),
		"zero_rtt_handshake": false,
		"heartbeat":          "10s",
		"tls":                tlsFromQuery(query, true, false),
	}
	node["tls"].(outbound)["alpn"] = toAnySlice(splitList(firstNonEmpty(query.Get("alpn"), "h3")))
	return node, nil
}

func parseWireGuardURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	query := u.Query()
	privateKey := firstNonEmpty(queryString(query, "privateKey"), queryString(query, "privatekey"))
	if privateKey == "" && u.User != nil {
		privateKey = strings.ReplaceAll(u.User.Username(), " ", "+")
	}
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("wireguard")
	}
	proxy := map[string]any{
		"name":        name,
		"type":        "wireguard",
		"server":      server,
		"port":        port,
		"private-key": privateKey,
		"public-key":  firstNonEmpty(queryString(query, "publicKey"), queryString(query, "publickey")),
		"ip":          firstNonEmpty(queryString(query, "ip"), queryString(query, "address")),
		"reserved":    queryString(query, "reserved"),
	}
	if psk := queryString(query, "presharedKey"); psk != "" {
		proxy["pre-shared-key"] = psk
	}
	if mtu := query.Get("mtu"); mtu != "" {
		proxy["mtu"] = mtu
	}
	return clashWireGuard(proxy)
}

func parseSocksURI(raw string) (outbound, error) {
	authority, query, fragment := proxyAuthority(raw)
	server, port, username, password, err := parseProxyAuthority(authority)
	if err != nil {
		return nil, err
	}
	_ = query
	name := unescape(fragment)
	if name == "" {
		name = randomName("socks")
	}
	node := outbound{
		"tag":         name,
		"type":        "socks",
		"version":     "5",
		"server":      server,
		"server_port": port,
	}
	if username != "" {
		node["username"] = username
	}
	if password != "" {
		node["password"] = password
	}
	return node, nil
}

func parseHTTPProxyURI(raw string, forceTLS bool) (outbound, error) {
	authority, query, fragment := proxyAuthority(raw)
	server, port, username, password, err := parseProxyAuthority(authority)
	if err != nil {
		return nil, err
	}
	name := unescape(fragment)
	if name == "" {
		name = randomName("http")
	}
	node := outbound{
		"tag":         name,
		"type":        "http",
		"server":      server,
		"server_port": port,
	}
	if username != "" {
		node["username"] = username
	}
	if password != "" {
		node["password"] = password
	}
	if forceTLS || query.Get("sni") != "" {
		tls := outbound{"enabled": true, "insecure": true}
		if sni := query.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		node["tls"] = tls
	}
	return node, nil
}

func parseAnyTLSURI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	server, port, err := splitHostPortLoose(hostWithPath(u))
	if err != nil {
		return nil, err
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}
	query := u.Query()
	password = firstNonEmpty(query.Get("auth"), password)
	if password == "" {
		return nil, fmt.Errorf("anytls requires password/auth")
	}
	name := unescape(u.Fragment)
	if name == "" {
		name = randomName("anytls")
	}
	node := outbound{
		"tag":         name,
		"type":        "anytls",
		"server":      server,
		"server_port": port,
		"password":    password,
		"tls":         tlsFromQuery(query, true, false),
	}
	if value := query.Get("idleSessionCheckInterval"); value != "" {
		node["idle_session_check_interval"] = value + "s"
	}
	if value := query.Get("idleSessionTimeout"); value != "" {
		node["idle_session_timeout"] = value + "s"
	}
	if value := query.Get("minIdleSession"); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			node["min_idle_session"] = n
		}
	}
	return node, nil
}

func decodeBase64String(raw string) (string, error) {
	unescaped, err := url.PathUnescape(strings.TrimSpace(raw))
	if err == nil {
		raw = unescaped
	}
	raw = strings.ReplaceAll(raw, " ", "+")
	padding := len(raw) % 4
	if padding != 0 {
		raw += strings.Repeat("=", 4-padding)
	}
	data, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(raw)
	}
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return string(data), nil
}

func tryDecodeBase64String(raw string) (string, bool) {
	decoded, err := decodeBase64String(raw)
	return decoded, err == nil
}

func stripSchemeAndFragment(raw string) (string, string) {
	if index := strings.Index(raw, "://"); index >= 0 {
		raw = raw[index+3:]
	}
	body, fragment, _ := strings.Cut(raw, "#")
	return body, fragment
}

func splitBodyQuery(body string) (string, url.Values) {
	left, rawQuery, ok := strings.Cut(body, "?")
	values := url.Values{}
	if ok {
		values, _ = url.ParseQuery(rawQuery)
	}
	return left, values
}

func mergeQuery(values url.Values, raw string) {
	parsed, _ := url.ParseQuery(raw)
	for key, items := range parsed {
		for _, item := range items {
			values.Add(key, item)
		}
	}
}

func splitCredential(value string) (string, string, bool) {
	method, password, ok := strings.Cut(value, ":")
	return method, password, ok && method != "" && password != ""
}

func splitHostPortLoose(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "//")
	if slash := strings.Index(value, "/"); slash >= 0 {
		value = value[:slash]
	}
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[:comma]
	}
	if strings.HasPrefix(value, "[") {
		host, portText, err := net.SplitHostPort(value)
		if err == nil {
			port, err := numberPort(portText)
			return strings.Trim(host, "[]"), port, err
		}
	}
	index := strings.LastIndex(value, ":")
	if index < 0 {
		return "", 0, fmt.Errorf("missing port")
	}
	port, err := numberPort(value[index+1:])
	if err != nil {
		return "", 0, err
	}
	return strings.Trim(value[:index], "[]"), port, nil
}

func numberPort(value string) (int, error) {
	text := numberText(value)
	if text == "" {
		return 0, fmt.Errorf("missing port")
	}
	port, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func hostWithPath(u *url.URL) string {
	host := u.Host
	if u.Path != "" {
		host += u.Path
	}
	return host
}

func portRange(host string) string {
	if comma := strings.Index(host, ","); comma >= 0 {
		value := host[comma+1:]
		if slash := strings.Index(value, "/"); slash >= 0 {
			value = value[:slash]
		}
		return value
	}
	return ""
}

func queryTruthy(query url.Values, key string) bool {
	return truthy(query.Get(key))
}

func queryString(query url.Values, key string) string {
	return strings.ReplaceAll(query.Get(key), " ", "+")
}

func splitList(value string) []string {
	value = strings.TrimSpace(strings.Trim(value, "{}"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(unescape(part))
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func intFromQueryDefault(query url.Values, key string, fallback int) int {
	if text := numberText(query.Get(key)); text != "" {
		if value, err := strconv.Atoi(text); err == nil {
			return value
		}
	}
	return fallback
}

func decodeQueryBase64(query url.Values, key string) string {
	value := query.Get(key)
	if value == "" {
		return ""
	}
	decoded, err := decodeBase64String(value)
	if err != nil {
		return ""
	}
	return decoded
}

func tlsFromQuery(query url.Values, enabled bool, defaultInsecure bool) outbound {
	tls := outbound{"enabled": enabled, "insecure": defaultInsecure}
	if queryTruthy(query, "insecure") || queryTruthy(query, "allowInsecure") || queryTruthy(query, "allow_insecure") {
		tls["insecure"] = true
	}
	if sni := firstNonEmpty(query.Get("sni"), query.Get("peer"), query.Get("serverName")); sni != "" && sni != "None" {
		tls["server_name"] = sni
	}
	if alpn := splitList(query.Get("alpn")); len(alpn) > 0 {
		tls["alpn"] = toAnySlice(alpn)
	}
	if fp := query.Get("fp"); fp != "" {
		tls["utls"] = outbound{"enabled": true, "fingerprint": fp}
	}
	return tls
}

func applyURITransport(node outbound, typ string, query url.Values) {
	switch typ {
	case "h2", "http":
		transport := outbound{"type": "http"}
		if host := query.Get("host"); host != "" {
			transport["host"] = host
		}
		if path := query.Get("path"); path != "" {
			transport["path"] = path
		}
		node["transport"] = transport
	case "ws", "websocket":
		path := firstNonEmpty(query.Get("path"), "/")
		transport := outbound{"type": "ws", "path": cleanEarlyDataPath(path, "/")}
		if host := firstNonEmpty(query.Get("host"), query.Get("obfsParam")); host != "" && host != "None" {
			transport["headers"] = outbound{"Host": host}
			if tls, ok := node["tls"].(outbound); ok && stringValue(tls, "server_name") == "" {
				tls["server_name"] = host
			}
		}
		if matches := regexp.MustCompile(`\?ed=(\d+)$`).FindStringSubmatch(path); len(matches) == 2 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				transport["early_data_header_name"] = "Sec-WebSocket-Protocol"
				transport["max_early_data"] = n
			}
		}
		node["transport"] = transport
	case "grpc":
		node["transport"] = outbound{"type": "grpc", "service_name": query.Get("serviceName")}
	}
}

func multiplexFromQuery(query url.Values) outbound {
	protocol := query.Get("protocol")
	switch protocol {
	case "smux", "yamux", "h2mux":
	default:
		return nil
	}
	mux := outbound{"enabled": true, "protocol": protocol}
	if maxStreams := query.Get("max-streams"); maxStreams != "" {
		if value, err := strconv.Atoi(maxStreams); err == nil {
			mux["max_streams"] = value
		}
	} else {
		if maxConnections := query.Get("max-connections"); maxConnections != "" {
			if value, err := strconv.Atoi(maxConnections); err == nil {
				mux["max_connections"] = value
			}
		}
		if minStreams := query.Get("min-streams"); minStreams != "" {
			if value, err := strconv.Atoi(minStreams); err == nil {
				mux["min_streams"] = value
			}
		}
	}
	if query.Get("padding") == "True" || query.Get("padding") == "true" {
		mux["padding"] = true
	}
	return mux
}

func applyShadowsocksURIPlugin(node outbound, query url.Values) {
	plugin := query.Get("plugin")
	if plugin == "" {
		plugin = query.Get("v2ray-plugin")
	}
	if plugin == "" {
		return
	}
	if strings.HasPrefix(plugin, "obfs-local") || strings.HasPrefix(plugin, "simple-obfs") {
		fields := semicolonFields(plugin)
		node["plugin"] = "obfs-local"
		parts := []string{"obfs=" + fields["obfs"]}
		if host := fields["obfs-host"]; host != "" {
			parts = append(parts, "obfs-host="+host)
		}
		node["plugin_opts"] = strings.Join(parts, ";") + ";"
		return
	}
	if strings.Contains(plugin, "v2ray-plugin") {
		fields := semicolonFields(plugin)
		node["plugin"] = "v2ray-plugin"
		parts := []string{"mode=" + fields["mode"]}
		if host := fields["host"]; host != "" {
			parts = append(parts, "host="+host)
		}
		if path := fields["path"]; path != "" {
			parts = append(parts, "path="+path)
		}
		if fields["tls"] != "" {
			parts = append(parts, "tls")
		}
		node["plugin_opts"] = strings.Join(parts, ";") + ";"
	}
}

func semicolonFields(value string) map[string]string {
	result := map[string]string{}
	for _, part := range strings.Split(value, ";") {
		key, value, ok := strings.Cut(part, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

func shadowsocksShadowTLSOutbounds(node outbound, rawPlugin string, server string, port int) ([]outbound, error) {
	decoded, err := decodeBase64String(rawPlugin)
	if err != nil {
		return nil, err
	}
	var opts map[string]any
	if err := json.Unmarshal([]byte(decoded), &opts); err != nil {
		return nil, fmt.Errorf("parse shadow-tls plugin: %w", err)
	}
	detour := stringValue(node, "tag") + "_shadowtls"
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
	if fp := stringValue(opts, "fp"); fp != "" {
		shadowTLS["tls"].(outbound)["utls"] = outbound{"enabled": true, "fingerprint": fp}
	}
	return []outbound{node, shadowTLS}, nil
}

func proxyAuthority(raw string) (string, url.Values, string) {
	body, fragment := stripSchemeAndFragment(raw)
	body, query := splitBodyQuery(body)
	if amp := strings.Index(body, "&"); amp >= 0 {
		mergeQuery(query, body[amp+1:])
		body = body[:amp]
	}
	return body, query, fragment
}

func parseProxyAuthority(authority string) (server string, port int, username string, password string, err error) {
	if decoded, ok := tryDecodeBase64String(authority); ok && strings.Contains(decoded, ":") {
		authority = decoded
	}
	if strings.Contains(authority, "@") {
		credential, hostPort, _ := strings.Cut(authority, "@")
		username, password, _ = strings.Cut(credential, ":")
		server, port, err = splitHostPortLoose(hostPort)
		return server, port, username, password, err
	}
	server, port, err = splitHostPortLoose(authority)
	return server, port, "", "", err
}
