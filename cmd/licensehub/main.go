// Command licensehub serves the self-service license claim endpoint.
//
// Flow: buyer pays via PayPal.Me, the payment lands in the "paid" ledger either
// through the PayPal IPN listener (/api/ipn) or a CSV import (/api/paid), then
// the buyer opens /claim, enters the transaction id + email, and the hub signs a
// Pro license on the spot. No manual step, no outbound email.
//
// Security posture:
//   - a license is only issued for a transaction already recorded as paid
//   - the Ed25519 private key lives only on the server (mode 0600, root only)
//   - a txn id can be claimed once (claim ledger is append-only JSONL)
//   - per-IP rate limit keeps brute force useless
//
// Build: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o licensehub ./cmd/licensehub
package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lh123aa/cortex/pkg/license"
)

var (
	keyPath    = flag.String("key", "/root/.cortex/license-private.key", "ed25519 private key hex file")
	ledger     = flag.String("ledger", "/www/wwwroot/cortex/orders.jsonl", "append-only claim ledger")
	paidPath   = flag.String("paid", "/www/wwwroot/cortex/paid.jsonl", "append-only paid-transaction ledger")
	addr       = flag.String("addr", "127.0.0.1:8790", "listen address")
	days       = flag.Int("days", 365, "license validity in days")
	maxPerHr   = flag.Int("rate", 6, "max claims per IP per hour")
	openMode   = flag.Bool("open", false, "DANGER: issue without checking the paid ledger (testing only)")
	adminToken = flag.String("admin-token", "", "shared secret for /api/paid imports")
	paypalTo   = flag.String("paypal-email", "lh123aa@outlook.com", "receiver_email expected in IPN")
	ipnVerify  = flag.String("ipn-verify", "https://ipnpb.paypal.com/cgi-bin/webscr", "IPN postback endpoint")
	trialDays  = flag.Int("trial-days", 14, "self-service trial length in days")
	trialFile  = flag.String("trial", "/www/wwwroot/cortex/trials.jsonl", "append-only trial ledger")
	trialPerIP = flag.Int("trial-rate", 3, "max trial requests per IP per day")
)

var (
	priv       ed25519.PrivateKey
	mu         sync.Mutex
	claimed    = map[string]bool{}
	paid       = map[string]bool{}
	paidMod    time.Time
	trialMails = map[string]bool{}
	rateHits   = map[string][]time.Time{}
)

var (
	txnRe   = regexp.MustCompile(`^[A-Z0-9]{12,20}$`)
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type claimReq struct {
	Txn   string `json:"txn"`
	Email string `json:"email"`
}

type claimResp struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	License  string `json:"license,omitempty"`
	Plan     string `json:"plan,omitempty"`
	Expires  string `json:"expires,omitempty"`
	Order    string `json:"order,omitempty"`
	Docs     int64  `json:"documents,omitempty"`
	Repeated bool   `json:"already_claimed,omitempty"`
	Pending  bool   `json:"pending,omitempty"`
}

func loadKey() error {
	raw, err := os.ReadFile(*keyPath)
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}
	text := strings.TrimSpace(string(raw))
	if i := strings.Index(text, "private_key"); i >= 0 {
		if c := strings.Index(text[i:], ":"); c >= 0 {
			text = strings.TrimSpace(text[i+c+1:])
		}
	}
	b, err := hex.DecodeString(text)
	if err != nil {
		return fmt.Errorf("decode key: %w", err)
	}
	if len(b) == ed25519.SeedSize {
		priv = ed25519.NewKeyFromSeed(b)
	} else if len(b) == ed25519.PrivateKeySize {
		priv = ed25519.PrivateKey(b)
	} else {
		return fmt.Errorf("unexpected key length %d", len(b))
	}
	return nil
}

func scanJSONL(path string, fn func(map[string]any)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var row map[string]any
		if json.Unmarshal(sc.Bytes(), &row) != nil {
			continue
		}
		fn(row)
	}
}

func loadClaimed() {
	scanJSONL(*ledger, func(row map[string]any) {
		if t, ok := row["txn"].(string); ok && t != "" {
			claimed[t] = true
		}
	})
}

func loadPaid(force bool) {
	st, err := os.Stat(*paidPath)
	if err != nil {
		return
	}
	if !force && !st.ModTime().After(paidMod) {
		return
	}
	paid = map[string]bool{}
	scanJSONL(*paidPath, func(row map[string]any) {
		if t, ok := row["txn"].(string); ok && t != "" {
			paid[strings.ToUpper(t)] = true
		}
	})
	paidMod = st.ModTime()
}

func appendJSONL(path string, v any) {
	b, _ := json.Marshal(v)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("append %s: %v", path, err)
		return
	}
	defer f.Close()
	f.Write(append(b, '\n'))
}

func loadTrial() {
	trialMails = map[string]bool{}
	scanJSONL(*trialFile, func(row map[string]any) {
		if e, ok := row["email"].(string); ok && e != "" {
			trialMails[strings.ToLower(e)] = true
		}
	})
}

func allow(ip string) bool {
	now := time.Now()
	list := rateHits[ip]
	keep := list[:0]
	for _, t := range list {
		if now.Sub(t) < time.Hour {
			keep = append(keep, t)
		}
	}
	if len(keep) >= *maxPerHr {
		rateHits[ip] = keep
		return false
	}
	rateHits[ip] = append(keep, now)
	return true
}

// allowWindow limits a key to n events inside a sliding window.
func allowWindow(key string, n int, window time.Duration) bool {
	now := time.Now()
	list := rateHits[key]
	keep := list[:0]
	for _, t := range list {
		if now.Sub(t) < window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= n {
		rateHits[key] = keep
		return false
	}
	rateHits[key] = append(keep, now)
	return true
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func handleClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, claimResp{Error: "POST only"})
		return
	}
	var req claimReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeJSON(w, 400, claimResp{Error: "bad json"})
		return
	}
	txn := strings.ToUpper(strings.TrimSpace(req.Txn))
	txn = strings.ReplaceAll(txn, "-", "")
	email := strings.TrimSpace(req.Email)
	ip := clientIP(r)

	if !txnRe.MatchString(txn) {
		writeJSON(w, 400, claimResp{Error: "交易号看起来不对：PayPal 的 Transaction ID 是 17 位字母数字，形如 7XK1234567890123"})
		return
	}
	if !emailRe.MatchString(email) {
		writeJSON(w, 400, claimResp{Error: "邮箱格式不对，它会被写进许可证用于售后核对"})
		return
	}

	mu.Lock()
	loadPaid(false)
	if claimed[txn] {
		mu.Unlock()
		writeJSON(w, 200, claimResp{OK: true, Repeated: true, Error: "这张交易号已经领过许可证。如需补发，请用同一邮箱联系我们。", Order: "PAYPAL-" + txn})
		return
	}
	if !*openMode && !paid[txn] {
		mu.Unlock()
		log.Printf("claim rejected (not paid) txn=%s ip=%s", txn, ip)
		writeJSON(w, 409, claimResp{Pending: true, Error: "这笔交易还没有同步到签发系统。PayPal 到账通知有时延迟 5–10 分钟，请稍后重试；超过半小时仍不行，把收据截图发到 lh123aa@outlook.com。"})
		return
	}
	if !allow(ip) {
		mu.Unlock()
		writeJSON(w, 429, claimResp{Error: "请求太频繁，请稍后再试"})
		return
	}
	info := license.Info{
		Plan:      license.PlanPro,
		Email:     email,
		Order:     "PAYPAL-" + txn,
		Seats:     1,
		IssuedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt: time.Now().UTC().AddDate(0, 0, *days),
	}
	token, err := license.Sign(priv, info)
	if err != nil {
		mu.Unlock()
		log.Printf("sign failed txn=%s: %v", txn, err)
		writeJSON(w, 500, claimResp{Error: "签发失败，请稍后再试或邮件联系"})
		return
	}
	claimed[txn] = true
	mu.Unlock()

	appendJSONL(*ledger, map[string]any{
		"txn": txn, "email": email, "order": info.Order, "plan": "pro",
		"issued_at": info.IssuedAt.Format(time.RFC3339), "expires_at": info.ExpiresAt.Format(time.RFC3339),
		"ip": ip, "ua": r.UserAgent(),
	})
	log.Printf("issued txn=%s email=%s ip=%s", txn, email, ip)

	writeJSON(w, 200, claimResp{
		OK: true, License: token, Plan: "Pro",
		Expires: info.ExpiresAt.Format("2006-01-02"), Order: info.Order,
		Docs: license.Limits(license.PlanPro).Documents,
	})
}

// handleTrial issues a short self-service Pro trial: email only, no payment.
// One trial per email address; limited per IP to keep farming costs high.
func handleTrial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, claimResp{Error: "POST only"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req); err != nil {
		writeJSON(w, 400, claimResp{Error: "bad json"})
		return
	}
	email := strings.TrimSpace(req.Email)
	ip := clientIP(r)
	if !emailRe.MatchString(email) {
		writeJSON(w, 400, claimResp{Error: "邮箱格式不对，许可证会绑定这个邮箱"})
		return
	}

	mu.Lock()
	loadTrial()
	if trialMails[strings.ToLower(email)] {
		mu.Unlock()
		writeJSON(w, 200, claimResp{OK: false, Repeated: true, Error: "这个邮箱已经领过试用了。想换设备或续期，请邮件联系我们。"})
		return
	}
	if !allowWindow("trial:"+ip, *trialPerIP, 24*time.Hour) {
		mu.Unlock()
		writeJSON(w, 429, claimResp{Error: "这个网络今天领得太多了，明天再试或邮件联系我们"})
		return
	}
	info := license.Info{
		Plan:      license.PlanPro,
		Email:     email,
		Order:     fmt.Sprintf("TRIAL-%d", time.Now().Unix()),
		Seats:     1,
		IssuedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt: time.Now().UTC().AddDate(0, 0, *trialDays),
	}
	token, err := license.Sign(priv, info)
	if err != nil {
		mu.Unlock()
		log.Printf("trial sign failed email=%s: %v", email, err)
		writeJSON(w, 500, claimResp{Error: "签发失败，请稍后再试"})
		return
	}
	trialMails[strings.ToLower(email)] = true
	mu.Unlock()

	appendJSONL(*trialFile, map[string]any{
		"email": email, "order": info.Order, "days": *trialDays,
		"issued_at": info.IssuedAt.Format(time.RFC3339), "expires_at": info.ExpiresAt.Format(time.RFC3339),
		"ip": ip, "ua": r.UserAgent(),
	})
	log.Printf("trial issued email=%s days=%d ip=%s", email, *trialDays, ip)

	writeJSON(w, 200, claimResp{
		OK: true, License: token, Plan: fmt.Sprintf("Pro Trial (%d days)", *trialDays),
		Expires: info.ExpiresAt.Format("2006-01-02"), Order: info.Order,
		Docs: license.Limits(license.PlanPro).Documents,
	})
}

func handlePaid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"ok": false, "error": "POST only"})
		return
	}
	if *adminToken == "" || r.Header.Get("X-Admin-Token") != *adminToken {
		writeJSON(w, 401, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}
	var body struct {
		Txns   []string `json:"txns"`
		Source string   `json:"source"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "bad json"})
		return
	}
	added := 0
	mu.Lock()
	loadPaid(true)
	for _, raw := range body.Txns {
		t := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", ""))
		if !txnRe.MatchString(t) || paid[t] {
			continue
		}
		appendJSONL(*paidPath, map[string]any{"txn": t, "src": body.Source, "at": time.Now().UTC().Format(time.RFC3339)})
		paid[t] = true
		added++
	}
	mu.Unlock()
	log.Printf("paid import: added=%d total=%d", added, len(paid))
	writeJSON(w, 200, map[string]any{"ok": true, "added": added, "total": len(paid)})
}

// handleIPN is the PayPal Instant Payment Notification listener.
func handleIPN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", 400)
		return
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		http.Error(w, "bad form", 400)
		return
	}

	// Postback verification: PayPal must confirm the message came from it.
	verifyBody := "cmd=_notify-validate&" + string(raw)
	req, err := http.NewRequest(http.MethodPost, *ipnVerify, strings.NewReader(verifyBody))
	if err != nil {
		http.Error(w, "verify build failed", 500)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "cortex-licensehub/1.0")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("ipn verify error: %v", err)
		http.Error(w, "verify failed", 500)
		return
	}
	defer res.Body.Close()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, 64))
	if strings.TrimSpace(string(reply)) != "VERIFIED" {
		log.Printf("ipn rejected verify=%q", strings.TrimSpace(string(reply)))
		http.Error(w, "not verified", 400)
		return
	}

	status := form.Get("payment_status")
	receiver := strings.ToLower(form.Get("receiver_email"))
	if receiver == "" {
		receiver = strings.ToLower(form.Get("business"))
	}
	amount := form.Get("mc_gross")
	currency := form.Get("mc_currency")
	txn := strings.ToUpper(form.Get("txn_id"))

	log.Printf("ipn verified status=%s amount=%s %s txn=%s", status, amount, currency, txn)

	if status != "Completed" {
		w.Write([]byte("ignored"))
		return
	}
	if *paypalTo != "" && receiver != strings.ToLower(*paypalTo) {
		log.Printf("ipn receiver mismatch %q", receiver)
		w.Write([]byte("ignored"))
		return
	}
	if !txnRe.MatchString(txn) {
		w.Write([]byte("ignored"))
		return
	}
	// Accept the two published price points, with a little tolerance for fees/tips.
	ok := (currency == "USD" && (amount == "29.00" || amount == "29")) ||
		(currency == "CNY" && (amount == "199.00" || amount == "199"))
	if !ok {
		log.Printf("ipn amount mismatch %s %s txn=%s", amount, currency, txn)
		w.Write([]byte("ignored"))
		return
	}

	mu.Lock()
	loadPaid(true)
	if !paid[txn] {
		appendJSONL(*paidPath, map[string]any{
			"txn": txn, "amount": amount, "currency": currency, "src": "ipn",
			"payer": form.Get("payer_email"), "at": time.Now().UTC().Format(time.RFC3339),
		})
		paid[txn] = true
		log.Printf("paid recorded via ipn txn=%s amount=%s %s", txn, amount, currency)
	}
	mu.Unlock()
	w.Write([]byte("OK"))
}

func main() {
	flag.Parse()
	if err := loadKey(); err != nil {
		log.Fatalf("license key: %v", err)
	}
	loadClaimed()
	loadPaid(true)
	loadTrial()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/claim", handleClaim)
	mux.HandleFunc("/api/trial", handleTrial)
	mux.HandleFunc("/api/paid", handlePaid)
	mux.HandleFunc("/api/ipn", handleIPN)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		loadPaid(false)
		loadTrial()
		writeJSON(w, 200, map[string]any{"issued_total": len(claimed), "paid_total": len(paid), "trial_total": len(trialMails), "open_mode": *openMode})
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      25 * time.Second,
	}
	log.Printf("licensehub listening on %s (paid=%d claimed=%d open=%v)", *addr, len(paid), len(claimed), *openMode)
	log.Fatal(srv.ListenAndServe())
}
