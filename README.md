# hook2feishu

> 通用编程工具 → 飞书通知网关

当 **Claude Code**、**Codex CLI** 等支持 hooks 的编程工具任务暂停或有请求时，
自动通过 [lark-cli](https://github.com/larksuite/cli) 推送通知到飞书。

## 支持的工具

| 工具 | 事件 | 自动检测 |
|------|------|----------|
| **Claude Code** | Stop / Notification / PreToolUse 等 | ✅ |
| **Codex CLI** | stop / notification / request | ✅ |
| 其他 hooks 工具 | 尽力解析 | ✅ |

## 快速开始

### 1. 下载

从 [Releases](https://github.com/Yarrow-Cai/hook2feishu/releases) 下载对应平台的二进制：

- `hook2feishu.exe` — Windows
- `hook2feishu_darwin_amd64` — macOS Intel
- `hook2feishu_darwin_arm64` — macOS Apple Silicon
- `hook2feishu_linux_amd64` — Linux

放到任意目录，记下路径。

### 2. 配置

在二进制同目录（或 `~/.config/hook2feishu/`）创建 `config.json`：

```json
{
  "open_id": "ou_xxx",
  "events": ["Stop", "Notification"],
  "lark_cli_profile": "business"
}
```

**必填项**：

| 字段 | 说明 |
|------|------|
| `open_id` | 飞书用户 open_id，消息发给谁 |

**可选项**：

| 字段 | 默认 | 说明 |
|------|------|------|
| `lark_cli_path` | `lark-cli`（从 PATH 查找） | lark-cli 二进制路径 |
| `lark_cli_profile` | 无 | lark-cli 配置文件名 |
| `events` | `["Stop", "Notification"]` | 要推送的事件类型 |
| `quiet_hours` | 无 | 静默时段，如 `[22, 8]` |
| `min_duration` | `0` | 最短任务时长（秒），短于此值不推送 |
| `tz_offset` | `8` | 时区偏移（东八区） |

配置文件查找顺序：
1. `HOOK2FEISHU_CONFIG` 环境变量指定路径
2. 二进制同目录 `config.json`
3. `~/.config/hook2feishu/config.json`

### 3. 接入工具

#### Claude Code

在 `.claude/settings.json` 中配置 hooks：

```json
{
  "hooks": {
    "Stop": [
      {
        "command": "/path/to/hook2feishu"
      }
    ],
    "Notification": [
      {
        "command": "/path/to/hook2feishu"
      }
    ]
  }
}
```

#### Codex CLI

在 Codex 配置中设置 hooks（具体字段见 Codex 文档）：

```json
{
  "hooks": {
    "stop": "/path/to/hook2feishu",
    "notification": "/path/to/hook2feishu"
  }
}
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `HOOK2FEISHU_CONFIG` | 指定配置文件路径 |
| `HOOK2FEISHU_DEBUG` | 设为 `1` 开启调试日志（写入 `~/.config/hook2feishu/debug.log`） |

## 工作原理

```
Claude Code / Codex / ...
    │  (stdin JSON)
    ▼
hook2feishu
    │  ① 自动检测工具类型
    │  ② 解析事件 + transcript + git
    │  ③ 构建飞书交互卡片
    └─→ lark-cli ─→ 飞书消息
```

- 无需直接调用飞书 API，认证由 `lark-cli` 管理
- 多工具共用一条推送通道
- 零外部依赖，纯 Go 标准库

## 编译

```bash
# 本地编译
make build

# 全平台 Release
make release
```

依赖：Go 1.26+、`lark-cli`

## 许可证

MIT
