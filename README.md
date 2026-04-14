# 更新日志

- **2026/01/19:** 完成钉钉聊天记录解密，提供 WEB UI 用于交互查看数据
- **2026/01/20:** 增加 `--token` 参数，允许用户提供登录态以实现自动下载远端静态资源。（不提供此参数某些图片将会无法加载）
  - 打开[钉钉官网](https://www.dingtalk.com/) 登录个人账号，从 Cookie 中获取 `account` 的值传入 `--token` 即可，示例: `--token oauth_k1%3****A%3D`
- **2026/04/14:** 增加钉钉 PC **V3** 数据目录解密（`-userconfig` / `-salt` + `real_uid`）；V2 仍使用目录 uid 与 `-k`。

# 为什么会有这个项目？

在实战红蓝对抗中，红队通常通过钓鱼、EDR 利用等手段获得办公网内员工机器的控制权，**后续需要借助后渗透工具从 IM 应用中提取敏感信息进行更深入的信息收集。**

# 使用方法

## 1. 解密密钥怎么来

钉钉 PC 客户端把聊天等数据放在用户目录下的 SQLite 里，库文件是**整库 AES 加密**的。密钥规则随「数据目录版本」不同：

### V2（目录名形如 `{纯数字uid}_v2`）

- **`-k`**：填资源管理器里看到的 **目录名里的那段数字 uid**（与 `C:\Users\{用户名}\AppData\Roaming\DingTalk\{uid}_v2` 文件夹名一致）。
- 程序内部使用 `MD5(uid)` 的十六进制串的前 16 个字符作为 AES 密钥（与旧版逆向结论一致）。

### V3（目录名形如 `{标识}_v3`，标识常为 16 进制字符串）

- **`-k`**：必须填 **`real_uid`**（业务 uid，常见为纯数字），**不要**把 `_v3` 文件夹名的前缀当成 `-k`。
  - 在 **`%AppData%\Roaming\DingTalk\log`** 下打开近期的 **`gaea.log.*`**，全文搜索 **`real_uid`**，日志里形如 `real_uid=123456789` 的取值即为 `-k`。
- **盐值 `salt`**（二选一提供即可）：
  - **推荐**：**`-userconfig`** 指向同账号 `_v3` 目录下的 **`user_config`** 文件（整文件为 Base64，程序会解码 JSON 并读取 `salt` / `slt` 字段）。
  - 或 **`-salt`**：手动填入解码后 JSON 里 `salt` 的字符串（会进 shell 历史，一般不如 `-userconfig`）。

V3 密钥派生（实现与公开逆向一致，供排错）：`password = real_uid字符串 + salt字符串` → PBKDF2-HMAC-SHA1（盐为固定串 `666DingTalk888` 的前 8 字节，迭代 1000，输出 32 字节）→ 对该结果做 MD5 → 取 MD5 十六进制串的**前 16 个字符**作为 AES 密钥（与 V2 最终密钥形态一致）。

**敏感提示**：`user_config`、`real_uid`、解密后的 `dingtalk.db` 均属于账号与聊天数据，勿提交到公开仓库或外传。

---

## 2. 数据库文件在哪里、如何解密

### 典型路径（PC 钉钉）

主库一般在：

```text
C:\Users\{你的Windows用户名}\AppData\Roaming\DingTalk\{用户目录标识}_v3\DBFiles\dingtalk.db
```

示例（仅说明结构，请替换为你本机用户名与目录名）：

```text
C:\Users\19079\AppData\Roaming\DingTalk\fcd904a4e78405a35634_v3\DBFiles\dingtalk.db
```

同目录下还会有 **`user_config`**（与 `DBFiles` 同级，在 `_v3` 根目录），解密 V3 库时需要配合使用。

### 如何拷贝到本工具所在环境

- 在 Windows 上可直接把 **`DBFiles\dingtalk.db`** 和（若需要 V3 解密）**`user_config`** 复制到运行 `dingwave` 的机器。
- 若在 **WSL** 中访问 Windows 盘，通常对应：
  - `/mnt/c/Users/{用户名}/AppData/Roaming/DingTalk/{标识}_v3/DBFiles/dingtalk.db`
  - `/mnt/c/Users/{用户名}/AppData/Roaming/DingTalk/{标识}_v3/user_config`

### 解密命令示例

**V2（仅 `-k` 为目录数字 uid）**

```bash
./dingwave -d dingtalk_encrypt.db -k 666165872 -o dingtalk_plain.db
```

**V3（`-k` 为 `real_uid`，并指定 `user_config`）**

```bash
./dingwave -d dingtalk.db \
  -k <从gaea日志中得到的real_uid> \
  -userconfig "/mnt/c/Users/你的用户名/AppData/Roaming/DingTalk/你的标识_v3/user_config" \
  -o dingtalk_plain.db
```

已得到明文库后，查看聊天记录时可直接：

```bash
./dingwave -d dingtalk_plain.db
```

---

## 3. 如何编译与启动

### 环境要求

- **Go**：与 `server/go.mod` 中版本一致（用于编译后端）。
- **Node**：`^20.19.0` 或 `>=22.12.0`（见 `frontend/package.json`）。
- **pnpm**：`npm install -g pnpm` 或使用 corepack。

### 编译前端（生成嵌入后端的静态资源）

```bash
cd frontend
pnpm install
pnpm build
```

构建结果输出到 **`server/dist`**（由 `frontend/vite.config.ts` 的 `outDir` 指定）。

### 编译单个本机可执行文件（推荐日常自用）

```bash
cd server
go build -ldflags "-s -w" -trimpath -o ../dingwave .
```

若从未构建过前端，`go build` 会因缺少 `server/dist` 失败；请先完成上一步 **`pnpm build`**。

### 启动服务（Web 查看聊天记录）

```bash
# 在仓库根目录，默认端口 8080
./dingwave -d dingtalk_plain.db

# 指定端口
./dingwave -d dingtalk_plain.db -p 9090
```

浏览器访问 **http://127.0.0.1:8080**（端口与 `-p` 一致）。结束服务：在终端 **Ctrl+C**。

**可选**：部分图片需钉钉登录态拉取，可从钉钉官网 Cookie 取 `account` 传入：

```bash
./dingwave -d dingtalk_plain.db --token 'oauth_k1%3A...'
```

**一键全量构建（多平台交叉编译）**：仓库根目录执行 `./build.sh`（依赖 `pnpm` 与 `go`，会先构建前端再编译各平台二进制到 `releases/`）。

---

## 快速使用（命令速查）

输入为**已解密**的数据库：

```bash
./dingwave -d dingtalk.db
```

输入为**加密**数据库（V2）：

```bash
./dingwave -d dingtalk_encrypt.db -k 666165872
```

加密库解密并**另存明文**：

```bash
./dingwave -d dingtalk_encrypt.db -k 666165872 -o dingtalk.db
```

# 已实现功能

## 查看所有会话

参考了钉钉GUI的设计，将会话区分为置顶、单聊与群聊，按最后一条消息发送日期降序展示：

<img width="2784" height="1602" alt="802b458c973f48027c82856d4b317d28" src="https://github.com/user-attachments/assets/93a71ef3-b2eb-4ffa-a5c7-4fe41e91161e" />

点击某个具体的会话类型后，会展开同类型的所有会话：

![d859893f448fed9f9b350cd677dda591](https://github.com/user-attachments/assets/7d79d8d9-0ae8-4607-ab4f-a9d5d7b42c07)

## 完整解析钉钉消息数据类型

程序已经实现了钉钉内大部分数据类型进行解析，无奈有的数据（例如附件、头像）是存储在本地的，仅通过数据库无法获取更多有效信息，**因此暂时只提供了部分图片以及部分附件的解析。**

<img width="2784" height="1602" alt="73218ec4fdfe65ba0e69461565b8c44e" src="https://github.com/user-attachments/assets/3b355698-df80-4f95-87ca-f0e60701ad8d" />

## 全局搜索聊天记录

<img width="2784" height="1602" alt="a74a24b705e9b8f420edfdb90e54ccc1" src="https://github.com/user-attachments/assets/f12158c3-7f76-4198-99f5-117aa6ec31e7" />

点击某个搜索会话，会展开所有搜索到的消息，同时高亮搜索内容：

<img width="522" height="944" alt="794eea21a999ced13bf7a9d9840ce93b" src="https://github.com/user-attachments/assets/0be637ac-e028-45ff-b494-f838884fff2d" />

点击会话内的具体搜索结果，会跳转到对应的聊天片段中，可以像正常查阅聊天内容一样上滑与下拉：

![695033e64497021be2460ed2fb6b160c](https://github.com/user-attachments/assets/335e21c6-3db9-4cf1-8992-a8ead93936a5)

**当然也支持指定某个会话进行搜索，实现效果与全局搜索效果一致，这里不再赘述。**

## 查看联系人列表

![4d5c7488800aaf866eebdd52518185a5](https://github.com/user-attachments/assets/669426fe-3d60-48aa-a270-31b24878365f)
