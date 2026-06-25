# sing-box-subscribe-cli Project Notes

## 项目定位

这是一个从 `sing-box-subscribe` Python 项目拆出来的纯 Go CLI，用于从订阅 URL 和 sing-box JSON 模板生成最终 `config.json`。Go module 是 `github.com/rainbend/sing-box-subscribe-cli`。

当前目标不是完整复刻原 Python Web 服务，而是提供可独立运行的 CLI：

- 指定订阅来源：位置参数或 `--url`
- 指定模板：`--template`，默认 `sb-config-1.14.json`
- 指定输出文件：`--out`
- 列出内置模板：`list`

## 当前能力

- 支持拉取 HTTP/HTTPS 订阅，也支持读取本地订阅文件。
- 支持解析常见 Clash YAML `proxies` 格式。
- 已实现 Clash `vmess` 和 `hysteria2` 到 sing-box outbound 的转换。
- 支持已有 sing-box `outbounds` 输入，跳过 `selector`、`urltest`、`direct`、`block`、`dns` 等模板型 outbound。
- 支持模板中的 `{all}` 占位符展开。
- 支持 selector/urltest 上的 `filter` 规则，规则展开后会移除 `filter` 字段。
- 支持 `--exclude-protocol`、`--exclude-node-name`、`--prefix` 和 `--only-nodes`。
- WireGuard outbound 会按原 Python 行为迁移到顶层 `endpoints`。

不要把真实订阅 URL、token 或节点内容写进文档、测试 fixture 或源码。需要验证真实订阅时，应只保留脱敏后的统计信息或使用本地 fixture。

## 内置模板

模板由 `internal/templates` 包通过 Go `embed.FS` 管理。模板 JSON 文件放在 `internal/templates/*.json`，`templates.go` 负责列出和读取它们。

当前内置模板：

- `config_template_groups_rule_set_tun.json`
- `config_template_groups_rule_set_tun_fakeip.json`
- `config_template_no_groups_tun_VN.json`
- `sb-config-1.12.json`
- `sb-config-1.14.json`

生成配置时默认使用内置模板 `sb-config-1.14.json`；显式传入 `--template sb-config-1.14.json` 仍会优先读取内置模板名。显式路径和 HTTP/HTTPS URL 仍然可用。显式路径包含 `/`、`\`、`./`、`../` 或绝对路径时会按文件路径读取。

## 目录结构

- `cmd/sing-box-subscribe-cli/main.go`: CLI 入口和 `list` 子命令。
- `internal/subconv/generate.go`: 生成流程编排。
- `internal/subconv/load.go`: 读取 URL、文件或内置模板。
- `internal/subconv/parse.go`: 订阅内容识别和分发。
- `internal/subconv/clash.go`: Clash proxy 到 sing-box outbound 的转换。
- `internal/subconv/template.go`: 模板 `{all}` 展开和 outbound 合并。
- `internal/subconv/uri.go`: URI 订阅解析，目前覆盖 `vmess` 和 `hysteria2`。
- `internal/templates/`: 内置模板和模板列表/读取逻辑。

## 常用命令

列出模板：

```bash
go run ./cmd/sing-box-subscribe-cli list
```

用内置模板生成配置：

```bash
go run ./cmd/sing-box-subscribe-cli \
  'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

也可以继续使用 `--url`：

```bash
go run ./cmd/sing-box-subscribe-cli \
  --url 'https://example.com/api/v1/client/subscribe?token=REDACTED' \
  --out config.json
```

测试和构建：

```bash
go test ./...
go build -o /tmp/sing-box-subscribe-cli ./cmd/sing-box-subscribe-cli
jq empty config.json
```

如果环境具备 `sing-box` 命令，应补充 `sing-box check -c config.json` 或等价校验。

## 维护注意事项

- 新增协议时优先补 `internal/subconv/clash.go` 和对应单元测试，再考虑 URI parser。
- 不要为了类型完整性把整个 sing-box schema 强建模，模板和节点仍以 `map[string]any` 为主，更适合跟随 sing-box 配置变化。
- 修改模板后至少运行 `jq empty internal/templates/*.json` 和 `go test ./...`。
- 修改模板展开逻辑后，重点检查 selector/urltest 的 `outbounds` 是否为空。
- 如果需要继续对齐原 Python 行为，应先用脱敏输入或 fixture 固定期望输出，再移植边缘兼容逻辑。
