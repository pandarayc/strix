# TUI Roadmap

## Phase 1 — 快速修复（体验立即可感知） ✅

- [x] **1.1 Status Bar 增强** — 加入上下文占比%、花费估算、plan mode 标记、session ID
- [x] **1.2 Spinner 上下文** — 显示当前工具名："⠹ ReadTool..."
- [x] **1.3 权限对话框修复** — lipgloss 动态宽度 + 加"Always Deny"选项
- [x] **1.4 消息分隔** — user 消息前加空行，assistant 块加左边框色条
- [x] **1.5 Diff 渲染** — Edit 结果用绿色+/红色-（已有 DiffStyles 直接用）
- [x] **1.6 欢迎页** — 启动显示版本 + 快捷键 + 当前模型
- [x] **1.7 输入历史** — Ctrl+P/Ctrl+N 上下切换历史输入
- [x] **1.8 Tool 截断改进** — 加"(expand with Enter)"提示

## Phase 2 — 结构重构

- [ ] **2.1 文件拆分** — model.go(517行) → model.go + view.go + update.go + commands.go
- [ ] **2.2 布局修复** — Header 固定顶部，viewport 仅滚动 messages
- [ ] **2.3 命令统一** — TUI 复用 cli.CommandRegistry，删除重复 switch-case
- [ ] **2.4 流式优化** — strings.Builder 替代 += 拼接

## Phase 3 — 交互增强

- [ ] **3.1 Tool 展开/折叠** — Enter 键 expand 完整 tool 输出
- [ ] **3.2 多行输入** — Alt+Enter 换行，Enter 发送
- [ ] **3.3 消息虚拟化** — 大量消息时只渲染可见区域
