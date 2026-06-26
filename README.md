# sing-box-subscribe-cli

[![CI](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml)
[![Release](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml)

Generate a ready-to-use [sing-box](https://sing-box.sagernet.org/) `config.json` from a subscription URL or local subscription file.

English | [简体中文](README.zh-CN.md)

`sing-box-subscribe-cli` is a small, pure Go command-line tool. It is useful for local scripts, servers, CI jobs, and any workflow where you want to turn a subscription into a sing-box configuration without running a web service.

## Features

- Read subscriptions from HTTP/HTTPS URLs or local files.
- Parse common Clash YAML subscriptions with a `proxies` list.
- Convert Clash `vmess`, `vless`, `trojan`, `ss`, `ssr`, `hysteria`, `hysteria2`, `tuic`, `wireguard`, `socks5`, `http`, and `anytls` nodes to sing-box outbounds.
- Reuse existing sing-box `outbounds` from a subscription source.
- Merge generated nodes into bundled sing-box JSON templates.
- Expand `{all}` in selector and urltest outbound lists.
- Support selector/urltest `filter` rules in templates.
- Filter nodes by protocol or name.
- Add a prefix to generated node tags.
- Output only generated nodes when you do not want to merge a template.

This tool generates configuration files. It does not run sing-box for you.

## Quick Start

Install the CLI, then generate a config from a subscription:

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

Validate the result when the tools are available:

```bash
jq empty config.json
sing-box check -c config.json
```

## Install

### macOS

Install with Homebrew:

```bash
brew install rainbend/tap/sing-box-subscribe-cli
```

Check the installed command:

```bash
sing-box-sub version
```

### Linux

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | bash
```

The installer supports `linux/amd64` and `linux/arm64`. It installs `sing-box-sub` to `/usr/local/bin` by default and may ask for `sudo`.

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | INSTALL_DIR="$HOME/.local/bin" bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | VERSION=v1.0.0 bash
```

### Windows

Download the matching `.exe` from the [GitHub Releases page](https://github.com/rainbend/sing-box-subscribe-cli/releases), rename it to `sing-box-sub.exe` if you like, and place it in a directory listed in `PATH`.

Check the installed command:

```powershell
sing-box-sub.exe version
```

### Container

Pull the image from GitHub Packages:

```bash
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:latest
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:v1.0.0
```

Run the CLI with the current directory mounted as `/work`:

```bash
docker run --rm \
  -v "$PWD:/work" \
  ghcr.io/rainbend/sing-box-subscribe-cli:latest \
  ./subscription.yaml --out config.json
```

The image supports `linux/amd64` and `linux/arm64`.

## Usage

Generate from a subscription URL:

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

You can also pass the source with `--url`:

```bash
sing-box-sub \
  --url 'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

Generate from a local subscription file:

```bash
sing-box-sub ./subscription.yaml --out config.json
```

Use another bundled template:

```bash
sing-box-sub ./subscription.yaml \
  --template config_template_groups_rule_set_tun.json \
  --out config.json
```

Use your own template file:

```bash
sing-box-sub ./subscription.yaml \
  --template ./my-template.json \
  --out config.json
```

Write only generated outbounds:

```bash
sing-box-sub ./subscription.yaml --only-nodes --out nodes.json
```

Filter or rename generated nodes:

```bash
sing-box-sub ./subscription.yaml \
  --prefix "Home - " \
  --exclude-protocol "ssr" \
  --exclude-node-name "expired|test" \
  --out config.json
```

## Templates

List bundled templates:

```bash
sing-box-sub list
```

Bundled templates:

- `config_template_groups_rule_set_tun.json`
- `config_template_groups_rule_set_tun_fakeip.json`
- `config_template_no_groups_tun_VN.json`
- `sb-config-1.12.json`
- `sb-config-1.14.json`

The default template is `sb-config-1.14.json`.

When `--template` is a bundled template name, the bundled template is used first. Template paths such as `./template.json` and `/path/to/template.json` are also supported, as are HTTP/HTTPS template URLs.

The bundled templates and rule layout are inspired by [Toperlock/sing-box-subscribe](https://github.com/Toperlock/sing-box-subscribe).

## Command Reference

```text
sing-box-sub [subscription URL or file] [flags]
sing-box-sub list
sing-box-sub version
```

Common flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--url` | empty | Subscription URL or local subscription file. |
| `--template` | `sb-config-1.14.json` | Template name, template path, or template URL. |
| `--out` | `config.json` | Output file path. Use `-` for stdout. |
| `--tag` | `tag_1` | Subscription group tag. |
| `--ua` | `clashmeta` | User-Agent for subscription and template HTTP requests. |
| `--prefix` | empty | Prefix for generated outbound tags. |
| `--exclude-protocol` | `ssr` | Protocols to skip, separated by commas. |
| `--exclude-node-name` | empty | Node tag substrings to skip, separated by commas or `|`. |
| `--only-nodes` | `false` | Write generated outbounds only. |
| `--timeout` | `60s` | HTTP request timeout. |

## Build From Source

Requirements:

- Go 1.24 or newer
- `make`, optional

Build:

```bash
git clone https://github.com/rainbend/sing-box-subscribe-cli.git
cd sing-box-subscribe-cli
make build
```

The binary is written to `./bin/sing-box-sub`.

Run tests:

```bash
go test ./...
```

Build a local container image:

```bash
docker build \
  --build-arg VERSION=dev \
  -t sing-box-subscribe-cli:dev .
```

## Releases

Maintainers publish a release by pushing a Git tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release workflow runs tests, builds Linux, macOS, and Windows binaries for `amd64` and `arm64`, injects the tag into `sing-box-sub version`, and uploads the binaries to GitHub Releases.

Container images are published to GitHub Packages. Version tags publish `<version>` and `latest`; pushes to `main` publish `main` and a `sha-...` tag.

## Privacy

Subscription URLs often contain private tokens. Do not share real subscription URLs, tokens, or node details in issues, documentation, tests, or fixtures. Use redacted examples or local fixtures when reproducing behavior.

## License

Licensed under the [Apache License 2.0](LICENSE).
