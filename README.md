# sing-box-subscribe-cli

Pure Go CLI for generating a sing-box config from a subscription URL and a JSON template.

Build the CLI:

```bash
make build
```

```bash
./bin/sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=...' \
  --out config.json
```

The subscription source can also be passed with `--url`:

```bash
./bin/sing-box-sub \
  --url 'https://example.com/api/v1/client/subscribe?token=...' \
  --out config.json
```

The default template is `sb-config-1.14.json`; pass `--template` to use another template name, file path, or URL. Template names are resolved from bundled templates first, then from `~/.config/sing-box-subscribe/internal/templates`.

List available templates:

```bash
./bin/sing-box-sub list
```

The first implementation focuses on the real Clash YAML subscription flow used by this project and supports Clash `vmess` and `hysteria2` proxies, plus existing sing-box `outbounds` input. It expands template selectors that contain `{all}` and appends generated outbounds to the template.
