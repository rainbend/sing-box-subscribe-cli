# sing-box-subscribe-cli

[![CI](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml)
[![Release](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml)

从订阅 URL 或本地订阅文件生成可直接使用的 [sing-box](https://sing-box.sagernet.org/) `config.json`。

[English](README.md) | 简体中文

`sing-box-subscribe-cli` 是一个纯 Go 命令行工具，适合本地脚本、服务器、CI 任务，以及任何不想运行 Web 服务、只想把订阅转换成 sing-box 配置的场景。

## 功能

- 从 HTTP/HTTPS URL 或本地文件读取订阅。
- 解析常见 Clash YAML 订阅中的 `proxies` 列表。
- 将 Clash `vmess`、`vless`、`trojan`、`ss`、`ssr`、`hysteria`、`hysteria2`、`tuic`、`wireguard`、`socks5`、`http` 和 `anytls` 节点转换为 sing-box outbounds。
- 复用订阅来源中已有的 sing-box `outbounds`。
- 将生成的节点合并到内置 sing-box JSON 模板。
- 展开 selector 和 urltest outbound 列表中的 `{all}`。
- 支持模板中的 selector/urltest `filter` 规则。
- 按协议或节点名称过滤节点。
- 给生成的节点 tag 添加前缀。
- 在不需要模板时，只输出生成的节点。

这个工具只生成配置文件，不负责运行 sing-box。

## 快速开始

安装 CLI，然后从订阅生成配置：

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

如果环境里有对应工具，可以校验生成结果：

```bash
jq empty config.json
sing-box check -c config.json
```

## 安装

### macOS

使用 Homebrew 安装：

```bash
brew install rainbend/tap/sing-box-subscribe-cli
```

检查安装结果：

```bash
sing-box-sub version
```

### Linux

安装最新版本：

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | bash
```

安装脚本支持 `linux/amd64` 和 `linux/arm64`。默认安装到 `/usr/local/bin/sing-box-sub`，必要时会请求 `sudo`。

安装到其他目录：

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | INSTALL_DIR="$HOME/.local/bin" bash
```

安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/rainbend/sing-box-subscribe-cli/main/install.sh | VERSION=v1.0.0 bash
```

如果希望在 sing-box 启动前自动重新生成配置，可以在 systemd unit 中使用 `ExecStartPre`：

```ini
[Unit]
Description=sing-box service
Documentation=https://sing-box.sagernet.org
After=network.target nss-lookup.target network-online.target

[Service]
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
ExecStartPre=/usr/local/bin/sing-box-sub --out /etc/sing-box/config.json 'https://example.com/api/v1/client/subscribe?token=REDACTED'
ExecStart=/usr/bin/sing-box -D /var/lib/sing-box -C /etc/sing-box run
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=10s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
```

如果 `ExecStartPre` 执行失败，systemd 会在启动 sing-box 前停止本次启动流程。请确认 `/usr/local/bin/sing-box-sub` 已存在，服务有权限写入 `/etc/sing-box/config.json`，并且服务启动时可以访问订阅 URL。

### Windows

从 [GitHub Releases 页面](https://github.com/rainbend/sing-box-subscribe-cli/releases) 下载匹配的 `.exe` 文件，可以重命名为 `sing-box-sub.exe`，并放到 `PATH` 包含的目录。

检查安装结果：

```powershell
sing-box-sub.exe version
```

### 容器

从 GitHub Packages 拉取镜像：

```bash
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:latest
docker pull ghcr.io/rainbend/sing-box-subscribe-cli:v1.0.0
```

把当前目录挂载为容器内的 `/work` 后运行：

```bash
docker run --rm \
  -v "$PWD:/work" \
  ghcr.io/rainbend/sing-box-subscribe-cli:latest \
  ./subscription.yaml --out config.json
```

镜像支持 `linux/amd64` 和 `linux/arm64`。

## 使用方式

从订阅 URL 生成配置：

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

也可以用 `--url` 传入订阅来源：

```bash
sing-box-sub \
  --url 'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

从本地订阅文件生成：

```bash
sing-box-sub ./subscription.yaml --out config.json
```

使用其他内置模板：

```bash
sing-box-sub ./subscription.yaml \
  --template config_template_groups_rule_set_tun.json \
  --out config.json
```

使用自己的模板文件：

```bash
sing-box-sub ./subscription.yaml \
  --template ./my-template.json \
  --out config.json
```

只输出生成的 outbounds：

```bash
sing-box-sub ./subscription.yaml --only-nodes --out nodes.json
```

过滤或重命名生成的节点：

```bash
sing-box-sub ./subscription.yaml \
  --prefix "Home - " \
  --exclude-protocol "ssr" \
  --exclude-node-name "expired|test" \
  --out config.json
```

## 模板

列出内置模板：

```bash
sing-box-sub list
```

当前内置模板：

- `config_template_groups_rule_set_tun.json`
- `config_template_groups_rule_set_tun_fakeip.json`
- `config_template_no_groups_tun_VN.json`
- `sb-config-1.12.json`
- `sb-config-1.14.json`

默认模板是 `sb-config-1.14.json`。

当 `--template` 是内置模板名时，会优先使用打包进二进制的模板。也支持 `./template.json`、`/path/to/template.json` 这类路径，以及 HTTP/HTTPS 模板 URL。

内置模板和规则组织参考了 [Toperlock/sing-box-subscribe](https://github.com/Toperlock/sing-box-subscribe)。

## 命令参考

```text
sing-box-sub [subscription URL or file] [flags]
sing-box-sub list
sing-box-sub version
```

常用参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--url` | 空 | 订阅 URL 或本地订阅文件。 |
| `--template` | `sb-config-1.14.json` | 模板名、模板路径或模板 URL。 |
| `--out` | `config.json` | 输出文件路径，使用 `-` 输出到 stdout。 |
| `--tag` | `tag_1` | 订阅组 tag。 |
| `--ua` | `clashmeta` | 请求订阅和模板时使用的 User-Agent。 |
| `--prefix` | 空 | 添加到生成 outbound tag 前面的前缀。 |
| `--exclude-protocol` | `ssr` | 要跳过的协议，多个值用逗号分隔。 |
| `--exclude-node-name` | 空 | 要跳过的节点 tag 子串，支持逗号或 `|` 分隔。 |
| `--only-nodes` | `false` | 只写出生成的 outbounds。 |
| `--timeout` | `60s` | HTTP 请求超时时间。 |

## 从源码构建

要求：

- Go 1.24 或更新版本
- `make`，可选

构建：

```bash
git clone https://github.com/rainbend/sing-box-subscribe-cli.git
cd sing-box-subscribe-cli
make build
```

二进制文件会输出到 `./bin/sing-box-sub`。

运行测试：

```bash
go test ./...
```

构建本地容器镜像：

```bash
docker build \
  --build-arg VERSION=dev \
  -t sing-box-subscribe-cli:dev .
```

## 发布

维护者通过推送 Git tag 发布版本：

```bash
git tag v1.0.0
git push origin v1.0.0
```

release workflow 会运行测试，构建 Linux、macOS 和 Windows 的 `amd64`、`arm64` 二进制文件，把 tag 注入 `sing-box-sub version`，并上传到 GitHub Releases。

容器镜像会发布到 GitHub Packages。版本 tag 会发布 `<version>` 和 `latest`；推送到 `main` 会发布 `main` 和 `sha-...` tag。

## 隐私

订阅 URL 通常包含私有 token。请不要在 issue、文档、测试或 fixture 中分享真实订阅 URL、token 或节点内容。复现问题时请使用脱敏示例或本地 fixture。

## License

本项目使用 [Apache License 2.0](LICENSE) 开源协议。
