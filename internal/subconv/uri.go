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

func parseURI(raw, proto string) (outbound, error) {
	switch proto {
	case "vmess":
		return parseVMessURI(raw)
	case "hysteria2":
		return parseHysteria2URI(raw)
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

func parseHysteria2URI(raw string) (outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if u.Path != "" {
		host += u.Path
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
	}
	server, portText, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(portText))
	if err != nil {
		return nil, err
	}
	query := u.Query()
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = randomName("hysteria2")
	}
	proxy := map[string]any{
		"name":             name,
		"type":             "hysteria2",
		"server":           strings.Trim(server, "[]"),
		"port":             port,
		"password":         firstNonEmpty(query.Get("auth"), user),
		"sni":              firstNonEmpty(query.Get("sni"), query.Get("peer")),
		"skip-cert-verify": truthy(query.Get("insecure")) || truthy(query.Get("allowInsecure")),
		"obfs":             query.Get("obfs"),
		"obfs-password":    query.Get("obfs-password"),
	}
	if up := numberText(query.Get("upmbps")); up != "" {
		proxy["up"] = up
	}
	if down := numberText(query.Get("downmbps")); down != "" {
		proxy["down"] = down
	}
	if alpn := query.Get("alpn"); alpn != "" {
		proxy["alpn"] = strings.Split(strings.Trim(alpn, "{}"), ",")
	}
	return clashHysteria2(proxy)
}

func decodeBase64String(raw string) (string, error) {
	unescaped, err := url.QueryUnescape(strings.TrimSpace(raw))
	if err == nil {
		raw = unescaped
	}
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
