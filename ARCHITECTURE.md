# Ergate — Architecture & Roadmap

Go-native AI-powered software engineering CLI，对标 Claude Code 并在 Go 生态中延伸。

支持 Anthropic / OpenAI / DeepSeek 三个 API provider，单二进制分发。

## Architecture

```
cmd/ergate/main.go              — 入口，启动 Cobra RootCmd
    │
internal/cli/                   — CLI 层
    │
    ├── bridge.go               — 依赖注入中心
    │
    ├── engine/                 — 核心循环：chat → tools → chat (786 lines)
    ├── llm/                    — Provider 抽象 + API 客户端 (anthropic/openai/deepseek)
    ├── tool/                   — 工具接口、注册表、9 个内置工具
    ├── task/                   — 后台任务 (bash subprocess + agent sub-LLM)
    ├── mcp/                    — MCP 协议 (stdio/sse/http transport)
    ├── memory/                 — 持久记忆 (.ergate/memory/*.md)
    ├── skill/                  — 技能系统 (SKILL.md frontmatter + 条件触发)
    ├── compact/                — 上下文压缩 (micro + auto via LLM summary)
    ├── planmode/               — Plan / Implement 状态机
    ├── hooks/                  — 工具生命周期回调 (pre/post/onStop)
    ├── filehistory/            — 文件修改前自动备份
    ├── worktree/               — Git worktree 创建/清理
    ├── session/                — 会话 JSON 持久化
    ├── config/                 — YAML + 环境变量配置 (ERGATE_* 前缀)
    ├── tui/                    — Bubbletea 终端 UI
    └── util/                   — Markdown 终端渲染
```

## Data Flow

```
User Input → Engine.Run()
  → buildSystemPrompt()          注入 memory + skills + planmode + agent instructions
  → [maybeCompact]               超阈时触发 MicroCompact 或 LLM 摘要
  → ChatRequest → LLM API (SSE stream)
  → Event channel → TUI / REPL   实时渲染
  → executeTools()               权限检查 → hook → 执行 → hook
  → handleToolResult()           追加 tool_result message
  → loop next turn
```

## Core Types

| Type | Package | Role |
|------|---------|------|
| `Engine` | `engine` | 核心循环，管理消息历史、工具执行、事件广播 |
| `Tool` | `tool` | 工具接口：Name / Description / InputSchema / Execute / ValidateInput / CheckPermissions |
| `LLMClient` | `llm` | Provider 接口：Chat / ChatStream / Close |
| `Message` | `llm` | discriminated union (user / assistant / system / progress) |
| `Command` | `cli` | REPL 命令：Call / GetPrompt / IsEnabled |
| `Event` | `engine` | 引擎事件：text / thinking / tool_use / tool_result / error / done |

## Configuration

文件：`config.yaml`、`.ergate/config.yaml` 或 `ERGATE_*` 环境变量。

| Key | Default | Description |
|-----|---------|-------------|
| `api_provider` | `anthropic` | anthropic / openai / deepseek |
| `api_key` | — | API 密钥 |
| `model` | `claude-sonnet-4-20250514` | 模型 ID |
| `max_turns` | `20` | 每轮对话最大工具调用循环 |
| `max_tokens` | `4096` | 单次 API 响应最大 token |
| `temperature` | `0.7` | 采样温度 |
| `permission_mode` | `normal` | normal / always / bypass |
| `enable_mcp` | `false` | 启用 MCP 服务器连接 |

完整示例见 `config.example.yaml`。

## Claude Code Alignment

| Claude Code (TypeScript) | Ergate (Go) | Status |
|---|---|---|
| `QueryEngine.ts` | `engine/engine.go` | Done |
| `Tool.ts` + `tools/` | `tool/` (9 builtins) | Done |
| `commands.ts` + `commands/` | `cli/command.go` + `commands_*.go` | Done |
| `services/compact/` | `compact/compact.go` | Done |
| `memdir/memdir.ts` | `memory/memory.go` | Done |
| `skills/loadSkillsDir.ts` | `skill/skill.go` | Done |
| `Task.ts` + `tasks/` | `task/` | Done |
| `hooks/` | `hooks/hooks.go` | Done |
| `components/` (Ink/React) | `tui/` (Bubbletea) | Basic, WIP |
| `services/mcp/` | `mcp/` | Done |
| `services/lsp/` | — | Not planned |
| `coordinator/` | — | Not planned |
| `vim/` | — | Not planned |
| `voice/` / `plugins/` / OAuth | — | Out of scope |

## Roadmap

### Phase A — Stability & Usability (current)

- [x] A.1 Headless permission fix — read-only tools skip interactive Check
- [ ] A.2 HTTP 400 fix — tool result content encoding
- [ ] A.3 TUI quick fixes (8 items, see `.ergate/docs/tui-roadmap.md` Phase 1)
- [ ] A.4 TUI structural refactor (4 items, see `.ergate/docs/tui-roadmap.md` Phase 2)

### Phase B — Quality

- [ ] B.1 Test coverage for engine / llm / tool / task
- [ ] B.2 TUI interaction enhancements (Phase 3)
- [ ] B.3 E2E smoke tests (headless multi-turn)
- [ ] B.4 CI pipeline (GitHub Actions: build + test + lint)

### Phase C — Differentiation

- [ ] C.1 `/diff` and `/commit` — auto-generate commit messages
- [ ] C.2 LSP integration — goToDefinition / findReferences
- [ ] C.3 Vim keybindings — optional modal editing
- [ ] C.4 Session management — list, search, import/export
- [ ] C.5 Go-specific tooling — `go vet`, `golangci-lint` runners
