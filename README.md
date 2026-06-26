# sing-box-subscribe-cli

[![CI](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml)
[![Release](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml)

Pure Go command-line tool for generating a final `config.json` for [sing-box](https://sing-box.sagernet.org/) from a subscription source and a JSON template.

English | [简体中文](README.zh-CN.md)

## What it does

`sing-box-subscribe-cli` is a focused CLI extracted from the original Python `sing-box-subscribe` workflow. It is designed for local automation, CI jobs, and small scripts that need to turn a subscription into a ready-to-use sing-box configuration.

It currently supports:

- HTTP/HTTPS subscription URLs and local subscription files.
- Clash YAML subscriptions with a `proxies` list.
- Clash `vmess` and `hysteria2` nodes.
- Existing sing-box `outbounds` input.
- Built-in sing-box JSON templates.
- `{all}` expansion in template selector and urltest outbound lists.
- selector/urltest `filter` rules, removed after expansion.
- Node filtering with `--exclude-protocol` and `--exclude-node-name`.
- Tag prefixing with `--prefix`.
- Node-only output with `--only-nodes`.
- WireGuard outbound migration to top-level `endpoints`, matching the original Python behavior.

This project is intentionally not a full web service. It only provides the CLI path needed to produce sing-box configuration files.

## Download

### Homebrew on macOS

Install the prebuilt macOS binary from the Homebrew tap:

```bash
brew install rainbend/tap/sing-box-subscribe-cli
```

The installed command is:

```bash
sing-box-sub version
```

### GitHub Releases

Prebuilt binaries are published on the [GitHub Releases page](https://github.com/rainbend/sing-box-subscribe-cli/releases).

Each tagged release includes binaries for:

| Platform | Architecture | Asset name pattern |
| --- | --- | --- |
| Linux | x86_64 | `sing-box-sub_<version>_linux_amd64` |
| Linux | arm64 | `sing-box-sub_<version>_linux_arm64` |
| macOS | Intel | `sing-box-sub_<version>_macos_amd64` |
| macOS | Apple Silicon | `sing-box-sub_<version>_macos_arm64` |
| Windows | x86_64 | `sing-box-sub_<version>_windows_amd64.exe` |
| Windows | arm64 | `sing-box-sub_<version>_windows_arm64.exe` |

On Linux or macOS, make the downloaded binary executable and move it into your `PATH`:

```bash
chmod +x sing-box-sub_v0.1.0_linux_amd64
sudo mv sing-box-sub_v0.1.0_linux_amd64 /usr/local/bin/sing-box-sub
```

On Windows, download the matching `.exe`, optionally rename it to `sing-box-sub.exe`, and place it in a directory listed in `PATH`.

Check the installed version:

```bash
sing-box-sub version
```

Release builds print the Git tag they were built from.

## Container image

Container images are published to GitHub Packages:

```bash
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:latest
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:v0.1.0
```

The image supports `linux/amd64` and `linux/arm64`.

Run the CLI in a container with the current directory mounted as `/work`:

```bash
docker run --rm \
  -v "$PWD:/work" \
  ghcr.io/rainbend/sing-box-subscribe-cli:latest \
  ./subscription.yaml --out config.json
```

Build a local image from source:

```bash
docker build \
  --build-arg VERSION=dev \
  -t sing-box-subscribe-cli:dev .
```

## Build from source

Requirements:

- Go 1.24 or newer.
- `make`, optional but recommended.

Clone and build:

```bash
git clone https://github.com/rainbend/sing-box-subscribe-cli.git
cd sing-box-subscribe-cli
make build
```

The binary is written to:

```bash
./bin/sing-box-sub
```

You can also build directly with Go:

```bash
go build -o ./bin/sing-box-sub ./cmd/sing-box-subscribe-cli
```

Run tests:

```bash
go test ./...
```

## Usage

Generate `config.json` from a subscription URL:

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

The subscription source can also be passed with `--url`:

```bash
sing-box-sub \
  --url 'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

Generate from a local subscription file:

```bash
sing-box-sub ./subscription.yaml --out config.json
```

Write only generated outbounds instead of merging a template:

```bash
sing-box-sub ./subscription.yaml --only-nodes --out nodes.json
```

Use a different built-in template:

```bash
sing-box-sub ./subscription.yaml \
  --template config_template_groups_rule_set_tun.json \
  --out config.json
```

Use a template file or URL:

```bash
sing-box-sub ./subscription.yaml \
  --template ./my-template.json \
  --out config.json
```

Filter and rename generated nodes:

```bash
sing-box-sub ./subscription.yaml \
  --prefix "Home - " \
  --exclude-protocol "ssr" \
  --exclude-node-name "expired|test" \
  --out config.json
```

Validate the generated file when the tools are available:

```bash
jq empty config.json
sing-box check -c config.json
```

## Built-in templates

List bundled templates:

```bash
sing-box-sub list
```

Current templates:

- `config_template_groups_rule_set_tun.json`
- `config_template_groups_rule_set_tun_fakeip.json`
- `config_template_no_groups_tun_VN.json`
- `sb-config-1.12.json`
- `sb-config-1.14.json`

The default template is:

```text
sb-config-1.14.json
```

When `--template` is set to a built-in template name, the bundled template is used first. Explicit paths such as `./template.json`, `/path/to/template.json`, and HTTP/HTTPS URLs are also supported.

## Command reference

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
| `--out` | `config.json` | Output config path, or `-` for stdout. |
| `--tag` | `tag_1` | Subscription group tag. |
| `--ua` | `clashmeta` | User-Agent for subscription and template HTTP requests. |
| `--prefix` | empty | Prefix added to generated outbound tags. |
| `--exclude-protocol` | `ssr` | Comma-separated protocols to skip. |
| `--exclude-node-name` | empty | Comma or pipe separated substrings to skip by node tag. |
| `--only-nodes` | `false` | Write only generated outbounds instead of merging a template. |
| `--timeout` | `60s` | HTTP request timeout. |

## Release process

Maintainers publish a release by pushing a Git tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs tests, cross-compiles Linux, macOS, and Windows binaries for `amd64` and `arm64`, injects the tag into `sing-box-sub version`, and uploads the binaries to GitHub Releases.

The container workflow builds and pushes multi-architecture images to GitHub Packages. Tags publish `<version>` and `latest`; pushes to `main` publish `main` and a `sha-...` tag.

Pull requests and pushes to `main` run the CI workflow, which tests the project and verifies the same target matrix can compile.

## Privacy and fixtures

Subscription URLs often contain private tokens. Do not commit real subscription URLs, tokens, or node details to issues, documentation, tests, or fixtures. Use redacted examples or local fixtures when reproducing behavior.

## License

Licensed under the [Apache License 2.0](LICENSE).
