# Dingwave 会话与消息查询 Agent Skill — 设计说明

**日期**：2026-04-14  
**状态**：已定稿待实现  
**范围**：在仓库内新增 `.agents/skills` 技能包（`SKILL.md` + 同目录下 `scripts/`），通过本地 Dingwave HTTP API 列出会话、分页拉取消息、关键字搜索、围绕某条消息查看上下文。

---

## 1. 背景与目标

用户在本地运行 Dingwave 服务（默认 `http://127.0.0.1:8080`），通过日志或前端可得到形如  
`GET /api/conversations/64096007722/messages?before=1769652371658` 的请求。  
需要一种**可重复、与实现一致**的方式：列会话、进入某会话、按关键字命中某条消息后查看前后文。

**非目标**：不修改服务端行为；不在技能内嵌入数据库直连；不处理需登录的远程部署（仅约定本地 `BASE_URL`）。

---

## 2. 交付物位置与目录结构

根路径（本仓库根目录下）：

`.agents/skills/dingwave-messages/`

建议布局：

```text
.agents/skills/dingwave-messages/
├── SKILL.md              # 元数据 + 给 Agent 的分步说明、与 API 对齐的语义
└── scripts/
    └── dwmsg.py          # Python 3，仅标准库 HTTP/JSON/argparse
```

可选（本期不做）：`reference.md`（长表格、排错）、`examples.md`。

---

## 3. 依赖与运行前提

- **Python**：3.8+（使用 `urllib.request`、`json`、`argparse`）。
- **Dingwave**：已用解密后的数据库启动 HTTP 服务（与 README 一致）。
- **环境变量**：
  - `BASE_URL`：默认 `http://127.0.0.1:8080`（与 `server/main.go` 中 `-p` 默认 `8080` 及 README 一致）；若用户改端口则必须设置。

执行方式（写入 `SKILL.md`，避免路径歧义）：

在技能根目录执行：`python3 scripts/dwmsg.py <子命令> ...`  
或使用绝对路径调用 `dwmsg.py`（仓库克隆路径因人而异，技能内优先写「先 `cd` 到技能根」）。

---

## 4. 与服务端契约（只读 API）

以下与 `server/internal/server/server.go`、`message_handler.go`、`conversation_handler.go` 一致。

| 能力 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 首页分区 | GET | `/api/conversations/home` | Query：`limit`（默认 5） |
| 会话列表 | GET | `/api/conversations` | Query：`type`、`page`、`size`（分页默认 20）、`order`（默认 `time`） |
| 会话消息 | GET | `/api/conversations/:cid/messages` | Query：`before`、`after`（int64 时间戳）、`size`（默认 50） |
| 会话内搜索 | GET | `/api/conversations/:cid/messages/search` | Query：`q`（必填）、`page`、`size` |
| 全局搜索 | GET | `/api/messages/search` | Query：`q`（必填）、`page`、`size` |

成功时响应体为 **直接 JSON 对象**（`Success` 返回 `data` 本体，无统一 `{code,data}` 包裹）。

**分页语义（`message_service.go`，实现上下文时必须遵守）**：

- `before = T`：筛选 `created_at < T`，取一页；服务内部先按时间倒序截取再反转，返回的 `items` 为**时间升序**的一段，靠近 `T` 的消息在列表**靠后**（即「更旧历史中紧邻 T 之前」的一页）。
- `after = T`：筛选 `created_at > T`，与 `before` 对称处理，得到**时间升序**、紧邻 `T` **之后**的一页（靠前者更靠近 `T`）。
- 未传 `before`/`after`：等价于从最新往旧取一页（仍经反转后为升序展示逻辑与前端对齐）。

消息锚点字段：使用 **`created_at`（毫秒时间戳）** 与 API 一致；`id` 仅作展示与去重键。

---

## 5. `dwmsg.py` 子命令设计

所有子命令在失败时：非 2xx 时把响应体打印到 stderr（或 stdout 一段清晰错误），退出码非 0。

| 子命令 | 作用 | 主要参数 |
|--------|------|-----------|
| `home` | 拉取首页分区 | `--limit` |
| `conversations` | 分页会话列表 | `--type`、`--page`、`--size`、`--order` |
| `messages` | 单会话分页消息 | `cid`、`--before`、`--after`、`--size`（互斥：before 与 after 不宜同时传；若同时传则文档规定优先顺序或报错，**建议：二者互斥，同时传则报错**） |
| `search-conv` | 会话内关键字 | `cid`、`--q`（必填）、`--page`、`--size` |
| `search-global` | 全局关键字（按会话聚合） | `--q`（必填）、`--page`、`--size` |
| `context` | 围绕锚点拼上下文 | `cid`、`--around <created_at>`、`--window`（默认如 15，表示 before 侧与 after 侧各取条数，或定义为「每侧条数」）；实现：分别调用 `messages?after=around&size=window` 与 `messages?before=around&size=window`，按 `id` 去重、`created_at` 升序合并，输出列表并在 `--around` 命中行标注 |
| `parse-log` | 从一行日志解析参数 | stdin 或参数传入一行，提取 `cid`、以及 `before`/`after`（若存在），**仅打印 JSON 一行**，便于管道进 `messages`（可选；若实现成本高可第一期只做文档中的 grep 示例，**本设计定为第一期包含**：单行正则即可） |

输出格式：

- 默认：表格化简要文本（时间、`sender_name` 或 `sender_id`、类型、`content_text` 截断）。
- `--json`：原始或规整后的 JSON，便于 Agent 再加工。

---

## 6. `SKILL.md` 内容要点

**YAML `description`（第三人称、含触发词）**：  
说明该技能在本地 Dingwave 已启动时，通过仓库内脚本查询会话列表、会话消息、会话内/全局关键字搜索、围绕 `created_at` 查看上下文；在用户提到对话列表、某 `cid`、日志中的 `GET /api/conversations/.../messages`、关键字定位、前后文时使用。

**正文章节建议**：

1. 前置：`BASE_URL`、确认服务可访问。
2. 路径：技能根目录与 `python3 scripts/dwmsg.py` 的调用方式。
3. 工作流示例：  
   - 列会话 → 选 `cid` → `messages` 最新页；  
   - 日志解析 → `parse-log` → `messages --before ...`；  
   - `search-global` / `search-conv` 得 `created_at` → `context --around ...`。
4. **禁止臆测分页方向**：明确引用本节第 4 与 `message_service` 行为一致。
5. 边界：大 `content_text` 截断、撤回消息 `is_recall` 展示说明。

语言：与用户仓库一致，**中文**为主（代码与 CLI 英文）。

---

## 7. 测试与验收

- 本地起服务后：  
  - `dwmsg.py conversations --size 5` 有 `items`；  
  - 对已知 `cid`：`messages` 与浏览器 Network 同参数条数一致；  
  - `search-conv` 与 `search-global` 与 API 一致；  
  - `context` 在插入已知锚点后，前后条数与顺序正确、无重复 `id`。

自动化：本期以 **手工验收** 为主；不在设计内强制 pytest（可选后续）。

---

## 8. 范围与后续

**本期包含**：目录与文件创建、`SKILL.md`、`scripts/dwmsg.py` 实现上述子命令、README 中可选一行指向该技能（若用户希望在主 README 暴露入口；**默认不改 README**，避免用户未要求的文档扩散）。

**后续可选**：`content_json` 折叠展示、图片 URL 解析、与 `scrapling` 等技能交叉引用。

---

## 9. 自检（spec 质量）

- 无 TBD 占位。  
- 分页语义与 `GetConversationMessages` 实现一致。  
- 技能路径已固定为仓库内 `.agents/skills/dingwave-messages/`。  
- 脚本位置已固定为技能内 `scripts/`。
