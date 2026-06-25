# sing-box-subscribe-cli

[![CI](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/ci.yml)
[![Release](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml/badge.svg)](https://github.com/rainbend/sing-box-subscribe-cli/actions/workflows/release.yml)

用于从订阅来源和 JSON 模板生成 [sing-box](https://sing-box.sagernet.org/) 最终 `config.json` 的纯 Go 命令行工具。

[English](README.md) | 简体中文

## 它做什么

`sing-box-subscribe-cli` 是从原 Python `sing-box-subscribe` 工作流中拆出的专注型 CLI。它适合本地自动化、CI 任务和脚本场景，用来把订阅转换成可直接使用的 sing-box 配置文件。

当前支持：

- HTTP/HTTPS 订阅 URL 和本地订阅文件。
- 带有 `proxies` 列表的 Clash YAML 订阅。
- Clash `vmess` 和 `hysteria2` 节点。
- 已有 sing-box `outbounds` 输入。
- 内置 sing-box JSON 模板。
- 模板 selector 和 urltest outbound 列表中的 `{all}` 展开。
- selector/urltest 的 `filter` 规则，展开后会移除 `filter` 字段。
- 使用 `--exclude-protocol` 和 `--exclude-node-name` 过滤节点。
- 使用 `--prefix` 给节点 tag 加前缀。
- 使用 `--only-nodes` 只输出节点。
- 按原 Python 行为将 WireGuard outbound 迁移到顶层 `endpoints`。

这个项目不是完整 Web 服务，只提供生成 sing-box 配置所需的 CLI 路径。

## 下载

预编译二进制文件发布在 [GitHub Releases 页面](https://github.com/rainbend/sing-box-subscribe-cli/releases)。

每个 tag release 会包含以下平台和架构：

| 平台 | 架构 | 文件名格式 |
| --- | --- | --- |
| Linux | x86_64 | `sing-box-sub_<version>_linux_amd64` |
| Linux | arm64 | `sing-box-sub_<version>_linux_arm64` |
| macOS | Intel | `sing-box-sub_<version>_macos_amd64` |
| macOS | Apple Silicon | `sing-box-sub_<version>_macos_arm64` |
| Windows | x86_64 | `sing-box-sub_<version>_windows_amd64.exe` |
| Windows | arm64 | `sing-box-sub_<version>_windows_arm64.exe` |

在 Linux 或 macOS 上，下载后赋予执行权限并移动到 `PATH` 中：

```bash
chmod +x sing-box-sub_v0.1.0_linux_amd64
sudo mv sing-box-sub_v0.1.0_linux_amd64 /usr/local/bin/sing-box-sub
```

在 Windows 上，下载匹配的 `.exe` 文件，可以重命名为 `sing-box-sub.exe`，并放到 `PATH` 包含的目录。

检查当前版本：

```bash
sing-box-sub version
```

release 构建会输出它对应的 Git tag。

## 从源码构建

要求：

- Go 1.24 或更新版本。
- `make`，可选但推荐。

克隆并构建：

```bash
git clone https://github.com/rainbend/sing-box-subscribe-cli.git
cd sing-box-subscribe-cli
make build
```

二进制文件会输出到：

```bash
./bin/sing-box-sub
```

也可以直接用 Go 构建：

```bash
go build -o ./bin/sing-box-sub ./cmd/sing-box-subscribe-cli
```

运行测试：

```bash
go test ./...
```

## 使用方式

从订阅 URL 生成 `config.json`：

```bash
sing-box-sub \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

订阅来源也可以通过 `--url` 传入：

```bash
sing-box-sub \
  --url 'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

从本地订阅文件生成：

```bash
sing-box-sub ./subscription.yaml --out config.json
```

只输出生成的 outbounds，不合并模板：

```bash
sing-box-sub ./subscription.yaml --only-nodes --out nodes.json
```

使用其他内置模板：

```bash
sing-box-sub ./subscription.yaml \
  --template config_template_groups_rule_set_tun.json \
  --out config.json
```

使用模板文件或模板 URL：

```bash
sing-box-sub ./subscription.yaml \
  --template ./my-template.json \
  --out config.json
```

过滤并重命名生成的节点：

```bash
sing-box-sub ./subscription.yaml \
  --prefix "Home - " \
  --exclude-protocol "ssr" \
  --exclude-node-name "expired|test" \
  --out config.json
```

如果环境里有对应工具，可以校验生成结果：

```bash
jq empty config.json
sing-box check -c config.json
```

## 内置模板

列出内置模板：

```bash
sing-box-sub list
```

当前模板：

- `config_template_groups_rule_set_tun.json`
- `config_template_groups_rule_set_tun_fakeip.json`
- `config_template_no_groups_tun_VN.json`
- `sb-config-1.12.json`
- `sb-config-1.14.json`

默认模板是：

```text
sb-config-1.14.json
```

当 `--template` 设置为内置模板名时，会优先使用打包进二进制的模板。也支持显式路径，例如 `./template.json`、`/path/to/template.json`，以及 HTTP/HTTPS URL。

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
| `--out` | `config.json` | 输出配置路径；使用 `-` 输出到 stdout。 |
| `--tag` | `tag_1` | 订阅组 tag。 |
| `--ua` | `clashmeta` | 请求订阅和模板时使用的 User-Agent。 |
| `--prefix` | 空 | 添加到生成 outbound tag 前面的前缀。 |
| `--exclude-protocol` | `ssr` | 跳过的协议，多个值用逗号分隔。 |
| `--exclude-node-name` | 空 | 按节点 tag 子串过滤，支持逗号或竖线分隔。 |
| `--only-nodes` | `false` | 只写出生成的 outbounds，不合并模板。 |
| `--timeout` | `60s` | HTTP 请求超时时间。 |

## 发布流程

维护者通过推送 Git tag 发布版本：

```bash
git tag v0.1.0
git push origin v0.1.0
```

release workflow 会运行测试，交叉编译 Linux、macOS 和 Windows 的 `amd64`、`arm64` 二进制文件，把 tag 注入 `sing-box-sub version`，并上传到 GitHub Releases。

合并请求会运行 CI workflow，执行测试并确认同一套目标矩阵可以正常编译。

## 隐私和 fixture

订阅 URL 通常包含私有 token。不要把真实订阅 URL、token 或节点内容提交到 issue、文档、测试或 fixture。复现问题时请使用脱敏示例或本地 fixture。

## License

本项目使用 [Apache License 2.0](LICENSE) 开源协议。
