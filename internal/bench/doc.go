package bench

import "time"

// jsonTagsConvention 记录基准 JSON 输出的字段命名约定（仅文档用途）：
//   - 字段名使用 snake_case，便于脚本消费
//   - 时间统一 RFC3339 字符串（generated_at）
//   - 延迟统一毫秒浮点（*_ms）
//   - 比率统一 0-1 浮点（recall_* / mrr / containment_at_10）
var _ = time.Second
