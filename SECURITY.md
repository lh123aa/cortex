# 安全策略

## 报告安全漏洞

如果你发现了安全漏洞，**请不要公开提 Issue**。

请直接发送邮件至项目维护者，或通过 GitHub 的安全 advisory 功能私下报告：

1. 访问 https://github.com/lh123aa/cortex/security/advisories
2. 点击 "New draft security advisory"
3. 填写漏洞详情

我们会在 **48 小时内**确认收到，并尽快修复。

## 受支持的版本

| 版本 | 支持状态 |
|------|---------|
| v2.x | ✅ 积极维护 |
| v1.x | ❌ 不再支持 |

## 安全最佳实践

- 始终使用最新版本
- 生产环境建议启用认证 (`auth_enabled: true`)
- 不要将 API 密钥硬编码在代码中
- 使用环境变量或配置文件管理敏感信息
