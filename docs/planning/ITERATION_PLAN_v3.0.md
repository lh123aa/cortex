# 🧠 Cortex v3.0 付费功能迭代计划

> **当前版本**: v2.4.0 (59 项优化完成，Bug=0)  
> **目标版本**: v3.0 (商业化版本)  
> **定价**: Free ≤1GB / Pro ¥15或$15/月 / Enterprise ¥98或$98/月  
> **市场**: 国内收人民币，国际收美金

---

## 一、迭代总览

| Sprint | 主题 | 内容 | 预计 |
|:------:|:-----|:-----|:----:|
| **1** | 付费基础设施 | Tier 字段 + 存储统计 + 用量检查 | 3 天 |
| **2** | License 与支付 | License Key + 微信支付 + Stripe | 5 天 |
| **3** | 企业功能 | RBAC + 审计日志 + SSO | 5 天 |
| **4** | 上线准备 | 文档 + 支付页面 + 发布 | 2 天 |

---

## Sprint 1: 付费基础设施

> 目标：能限制免费用户，能识别付费用户

### 1.1 User 模型增加 Tier 字段

**文件**: `internal/models/user.go`

```go
type User struct {
    ID              string    `json:"id"`
    Username        string    `json:"username"`
    PasswordHash    string    `json:"-"`
    Role            string    `json:"role"`
    Tier            string    `json:"tier"`        // free | pro | enterprise
    StorageUsedBytes int64    `json:"storage_used_bytes"`
    StorageLimitBytes int64   `json:"storage_limit_bytes"`  // 根据 tier 自动计算
    LicenseKey      string    `json:"license_key,omitempty"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    IsActive        bool      `json:"is_active"`
}
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| User 结构体加字段 | `models/user.go` | 10 行 |
| schema.sql users 表加列 | `storage/schema.sql` | 5 行 |
| 数据库迁移 | `storage/sqlite.go` | 15 行 |

### 1.2 存储用量统计

**文件**: `internal/storage/crud.go`

新增方法：
```go
func (s *SQLiteStorage) CalculateStorageUsed(userID string) (int64, error)
```

- 查询该用户所有文档的 `FileSize` 总和
- 缓存结果减少重复计算

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| CalculateStorageUsed | `storage/crud.go` | 20 行 |
| Storage 接口 | `storage/storage.go` | 1 行 |

### 1.3 用量检查中间件

**文件**: `internal/api/rest.go`

在文档索引 API 前插入检查：

```go
func (s *RESTServer) storageLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method != "POST" {
            c.Next()
            return
        }
        userID := GetUserID(c)
        tier := GetUserTier(c)
        if tier == "enterprise" {
            c.Next() // 企业版不限量
            return
        }
        used, _ := s.storage.CalculateStorageUsed(userID)
        limit := getStorageLimit(tier) // free=1GB, pro=unlimited
        if used >= limit {
            c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
                "error": "storage limit exceeded, upgrade at https://cortex.ai/pricing",
                "used_bytes": used,
                "limit_bytes": limit,
            })
            return
        }
        c.Next()
    }
}
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| 中间件 | `internal/api/rest.go` | 30 行 |
| 路由注册 | `internal/api/rest.go` | 3 行 |

### 1.4 CLI 查看用量

```bash
cortex usage
# 输出:
# Storage: 234 MB / 1 GB (23%)
# Tier:    Free
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| usage 命令 | `cmd/cortex/main.go` | 30 行 |

---

## Sprint 2: License 与支付

> 目标：用户可以付费升级

### 2.1 License Key 系统

| 功能 | 说明 |
|:-----|:------|
| 生成 Key | 服务端生成签名 License Key |
| 激活 Key | 用户输入 Key → 验证 → 升级 Tier |
| 吊销 Key | 管理端吊销 License |
| 校验 Key | 启动时校验，过期自动降级 |

```go
type License struct {
    ID        string    `json:"id"`
    Key       string    `json:"key"`
    Tier      string    `json:"tier"`
    UserID    string    `json:"user_id"`
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
    Active    bool      `json:"active"`
}
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| License 模型 | `internal/models/license.go` | 20 行 |
| License 存储 | `internal/storage/license.go` | 60 行 |
| 激活 API | `internal/api/license.go` | 40 行 |
| License 验证 | `internal/auth/license.go` | 30 行 |

### 2.2 支付集成 - 国内 (微信/支付宝)

| 功能 | 说明 |
|:-----|:------|
| 下单 API | `POST /api/payment/create` 创建订单 |
| 支付回调 | 微信/支付宝异步通知处理 |
| 发货 | 支付成功 → 自动发放 License Key |
| 续费提醒 | 到期前 7 天邮件/站内通知 |

**可选方案**:
- 使用 [Lemon Squeezy](https://lemonsqueezy.com)（同时支持信用卡+支付宝+微信，无需自己对接支付牌照）
- 直接对接微信支付/支付宝（需企业资质）

| 方案 | 资质要求 | 手续费 | 开发量 |
|:-----|:---------|:------:|:------:|
| Lemon Squeezy | 无 | 5% | 3 天 |
| 微信支付直连 | 企业营业执照 | 0.6% | 7 天 |
| 支付宝直连 | 企业营业执照 | 0.6% | 7 天 |

### 2.3 支付集成 - 国际 (Stripe)

| 功能 | 说明 |
|:-----|:------|
| Stripe Checkout | 预构建支付页面 |
| Webhook | 支付成功/失败/续费通知 |
| Subscription | 按月自动扣费 |
| Customer Portal | 用户自助管理订阅 |

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| Stripe API 集成 | `internal/billing/stripe.go` | 100 行 |
| Webhook 处理 | `internal/billing/webhook.go` | 60 行 |
| 前端支付页 | `internal/api/admin.html` | 50 行 |

---

## Sprint 3: 企业功能

> 目标：企业客户愿意付费的功能

### 3.1 RBAC 权限系统

| 角色 | 权限 |
|:-----|:------|
| **admin** | 全部权限：管理用户、管理知识库、查看审计日志 |
| **editor** | 读写知识库、管理自己的记忆 |
| **viewer** | 只读知识库、搜索、RAG 查询 |

```go
type Permission string

const (
    PermRead        Permission = "read"
    PermWrite       Permission = "write"
    PermAdmin       Permission = "admin"
    PermManageUsers Permission = "manage_users"
    PermViewAudit   Permission = "view_audit"
)

type Role struct {
    Name        string       `json:"name"`
    Permissions []Permission `json:"permissions"`
}

var DefaultRoles = map[string][]Permission{
    "admin":   {PermRead, PermWrite, PermAdmin, PermManageUsers, PermViewAudit},
    "editor":  {PermRead, PermWrite},
    "viewer":  {PermRead},
}
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| RBAC 模型 | `internal/models/rbac.go` | 40 行 |
| 权限中间件 | `internal/api/rbac.go` | 30 行 |
| 角色管理 API | `internal/api/rbac_handler.go` | 80 行 |

### 3.2 审计日志

| 事件类型 | 记录内容 |
|:---------|:---------|
| search | 谁搜索了什么、结果数 |
| index | 谁索引了什么文件 |
| memory_write | 谁写入了什么记忆 |
| memory_delete | 谁删除了什么记忆 |
| login | 登录时间/IP |
| tier_change | 套餐变更记录 |

```go
type AuditLog struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    Action    string    `json:"action"`
    Resource  string    `json:"resource"`
    Detail    string    `json:"detail"`
    IP        string    `json:"ip"`
    CreatedAt time.Time `json:"created_at"`
}
```

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| 审计日志模型 | `internal/models/audit.go` | 15 行 |
| 审计日志存储 | `internal/storage/audit.go` | 50 行 |
| 审计日志中间件 | `internal/api/audit.go` | 30 行 |
| 审计查询 API | `internal/api/audit_handler.go` | 40 行 |

### 3.3 SSO/OAuth 集成

| Provider | 协议 |
|:---------|:-----|
| Google | OAuth 2.0 |
| GitHub | OAuth 2.0 |
| 企业 LDAP | LDAP(S) |
| 通用 OIDC | OpenID Connect |

| 改动 | 位置 | 工作量 |
|:-----|:------|:------:|
| OAuth 路由 | `internal/api/oauth.go` | 80 行 |
| LDAP 集成 | `internal/auth/ldap.go` | 60 行 |
| 用户关联 | `internal/auth/sso.go` | 40 行 |

---

## Sprint 4: 上线准备

> 目标：用户可以自助购买和使用

### 4.1 定价页面

| 页面 | 内容 |
|:-----|:------|
| `/pricing` | 套餐对比、下单按钮 |
| `/billing` | 当前套餐、用量、历史订单 |
| `/upgrade` | 升级确认、支付页面 |

### 4.2 文档更新

| 文件 | 更新内容 |
|:-----|:---------|
| `README.md` | 添加定价信息、套餐对比标签 |
| `USAGE_GUIDE.md` | 添加升级流程、License 激活说明 |
| `docs/` | 企业功能使用文档 |

### 4.3 发布检查清单

```
[ ] go test ./... 全部通过
[ ] go vet 无问题
[ ] 免费版用量检查正常
[ ] 企业版不限量正常
[ ] 支付流程端到端可用
[ ] License 激活/吊销正常
[ ] License 过期降级正常
[ ] RBAC 三级权限验证通过
[ ] 审计日志记录正确
[ ] SSO 登录正常
[ ] English README 同步更新
```

---

## 二、时间线

```
第1周 ─── Sprint 1: 付费基础设施
  Day 1:  User Tier 字段 +  schema 迁移
  Day 2:  存储用量统计 + 缓存
  Day 3:  用量检查中间件 + CLI usage 命令

第2周 ─── Sprint 2: License 与支付
  Day 1-2: License Key 系统
  Day 3:   微信/支付宝集成 (或 Lemon Squeezy)
  Day 4-5: Stripe 集成

第3-4周 ── Sprint 3: 企业功能
  Day 1-2: RBAC 权限系统
  Day 3:   审计日志
  Day 4-5: SSO/OAuth 集成

第5周 ─── Sprint 4: 上线准备
  Day 1:   定价/账单页面
  Day 2:   文档 + 中英文 README
  Day 3:   E2E 测试 + Bug 修复
  Day 4:   发布 v3.0
```

---

## 三、依赖关系

```
Sprint 1 ──── 必须先做，所有其他功能依赖 Tier 判断
  ├── Sprint 2 ─── 依赖用户身份识别 (Tier)
  │   └── Sprint 4 ─── 发布上线
  └── Sprint 3 ─── 依赖 Tier 判断 (仅 enterprise 可用)
      └── Sprint 4
```

---

## 四、风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|:-----|:----:|:----:|:------|
| 支付牌照资质 | 中 | 高 | 先用 Lemon Squeezy 替代，0 资质要求 |
| 用户反感付费墙 | 中 | 中 | 1GB 免费全功能，绝大多数人不受影响 |
| Stripe + 微信双维护 | 低 | 中 | 抽象 BillingProvider 接口，统一发货逻辑 |
| License 被破解 | 中 | 低 | 云端校验 + 定期检查，不过度投入 DRM |

---

## 五、总结

```
v3.0 = v2.4 + 3 个 Sprint + 1 个发布 Sprint
     = 59 项优化 + 15 项付费功能
     = Bug=0 的基础 + 可持续的商业化

总工期: 4-5 周 (单人)
关键里程碑: Sprint 1 做完就能限制免费版
            Sprint 2 做完就能收钱
            Sprint 3 做完企业客户就能用
```
