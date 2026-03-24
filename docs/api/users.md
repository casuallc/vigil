# 用户管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/users/register | POST | 注册用户 |
| /api/users/login | POST | 用户登录 |
| /api/users | GET | 列出用户 |
| /api/users/{username} | GET | 获取用户详情 |
| /api/users/{username} | PUT | 更新用户 |
| /api/users/{username} | DELETE | 删除用户 |
| /api/users/{username}/configs | GET | 获取用户配置 |
| /api/users/{username}/configs | PUT | 更新用户配置 |

---

## POST /api/users/register

**功能描述**：注册新用户

**请求参数**：
- 请求体：
  - `username`：用户名（必填）
  - `password`：密码（必填）
  - `email`：邮箱（可选）
  - `role`：角色（可选，默认：user）
  - `nickname`：昵称（可选）
  - `avatar`：头像 URL（可选）
  - `region`：地区（可选）

**请求体示例**：
```json
{
  "username": "newuser",
  "password": "securepassword",
  "email": "newuser@example.com",
  "nickname": "张三",
  "avatar": "https://example.com/avatars/user.jpg",
  "region": "北京"
}
```

**响应格式**：
```json
{
  "message": "User registered successfully",
  "user": {
    "id": "usr_1234567890",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "nickname": "张三",
    "avatar": "https://example.com/avatars/user.jpg",
    "region": "北京",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
}
```

**错误响应**：
```json
{
  "error": "Username already exists"
}
```

---

## POST /api/users/login

**功能描述**：用户登录

**请求参数**：
- 请求体：
  - `username`：用户名（必填）
  - `password`：密码（必填）

**请求体示例**：
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应示例**：
```json
{
  "message": "Login successful",
  "user": {
    "id": "usr_1773976593",
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "created_at": "2026-03-20T11:16:33.071675+08:00",
    "updated_at": "2026-03-20T11:16:33.071675+08:00"
  }
}
```

---

## GET /api/users

**功能描述**：列出所有用户

**请求参数**：无

**响应格式**：
```json
[
  {
    "id": "usr_1234567890",
    "username": "user1",
    "email": "user1@example.com",
    "role": "user",
    "nickname": "张三",
    "avatar": "https://example.com/avatars/user1.jpg",
    "region": "北京",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  },
  {
    "id": "usr_1234567891",
    "username": "user2",
    "email": "user2@example.com",
    "role": "admin",
    "nickname": "李四",
    "avatar": "https://example.com/avatars/user2.jpg",
    "region": "上海",
    "created_at": "2023-01-02T00:00:00Z",
    "updated_at": "2023-01-02T00:00:00Z"
  }
]
```

---

## GET /api/users/{username}

**功能描述**：获取用户详情

**请求参数**：
- `username`：用户名（路径参数）

**响应格式**：
```json
{
  "id": "usr_1234567890",
  "username": "username",
  "email": "user@example.com",
  "role": "user",
  "nickname": "张三",
  "avatar": "https://example.com/avatars/user.jpg",
  "region": "北京",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

---

## PUT /api/users/{username}

**功能描述**：更新用户信息

**请求参数**：
- `username`：用户名（路径参数）
- 请求体：
  - `email`：邮箱（可选）
  - `password`：密码（可选）
  - `nickname`：昵称（可选）
  - `avatar`：头像 URL（可选）
  - `region`：地区（可选）

**请求体示例**：
```json
{
  "email": "updated@example.com",
  "nickname": "新昵称",
  "avatar": "https://example.com/avatars/new.jpg",
  "region": "深圳"
}
```

**响应格式**：
```json
{
  "message": "User updated successfully",
  "username": "username"
}
```

---

## DELETE /api/users/{username}

**功能描述**：删除用户

**请求参数**：
- `username`：用户名（路径参数）

**响应格式**：
```json
{
  "message": "User deleted successfully",
  "username": "username"
}
```

---

## GET /api/users/{username}/configs

**功能描述**：获取用户配置信息

**请求参数**：
- `username`：用户名（路径参数）

**响应格式**：
```json
{
  "configs": "{\"theme\":\"dark\",\"language\":\"zh-CN\",\"notifications\":true}"
}
```

---

## PUT /api/users/{username}/configs

**功能描述**：更新用户配置信息

**请求参数**：
- `username`：用户名（路径参数）
- 请求体：
  - `configs`：配置信息（JSON 字符串，可以存储大型配置）

**请求体示例**：
```json
{
  "configs": "{\"theme\":\"dark\",\"language\":\"zh-CN\",\"notifications\":true,\"customSettings\":{\"fontSize\":14,\"sidebar\":\"collapsed\"}}"
}
```

**响应格式**：
```json
{
  "message": "User configs updated successfully"
}
```

**注意事项**：
- `configs` 字段支持存储大型 JSON 字符串（最大可达 1MB 以上）
- 建议将配置存储为 JSON 格式的字符串
- 配置内容应根据业务需求自行解析和处理

---

## 数据库配置

Vigil 使用 SQLite 数据库存储用户信息，配置文件位于 `conf/config.yaml`。

### 数据库配置项

```yaml
database:
  driver: sqlite      # 数据库驱动，目前支持 sqlite
  path: ./conf/users.db  # 数据库文件路径
```

### 数据库表结构

用户表 `users` 结构如下：

| 字段名 | 类型 | 描述 |
|--------|------|------|
| id | TEXT | 用户唯一标识 |
| username | TEXT | 用户名（唯一） |
| password | TEXT | 密码（bcrypt 加密） |
| email | TEXT | 邮箱地址 |
| role | TEXT | 角色（admin/user） |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |
| last_login_at | DATETIME | 最后登录时间 |
| last_login_ip | TEXT | 最后登录 IP |
| login_count | INTEGER | 登录次数 |
| avatar | TEXT | 头像 URL |
| nickname | TEXT | 昵称 |
| region | TEXT | 地区 |
| configs | TEXT | 用户配置（JSON 字符串） |

### 数据迁移

如果之前使用 JSON 文件存储用户数据（`conf/users.json`），系统会在启动时自动迁移到 SQLite 数据库。

**自动列迁移**：当数据库表结构更新时（如新增 avatar、nickname、region、configs 字段），系统会在启动时自动检测并执行 ALTER TABLE 语句添加新列，无需手动迁移。现有数据不会受到影响。
