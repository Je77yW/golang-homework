# Homework04：个人博客后端

使用 Gin、GORM 和 SQLite 实现用户注册登录、JWT 认证、文章 CRUD 和评论功能。

## 环境与启动

- Go 1.27.3 或更高版本（与 `go.mod` 一致）。
- 启用 CGO，并安装 C 编译器。SQLite 驱动依赖 CGO；macOS 可安装 Xcode Command Line Tools，Linux 可安装 GCC。

在项目根目录运行：

```sh
go mod download
export JWT_SECRET="$(openssl rand -hex 32)"
go run .
```

默认监听 `:8080`，也可以设置 `ADDR=127.0.0.1:8081`。`JWT_SECRET` 必须至少 32 字节；未设置或长度不足时程序会报错退出。需要保持登录状态时，应妥善保存并在重启后复用同一密钥；更换密钥会使旧令牌失效。

`util/helpers.go` 负责创建数据库目录和连接 SQLite，默认文件是相对于启动目录的 `db/blog.db`，启动时自动创建 `users`、`posts`、`comments` 三张表。请从项目根目录启动。数据库文件不纳入 Git。

## 接口

| 方法 | 路径 | 认证 | 请求体 / 功能 |
| --- | --- | --- | --- |
| POST | `/register` | 否 | `username`、`password`、`email`，注册用户 |
| POST | `/login` | 否 | `username`、`password`，返回 JWT |
| GET | `/posts` | 否 | 获取所有文章，按创建时间倒序 |
| GET | `/posts/:id` | 否 | 获取单篇文章 |
| POST | `/posts` | 是 | `title`、`content`，创建文章 |
| PUT | `/posts/:id` | 作者 | `title`、`content`，更新文章（两个字段都必填） |
| DELETE | `/posts/:id` | 作者 | 删除文章及其评论 |
| POST | `/posts/:id/comments` | 是 | `content`，发表评论 |
| GET | `/posts/:id/comments` | 否 | 获取该文章的评论，按创建时间正序 |

请求体使用 JSON；认证接口需要 `Authorization: Bearer <token>`。JWT 使用 HS256 签名，有效期 24 小时，校验签名、算法、签发者和过期时间，并确认用户仍然存在。

注册时用户名最长 50 个字符且唯一，邮箱必须合法且唯一（统一转小写存储）；密码至少 8 个字符、最多 72 字节，使用 bcrypt 哈希保存，任何响应均不返回密码。标题最长 200 个字符；用户名、密码、标题、正文和评论不能全部为空白。作者 ID 来自认证信息，评论的文章 ID 来自 URL，客户端不能通过请求体冒用身份或修改归属。

注册、创建文章及评论返回 `201`；登录、查询、更新返回 `200`；删除返回 `204` 且无响应体。用户、文章、评论响应直接返回对象，列表返回数组（空列表为 `[]`）。登录响应如下：

```json
{"token":"<JWT>","token_type":"Bearer","expires_at":"2026-11-11T12:00:00Z"}
```

错误统一返回 `{"error":"错误说明"}`：

| 状态码 | 含义 |
| --- | --- |
| 400 | 参数无效、非法 ID、JSON 错误或请求体超限 |
| 401 | 登录失败、缺失/无效/过期令牌、令牌用户不存在 |
| 403 | 试图修改或删除他人的文章 |
| 404 | 文章不存在（包括查询/创建其评论）或路由不存在 |
| 405 | 路由不支持该 HTTP 方法 |
| 409 | 用户名或邮箱重复 |
| 500 | 数据库或内部错误，详细原因记录在服务日志中 |

日志通过标准库 `log/slog` 输出请求方法、路径、状态码、耗时和系统错误；GORM 输出数据库错误和慢查询，隐藏 SQL 参数。
