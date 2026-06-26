package subconv

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestURIAdditionalProtocols(t *testing.T) {
	ssCredential := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:pass"))
	ssrPassword := base64.StdEncoding.EncodeToString([]byte("pass"))
	ssrRemark := base64.StdEncoding.EncodeToString([]byte("SSR-URI"))
	ssrPayload := base64.StdEncoding.EncodeToString([]byte("ssr.example.com:8388:origin:aes-128-gcm:plain:" + ssrPassword + "/?remarks=" + ssrRemark))
	httpAuthority := base64.StdEncoding.EncodeToString([]byte("user:pass@http.example.com:8080"))
	socksAuthority := base64.StdEncoding.EncodeToString([]byte("user:pass@socks.example.com:1080"))

	input := strings.Join([]string{
		"ss://" + ssCredential + "@ss.example.com:8388#SS-URI",
		"ssr://" + ssrPayload,
		"trojan://pass@trojan.example.com:443?sni=trojan.example.com#Trojan-URI",
		"vless://00000000-0000-0000-0000-000000000000@vless.example.com:443?security=tls&type=ws&host=edge.example.com&path=/ws#VLESS-URI",
		"hysteria://hy.example.com:443?auth=pass&peer=hy.example.com#HY-URI",
		"hysteria2://pass@hy2.example.com:443?sni=hy2.example.com#HY2-URI",
		"tuic://00000000-0000-0000-0000-000000000000:pass@tuic.example.com:443?sni=tuic.example.com#TUIC-URI",
		"wg://private@wg.example.com:51820?publicKey=public&ip=172.16.0.2#WG-URI",
		"http://" + httpAuthority + "#HTTP-URI",
		"socks://" + socksAuthority + "#SOCKS-URI",
		"anytls://pass@any.example.com:443?sni=any.example.com#AnyTLS-URI",
	}, "\n")

	nodes, warnings, err := ParseSubscription([]byte(input), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	wantTypes := map[string]int{
		"shadowsocks":  1,
		"shadowsocksr": 1,
		"trojan":       1,
		"vless":        1,
		"hysteria":     1,
		"hysteria2":    1,
		"tuic":         1,
		"wireguard":    1,
		"http":         1,
		"socks":        1,
		"anytls":       1,
	}
	gotTypes := typeCounts(nodes)
	for typ, want := range wantTypes {
		if gotTypes[typ] != want {
			t.Fatalf("type %s count = %d, want %d; nodes=%#v", typ, gotTypes[typ], want, nodes)
		}
	}
}
