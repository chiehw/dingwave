---
name: dingwave-messages
description: >-
  在本地 Dingwave 仓库中通过合并 SQLite（conversations/messages/users）离线查询钉钉会话与消息：
  列会话、分页消息、关键字搜索、围绕某条消息的上下文；可解析日志里的 GET /api/conversations/.../messages。
  在用户提到对话列表、cid、合并库、merged-out、离线查库、前后文、不启服务查消息时使用。
---

# Dingwave 会话与消息（离线合并库）

## 1. 原则

- 数据来自 **合并后的 SQLite**（表 `messages`、`conversations`、`users`），**不要**直接对未合并的 `tbmsg_*` 写 SQL。
- 查库前由 **`scripts/ensure_merged.py`** 负责：**检查**合并库是否存在且含 `messages` 表；若缺失或源库 `-d` 比合并库新，则 **自动调用** `dingwave -merged-out … -export-only` 再生成。
- 合并库路径默认：本技能目录下 **`cache/merged.db`**（已被仓库根 `.gitignore` 的 `*.db` 忽略，勿提交）。

## 2. 环境变量（用户或你在会话里一次性配置）

| 变量 | 含义 |
|------|------|
| `DINGWAVE_SOURCE_DB` | 与 `dingwave -d` 相同（加密库或明文库路径） |
| `DINGWAVE_EXTRA_FLAGS` | 解密/密钥参数，一行字符串，用 **shell 风格引号** 包住含空格的项；脚本用 `shlex` 拆成 argv。例：`-k 123456 -salt abc` 或带 `-userconfig` |
| `DINGWAVE_MERGED_DB` | 合并库输出路径（可选；不设则用 `cache/merged.db`） |
| `DINGWAVE_BIN` | `dingwave` 可执行文件路径（可选；不设则先尝试**仓库根**下 `./dingwave`，再用 PATH） |
| `DINGWAVE_DB` | 给 `dwmsg.py` 等查询脚本用的库路径；**建议在 ensure 成功后设为 ensure 打印的路径** |

## 3. 标准流程（Agent 每轮查消息前执行）

1. **定位技能根目录**：仓库内 `.agents/skills/dingwave-messages/`（本文件所在目录）。
2. **运行检查/生成**（在技能根下）：

   ```bash
   python3 scripts/ensure_merged.py
   ```

   - 成功时脚本向 **stdout 打印一行**合并库绝对路径，向 stderr 打日志；用该路径作为 `DINGWAVE_DB` 或传给 `dwmsg.py --db`。
   - 若提示缺少 `DINGWAVE_SOURCE_DB`，请向用户说明需设置源库与 `DINGWAVE_EXTRA_FLAGS` 后再试。

3. **强制全量重导**（用户改密钥或怀疑损坏时）：

   ```bash
   python3 scripts/ensure_merged.py --force
   ```

4. 再运行 **`dwmsg.py`**（实现后）做 `conversations` / `messages` / `search-*` / `context` 等；若尚未实现 `dwmsg.py`，可用 `sqlite3` 只读打开上一步路径自行查表。

## 4. 与 Go 程序的关系

- 生成合并库等价于执行：

  `dingwave -d <源> <与现有一致的解密参数> -merged-out <路径> -export-only`

- 带 **`-merged-out` 且不设 `-export-only`** 时：迁移写入该文件并 **照常启动 HTTP**，服务端与离线脚本可共用同一文件型库（注意并发写风险；只读查询安全）。

## 5. 分页语义（与线上一致，避免猜反）

- `before = T`：`created_at < T`，取一页后按时间 **升序** 展示「紧邻 T 之前」的一段。
- `after = T`：`created_at > T`，同理为 **升序**、「紧邻 T 之后」的一段。

详见仓库内 spec：`docs/superpowers/specs/2026-04-14-dingwave-messages-skill-design.md`。
