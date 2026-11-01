// Package license implements Cortex's offline, Ed25519-signed license v1 format.
//
// A license is a self-contained text token:
//
//	base64url(JSON payload) "." base64url(Ed25519 signature)
//
// The payload is signed by Cortex's private key and verified locally with the
// embedded public key, so no network connection or license server is required.
// Only the public key ships inside the binary; the private key lives offline
// (see cmd/licensegen) and is never embedded.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FormatVersion is the current license payload version ("v" field).
const FormatVersion = 1

// EnvLicensePath overrides the default license file location. It is primarily
// useful for tests and portable/CI installations.
const EnvLicensePath = "CORTEX_LICENSE_PATH"

// DefaultPricingURL is where users are pointed when they hit a quota or want to
// upgrade their plan.
const DefaultPricingURL = "https://github.com/lh123aa/cortex#pricing"

// PublicKeyHex is the hex-encoded Ed25519 public key used to verify licenses.
//
// It can be overridden at build time for custom/private deployments:
//
//	go build -ldflags "-X github.com/lh123aa/cortex/pkg/license.PublicKeyHex=<64 hex chars>" ./cmd/cortex
var PublicKeyHex = "2619a532e715e2b3a79c38ab3c129d35ef9a5133886b991a64cc16619fdd159f"

// Plan is a commercial license tier.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanTrial      Plan = "trial"
	PlanPro        Plan = "pro"
	PlanTeam       Plan = "team"
	PlanEnterprise Plan = "enterprise"
)

// Valid reports whether the plan is a known tier.
func (p Plan) Valid() bool {
	switch p {
	case PlanFree, PlanTrial, PlanPro, PlanTeam, PlanEnterprise:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (p Plan) String() string { return string(p) }

// Info is a verified license together with the raw key that produced it.
type Info struct {
	Plan      Plan      `json:"plan"`
	Features  []string  `json:"features,omitempty"`
	Email     string    `json:"email,omitempty"`
	Order     string    `json:"order,omitempty"`
	Seats     int       `json:"seats,omitempty"`
	IssuedAt  time.Time `json:"issued_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"` // zero value = perpetual
	Key       string    `json:"key,omitempty"`
}

// Perpetual reports whether the license never expires.
func (i *Info) Perpetual() bool { return i == nil || i.ExpiresAt.IsZero() }

// HasFeature reports whether the license grants a named feature.
//
// feature 是闭源企业版用来判断「能否注册某个扩展」的唯一依据，
// 例如 audit-log、sso-oidc、rbac-team、es-compat。
// 规则：free 永远没有 feature；plan 隐含其基础 feature（见 PlanFeatures）；
// 另外 license payload 里显式列出的 feature 追加生效。
func (i *Info) HasFeature(feature string) bool {
	if i == nil || feature == "" {
		return false
	}
	for _, f := range PlanFeatures(i.Plan) {
		if f == feature || f == "*" {
			return true
		}
	}
	for _, f := range i.Features {
		if f == feature || f == "*" {
			return true
		}
	}
	return false
}

// PlanFeatures 返回套餐默认包含的 feature 集合。
// "*" 表示全部开放（Enterprise）。
func PlanFeatures(plan Plan) []string {
	switch plan {
	case PlanEnterprise:
		return []string{"*"}
	case PlanTeam:
		return []string{"console", "audit-log", "rbac-team", "sso-oidc", "backup-s3", "rerank"}
	case PlanPro:
		return []string{"console", "rerank", "auto-index"}
	case PlanTrial:
		// 试用期开放 Pro 能力，便于转化
		return []string{"console", "rerank", "auto-index"}
	default:
		return nil
	}
}

// DaysLeft 返回剩余天数；永久或已过期分别返回 -1 / 0。
func (i *Info) DaysLeft() int {
	if i == nil || i.ExpiresAt.IsZero() {
		return -1
	}
	d := time.Until(i.ExpiresAt)
	if d <= 0 {
		return 0
	}
	return int(d.Hours() / 24)
}

// IsTrial 是否试用许可。
func (i *Info) IsTrial() bool { return i != nil && i.Plan == PlanTrial }

// PlanLimits describes the quota attached to a plan. A value of -1 means
// unlimited.
type PlanLimits struct {
	Documents    int64
	StorageBytes int64
}

const gib = int64(1) << 30

// Limits returns the document and storage quotas for a plan. Unknown plans get
// the conservative Free quotas.
func Limits(plan Plan) PlanLimits {
	switch plan {
	case PlanTrial, PlanPro:
		// 试用期按 Pro 配额：试用就该体验付费能力，而不是被 5k 文档上限卡住。
		return PlanLimits{Documents: 500_000, StorageBytes: 100 * gib}
	case PlanTeam:
		return PlanLimits{Documents: 1_000_000, StorageBytes: 500 * gib}
	case PlanEnterprise:
		return PlanLimits{Documents: -1, StorageBytes: -1}
	default: // PlanFree and anything unknown
		return PlanLimits{Documents: 5_000, StorageBytes: 1 * gib}
	}
}

// QuotaError is returned by CheckQuota when a plan limit would be exceeded.
type QuotaError struct {
	Plan     Plan
	Resource string // "documents" or "storage"
	Used     int64
	Limit    int64
}

func (e *QuotaError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s limit reached (%s used of %s allowed)", e.ResourceLabel(), formatQuotaValue(e.Resource, e.Used), formatQuotaValue(e.Resource, e.Limit))
	switch e.Plan {
	case PlanPro:
		fmt.Fprintf(&b, ". Upgrade to Team: %s", DefaultPricingURL)
	case PlanTeam:
		fmt.Fprintf(&b, ". Upgrade to Enterprise: %s", DefaultPricingURL)
	default:
		fmt.Fprintf(&b, ". Upgrade to Pro: %s", DefaultPricingURL)
	}
	return b.String()
}

func (e *QuotaError) ResourceLabel() string {
	if e.Resource == "storage" {
		return "storage"
	}
	return "document"
}

func formatQuotaValue(resource string, v int64) string {
	if resource == "storage" {
		return formatBytes(v)
	}
	return fmt.Sprintf("%d documents", v)
}

// IsQuotaExceeded reports whether err is a quota violation.
func IsQuotaExceeded(err error) bool {
	var qe *QuotaError
	return errors.As(err, &qe)
}

// CheckQuota verifies that the current document count and storage usage fit
// within the plan's limits. A nil Info is treated as the Free plan.
func CheckQuota(info *Info, documents int, storageBytes int64) error {
	plan := PlanFree
	if info != nil && info.Plan != "" {
		plan = info.Plan
	}
	lim := Limits(plan)
	if lim.Documents >= 0 && int64(documents) >= lim.Documents {
		return &QuotaError{Plan: plan, Resource: "documents", Used: int64(documents), Limit: lim.Documents}
	}
	if lim.StorageBytes >= 0 && storageBytes > lim.StorageBytes {
		return &QuotaError{Plan: plan, Resource: "storage", Used: storageBytes, Limit: lim.StorageBytes}
	}
	return nil
}

// formatBytes renders a byte count in a compact human-readable form.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	switch exp {
	case 0:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(div))
	case 1:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(div))
	case 2:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(div))
	default:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(div))
	}
}

// payload is the signed JSON document. Field names are frozen for v1.
type payload struct {
	Version   int      `json:"v"`
	Plan      string   `json:"plan"`
	Features  []string `json:"features,omitempty"`
	Email     string   `json:"email,omitempty"`
	Order     string   `json:"order,omitempty"`
	Seats     int      `json:"seats"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"` // 0 = perpetual
}

// Path returns the license file location: $CORTEX_LICENSE_PATH when set,
// otherwise ~/.cortex/license.key.
func Path() string {
	if p := strings.TrimSpace(os.Getenv(EnvLicensePath)); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".cortex", "license.key")
	}
	return filepath.Join(home, ".cortex", "license.key")
}

// publicKey decodes and validates the embedded public key.
func publicKey() (ed25519.PublicKey, error) {
	raw := strings.TrimSpace(PublicKeyHex)
	if raw == "" {
		return nil, errors.New("license public key is not configured (build with -ldflags -X github.com/lh123aa/cortex/pkg/license.PublicKeyHex=...)")
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid license public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid license public key length %d, want %d bytes", len(b), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(b), nil
}

// Sign creates a signed license token from info using the given private key.
func Sign(priv ed25519.PrivateKey, info Info) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid Ed25519 private key length %d, want %d bytes", len(priv), ed25519.PrivateKeySize)
	}
	plan := info.Plan
	if plan == "" {
		plan = PlanFree
	}
	if !plan.Valid() {
		return "", fmt.Errorf("invalid plan %q (want free, pro, team or enterprise)", plan)
	}

	issued := info.IssuedAt
	if issued.IsZero() {
		issued = time.Now().UTC()
	}
	var expires int64
	if !info.ExpiresAt.IsZero() {
		expires = info.ExpiresAt.Unix()
		if expires <= issued.Unix() {
			return "", errors.New("expires_at must be after issued_at")
		}
	}

	p := payload{
		Version:   FormatVersion,
		Plan:      string(plan),
		Features:  info.Features,
		Email:     info.Email,
		Order:     info.Order,
		Seats:     info.Seats,
		IssuedAt:  issued.Unix(),
		ExpiresAt: expires,
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to encode license payload: %w", err)
	}
	sig := ed25519.Sign(priv, raw)
	return b64.EncodeToString(raw) + "." + b64.EncodeToString(sig), nil
}

// b64 is the strict raw (unpadded) base64url codec used for license tokens.
// Strict mode rejects non-canonical encodings (non-zero trailing padding bits),
// so flipping even the final character of a payload/signature cannot decode to
// the same bytes.
var b64 = base64.RawURLEncoding.Strict()

// Verify checks a license token's signature, version, plan and expiry.
func Verify(key string) (*Info, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("license key is empty")
	}
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid license format: want base64url(payload).base64url(signature)")
	}
	raw, err := b64.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid license payload encoding: %w", err)
	}
	sig, err := b64.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid license signature encoding: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid license signature length %d, want %d bytes", len(sig), ed25519.SignatureSize)
	}
	pub, err := publicKey()
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(pub, raw, sig) {
		return nil, errors.New("license signature verification failed")
	}

	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("invalid license payload: %w", err)
	}
	if p.Version != FormatVersion {
		return nil, fmt.Errorf("unsupported license version %d (this build supports v%d)", p.Version, FormatVersion)
	}
	plan := Plan(strings.ToLower(strings.TrimSpace(p.Plan)))
	if !plan.Valid() {
		return nil, fmt.Errorf("invalid license plan %q", p.Plan)
	}

	issued := time.Time{}
	if p.IssuedAt != 0 {
		issued = time.Unix(p.IssuedAt, 0).UTC()
	}
	expires := time.Time{}
	if p.ExpiresAt != 0 {
		expires = time.Unix(p.ExpiresAt, 0).UTC()
		if !time.Now().Before(expires) {
			return nil, fmt.Errorf("license expired on %s", expires.Format("2006-01-02"))
		}
	}

	return &Info{
		Plan:      plan,
		Features:  p.Features,
		Email:     p.Email,
		Order:     p.Order,
		Seats:     p.Seats,
		IssuedAt:  issued,
		ExpiresAt: expires,
		Key:       key,
	}, nil
}

// Current loads and verifies the locally installed license. It returns
// (nil, nil) when no license is installed.
func Current() (*Info, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read license file %s: %w", Path(), err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return nil, nil
	}
	return Verify(key)
}

// Activate verifies a license key and writes it to Path().
func Activate(key string) (*Info, error) {
	info, err := Verify(key)
	if err != nil {
		return nil, err
	}
	path := Path()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create license directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(info.Key+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write license file %s: %w", path, err)
	}
	return info, nil
}

// Deactivate removes the locally installed license. It is a no-op when no
// license is installed.
func Deactivate() error {
	if err := os.Remove(Path()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove license file %s: %w", Path(), err)
	}
	return nil
}
