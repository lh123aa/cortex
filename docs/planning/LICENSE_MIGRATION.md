# 许可证迁移记录：AGPL-3.0 → Apache-2.0

> 决策日期：2026-10-30
> 决策依据：[系统性评估与开源/商业化方案 §4.4](./OPEN_CORE_AND_MONETIZATION_STRATEGY.md)
> 结论：**方案 A —— 核心 Apache-2.0，企业版商业许可（闭源、不入本仓库）**

---

## 1. 迁移前状态

| 项 | 值 |
|---|---|
| `LICENSE` 文件 | GNU AGPL-3.0 |
| `README` 声明 | 曾长期写 “MIT”，后修正为 AGPL-3.0 |
| 影响 | 企业法务会直接否决；与「开源核心 + 商业企业版」的目标冲突 |

## 2. 迁移决策

选择 **Apache-2.0** 而不是继续 AGPL 或退回 MIT，理由：

1. **企业友好**：明确授予专利许可（MIT 没有），企业法务接受度高；无「网络服务必须开源」义务（AGPL 有）。
2. **允许闭源企业版共存**：企业版通过 `pkg/ext` 接口与核心交互，属于独立程序，不构成衍生作品，法律边界清晰。
3. **生态兼容**：与 XERJ（Apache-2.0）、Dify（Apache-2.0）等主流同类项目一致，降低采用摩擦。
4. **商标保留**：Apache-2.0 第 6 条明确不授予商标权，名称与 Logo 仍受保护。

**接受的代价**：无法阻止大厂直接白嫖核心代码。缓解手段是持续迭代速度、企业功能、支持服务与商标，而不是靠许可证限制。

## 3. 版权人确认

| 提交身份 | 提交数 | 说明 |
|---|---:|---|
| `Cortex <cortex@example.com>` | 114 | 本地占位身份（仓库所有者的开发机配置） |
| `lh123aa <130943782+lh123aa@users.noreply.github.com>` | 2 | 仓库所有者 |

**结论**：历史提交均来自仓库所有者本人（或本地占位身份），不存在第三方版权人，迁移无需外部同意。

> ⚠️ 如果后续发现有外部贡献者参与过历史提交，需要补签贡献者许可协议（CLA），见 §5。

## 4. 执行清单

- [x] `LICENSE` 替换为 Apache License 2.0 全文（含 Appendix 版权行）
- [x] 新增 `NOTICE`（项目版权、第三方依赖许可、商标声明）
- [x] `README.md`：徽章、对比表、许可证章节、页脚共 8 处
- [x] `README.en.md`：徽章、许可证章节、页脚共 4 处
- [x] `docs/index.html`（营销落地页）：描述、文案、徽章共 6 处
- [x] `docs/assets/video/source/index.html`（演示视频源）：1 处
- [x] `COMMERCIAL_PLAN.md`：3 处
- [x] `docs/promotion/*.md`（9 篇宣传稿）：10 处
- [x] `docs/planning/OPEN_CORE_AND_MONETIZATION_STRATEGY.md`：决策章节更新为「已执行」
- [x] 扫描跟踪文件中的明文密钥：无命中

合计替换 **33 处**许可声明。

## 5. 后续贡献者政策（建议立即启用）

Apache-2.0 第 5 条规定：未明确声明的贡献默认按本许可授权。这对单项目足够，但**不便于未来双许可或企业版合并**。建议：

1. **DCO（Developer Certificate of Origin）**：提交时 `git commit -s`，声明有权贡献。轻量，适合开源社区。
2. **CLA（Contributor License Agreement）**：允许项目方在 Apache-2.0 之外以其他条款（含商业许可）再许可同一份贡献。若未来要把社区贡献并入企业版，CLA 是必需的。

建议：**默认 DCO + 对企业版相关目录要求 CLA**，并在 `CONTRIBUTING.md` 写明。

## 6. 企业版许可边界

| | 开源核心（本仓库） | 企业版（私有仓库） |
|---|---|---|
| 许可 | Apache-2.0 | 商业许可（订阅） |
| 代码可见性 | 公开 | 闭源 |
| 修改与分发 | 允许（含闭源产品），保留 NOTICE | 按合同 |
| 商标使用 | 仅合理引用 | 按合同 |
| 支持 | 社区 | SLA |

技术边界：企业版通过 `pkg/ext` 注册扩展（`Extension` 接口），核心在启动时按 License 的 `features` 决定是否启用。核心代码不包含任何闭源实现。

## 7. 回滚方案

若迁移后发现法律问题或需要恢复 AGPL：

```bash
git revert <本次迁移提交>
# 或
git checkout <迁移前提交> -- LICENSE README.md README.en.md docs/index.html
```

但注意：已按 Apache-2.0 发布的版本无法收回，下游可继续按 Apache-2.0 使用该版本。因此**迁移应在首次公开发布之前完成**。
