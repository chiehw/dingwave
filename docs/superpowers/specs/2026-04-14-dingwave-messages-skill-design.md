# Dingwave 会话与消息查询 Agent Skill — 设计说明

**日期**：2026-04-14  
**状态**：已修订（直连 SQLite，不依赖 HTTP）  
**范围**：在仓库内新增 `.agents/skills` 技能包（`SKILL.md` + 同目录下 `scripts/`），通过 **只读打开本地合并后的 SQLite 库** 列出会话、分页拉取消息、关键字搜索、围绕某条消息查看上下文。

---

## 1. 背景与目标

用户手上有解密后的钉钉库，经 Dingwave 迁移后得到统一的 **`conversations` / `messages` / `users`** 等表（逻辑见 `server/internal/database/database.go`）。希望用脚本在**不启动 HTTP、不调 API** 的情况下完成：列会话、按 `cid` 翻页消息、关键字搜索、围绕某条消息看前后文。  
日志中的 `GET /api/conversations/:cid/messages?before=...` 仍可作为**锚点参数**（解析出 `cid`、时间戳），但数据来自 **SQL 查询**，不是网络请求。

**非目标**：不在 Python 里重复实现解密、不直接查询未合并的 `tbmsg_*` 分表、不修改合并算法（仍以 Go 侧迁移为准）。

---

## 2. 为何不能「只指定解密后的 dingtalk.db」

当前 `main` 在解密后调用 `MigrateToMemory`：从磁盘 SQLite 读取 `tbconversation`、`tbmsg%`、`tbuser_profile_v2` 等**原始表**，写入 **`:memory:`** 中的 GORM 表，并在内存中执行 `updateContentText`、`updateSingleChatTitles`、`updateConversationStats` 等（见 `database.go`）。  
因此：**磁盘上的解密文件默认没有 `messages` 统一表**，技能脚本若只连该文件会查不到目标 schema。

**结论**：技能脚本连接的必须是 **已完成与线上一致迁移流水线、且落盘为文件型 SQLite 的「合并库」**（见第 3 节）。

---

## 3. 合并库来源（实现前提，与技能同周期交付）

仓库需增加一种**生成合并库文件**的方式（名称可任选其一，以下为建议）：

- 建议在 `server/main.go` 增加参数，例如 **`-merged-out <path>`**：在现有解密/校验/迁移逻辑之后，除内存服务外（或在不启动 HTTP 的模式下），将 **与当前 `MigrateToMemory` 相同结果** 写入 **文件 DSN** 的 SQLite（`gorm` + `glebarez/sqlite` 与现有一致），使该文件包含下列表及列。  
- 若用户未传 `-merged-out`，行为与现在一致（仅内存），不强制写盘。

技能与脚本侧约定环境变量：

- **`DINGWAVE_DB`**（必填）：合并库 SQLite 文件的绝对或相对路径。

可选：若将来支持「只导出、不监听端口」，可在 spec 实现阶段用同一 flag 组合 `Ctrl+C` 前已写出文件描述清楚即可。

---

## 4. 表与列（与 `model.go` / GORM 默认命名一致）

脚本只使用只读连接：`sqlite3.connect(..., uri=True)` + `mode=ro`（或打开后避免写操作）。

| 表名 | 说明 |
|------|------|
| `conversations` | 会话 |
| `messages` | 消息（多 `tbmsg_*` 已合并） |
| `users` | 用户昵称等 |
| `current_users` | 当前用户单行（若存在；用于 `home` 展示，可选） |

主要列（蛇形命名，与 GORM 默认一致）：

- **`conversations`**：`id`, `cid`, `type`, `title`, `is_top`, `message_count`, `last_message_at`, `last_message_id`, `last_message_preview`, `created_at`
- **`messages`**：`id`, `cid`, `original_cid`, `sender_id`, `content_type`, `content_text`, `content_json`, `created_at`, `is_recall`
- **`users`**：`id`, `nickname`, `email`

发送者展示：`messages.sender_id` **LEFT JOIN** `users.id` 得 `nickname`（与 API 中 `populateMessageSenders` 效果一致）。

---

## 5. 查询语义（与 `message_service.go` / `conversation_service.go` 对齐）

以下用于 `dwmsg.py` 的 SQL 或等价逻辑，保证与线上一致。

### 5.1 会话列表 `conversations`

与 `ConversationService.List` 一致：

- Query 参数 `type` 为 **0**（默认）：`WHERE is_top = 1`（SQLite 布尔存 0/1）。  
- `type` 为 **1** 或 **2**：`WHERE type = ?`。  
- `order=count`：`ORDER BY message_count DESC`；否则：`ORDER BY last_message_at DESC`。  
- 分页：`LIMIT size OFFSET (page-1)*size`，并单独 `COUNT(*)` 得 `total`。

### 5.2 会话消息 `messages`

与 `MessageService.GetConversationMessages` 一致（返回列表均为 **`created_at` 升序**）：

- **仅 `cid`**：`WHERE cid = ? ORDER BY created_at DESC LIMIT size+1`，判断是否 `has_more`，取前 `size` 条后 **按 `created_at` 升序**输出（与 Go 中两次反转后的顺序一致）。
- **`before = T`**：`WHERE cid = ? AND created_at < ? ORDER BY created_at DESC LIMIT size+1` → 截断 → **升序**输出。
- **`after = T`**：`WHERE cid = ? AND created_at > ? ORDER BY created_at ASC LIMIT size+1` → 截断 → **升序**输出（与 Go 对 `after` 分支的最终顺序一致）。

**约束**：`before` 与 `after` **互斥**；同时传入则报错退出。

### 5.3 会话内搜索

与 `SearchInConversation`：`WHERE cid = ? AND content_text LIKE '%' || ? || '%' ORDER BY created_at DESC LIMIT size OFFSET (page-1)*size`，并 `COUNT(*)`。

### 5.4 全局搜索

与 `SearchGlobal`：按 `cid` 聚合匹配条数，`GROUP BY cid`，`ORDER BY match_count DESC`，分页；标题等从 `conversations` 再查。

### 5.5 `home`（可选与 API 对齐）

与 `GetHome`：`is_top` 会话、`type=1` 单聊、`type=2` 群聊，各 `ORDER BY last_message_at DESC LIMIT limit`，并各 `COUNT(*)`。当前用户从 `current_users` 取首行（若无表则省略）。

### 5.6 `context`

给定 `cid` 与锚点 `created_at = A`、窗口 `window`（每侧条数，默认 15）：

- **较新**：`after` 语义 SQL → 取 `created_at > A` 升序前 `window` 条（实现上等价于 `ORDER BY created_at ASC LIMIT window` 在 `> A` 集合上取最靠近 A 的一段；与 Go 的 `after` 分页对齐时，用与 5.2 相同 LIMIT/OFFSET 规则：即一次取一页大小为 `window`，`after=A`）。
- **较旧**：`before=A`，`size=window`。
- 合并两段结果，按 `id` 去重，按 `created_at` 升序输出；锚点行若不在结果中可单独标出或插入占位说明。

### 5.7 `parse-log`

与 HTTP 版设计相同：单行正则提取 `cid`、`before`、`after`，输出一行 JSON，供后续子命令复用。

---

## 6. 交付物位置与目录结构

根路径（本仓库根目录下）：

`.agents/skills/dingwave-messages/`

```text
.agents/skills/dingwave-messages/
├── SKILL.md              # 元数据 + 环境变量、合并库生成方式、SQL 语义索引
└── scripts/
    └── dwmsg.py          # Python 3：sqlite3、json、argparse（标准库）
```

---

## 7. `dwmsg.py` 子命令（与上一版名称兼容，实现从 HTTP 改为 SQL）

| 子命令 | 作用 |
|--------|------|
| `home` | 同 API 首页分区逻辑（读库） |
| `conversations` | 分页会话列表 |
| `messages` | 单会话分页消息 |
| `search-conv` | 会话内关键字 |
| `search-global` | 全局按会话聚合 |
| `context` | 围绕 `created_at` 锚点拼上下文 |
| `parse-log` | 从日志行解析 `cid` / `before` / `after` |

公共参数：

- 必填：环境变量 **`DINGWAVE_DB`**；或通过 **`--db <path>`** 覆盖（便于单次调用）。
- **`--json`**：输出 JSON；默认表格化文本（时间、nickname、content_type、content_text 截断）。

错误：库文件不存在、缺表、SQL 失败 → stderr 清晰信息，退出码非 0。

---

## 8. `SKILL.md` 内容要点

**`description`**：说明在已存在 **合并库 SQLite**（由 Dingwave `-merged-out` 或等价方式生成）时，用 `dwmsg.py` 离线查询会话与消息；触发词含：对话列表、`cid`、日志 URL、关键字、前后文、不启动服务。

**正文**：

1. 必须先有合并库路径；如何生成（指向 README 或 `main` 帮助中的 `-merged-out`）。  
2. 设置 `DINGWAVE_DB` 或 `--db`。  
3. 工作流示例与 HTTP 版相同，仅将「服务已启动」改为「合并库已生成」。  
4. 明确禁止假设磁盘解密原库可直接当 `DINGWAVE_DB`（除非未来官方支持，当前不支持）。  
5. 分页与搜索语义以本节 spec 为准，与 `message_service` / `conversation_service` 一致。

---

## 9. 测试与验收

- 用真实数据跑一遍 `-merged-out` 生成文件，设置 `DINGWAVE_DB`。  
- `conversations` / `messages` 与**同一数据**下通过 API（若临时起服务）或单元对比抽样一致。  
- `context` 锚点前后条数、顺序、`id` 无重复。

---

## 10. 范围与后续

**本期包含**：

1. Go 侧：**`-merged-out`（或等价）** 生成合并库文件。  
2. `.agents/skills/dingwave-messages/`：`SKILL.md` + `scripts/dwmsg.py`。  

**默认不改 README**；若补充一行说明合并库与技能，需用户单独提出。

**后续可选**：只导出模式、大文本分页策略、`content_json` 摘要。

---

## 11. 自检

- 已说明「不能直连未合并解密库」的原因与数据流。  
- 查询语义与现有 Go 服务对齐。  
- 技能路径：仓库内 `.agents/skills/dingwave-messages/`。  
- 脚本在技能目录 `scripts/` 内。
