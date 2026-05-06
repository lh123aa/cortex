package models

import "time"

// License 许可证
type License struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Tier      string    `json:"tier"` // pro | enterprise
	UserID    string    `json:"user_id,omitempty"`
	MaxUsers  int       `json:"max_users"`  // 企业版允许的用户数
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// ActivateRequest 激活 License 请求
type ActivateLicenseRequest struct {
	Key      string `json:"key" binding:"required"`
	Metadata string `json:"metadata,omitempty"` // 设备/环境信息
}

// ActivateLicenseResponse 激活结果
type ActivateLicenseResponse struct {
	Success  bool   `json:"success"`
	Tier     string `json:"tier,omitempty"`
	Message  string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
}
