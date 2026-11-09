// Command licensegen is Cortex's offline license issuing tool.
//
// It is used by Cortex maintainers/sales to generate a key pair, issue signed
// license tokens for customers, and verify a token before it is sent out.
// The private key must never be embedded in the Cortex binary or committed to
// the repository.
//
// Usage:
//
//	licensegen keygen [--out-private private.key]
//	licensegen issue --private-key private.key --plan pro --email user@example.com \
//	    --days 365 --seats 1 --out license.key
//	licensegen issue --private-key private.key --plan pro --expires 2027-11-09
//	licensegen verify --key-file license.key
//	licensegen batch --private-key private.key --orders orders.csv --out-dir out/
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lh123aa/cortex/pkg/license"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2:])
	case "issue":
		issue(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	case "batch":
		batch(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `licensegen - Cortex offline license tool

Commands:
  keygen   Generate an Ed25519 key pair (keep the private key offline!)
  issue    Issue a signed license token for one customer
  verify   Verify a license token before sending it to a customer
  batch    Issue many licenses from a CSV order file

Examples:
  licensegen keygen --out-private _temp/license-private.key
  licensegen issue --private-key _temp/license-private.key --plan pro \
      --email user@example.com --days 365 --seats 1 --out _temp/license.key
  licensegen issue --private-key _temp/license-private.key --plan pro \
      --email user@example.com --expires 2027-11-09
  licensegen verify --key-file _temp/license.key
  licensegen batch --private-key _temp/license-private.key --orders orders.csv --out-dir _temp/out

Flags for issue:
  --private-key <path>   private key hex file (required)
  --plan <plan>          free|trial|pro|team|enterprise (default pro)
  --email <email>        customer email
  --order <id>           order / invoice id
  --days <n>             validity in days, 0 = perpetual (default 365)
  --expires <YYYY-MM-DD> explicit expiry date (UTC); overrides --days
  --seats <n>            number of seats (default 1)
  --features <a,b,c>     comma separated feature list
  --out <path>           write license text to a file
  --no-verify            skip the post-issue self-verification
  --quiet                suppress stderr chatter (token still on stdout)
  --public-key <hex>     self-verify against this key instead of the built-in one
                         (key rotation / plan-B testing)

Flags for verify:
  --key <token>          license token text
  --key-file <path>      read the token from a file
  --public-key <hex>     override the public key (default: embedded build key)

Flags for batch (CSV header: email,plan,days,seats,order[,expires][,features]):
  --private-key <path>   private key hex file (required)
  --orders <path>        input CSV (required; "-" reads stdin)
  --out-dir <dir>        directory to write <order>.key files (required)
  --default-days <n>     fallback validity when the row has no days (default 365)
`)
}

func keygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	outPrivate := fs.String("out-private", "", "write the private key hex to this file (keep secret)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal("failed to generate key pair", err)
	}
	privHex := hex.EncodeToString(priv)
	fmt.Printf("private_key: %s\n", privHex)
	fmt.Printf("public_key:  %s\n", hex.EncodeToString(pub))
	if *outPrivate != "" {
		if err := os.WriteFile(*outPrivate, []byte(privHex+"\n"), 0o600); err != nil {
			fatal("failed to write private key file", err)
		}
		fmt.Printf("\nprivate key written to: %s\n", *outPrivate)
		fmt.Println("Keep this file offline. Never commit it or ship it with the binary.")
	}
	fmt.Println("\nEmbed the public key at build time with:")
	fmt.Printf("  go build -ldflags \"-X github.com/lh123aa/cortex/pkg/license.PublicKeyHex=%s\" ./cmd/cortex\n",
		hex.EncodeToString(pub))
}

// issueParams is the full input for "one order -> one license".
type issueParams struct {
	privateKeyPath string
	plan           string
	email          string
	order          string
	days           int
	expires        string
	seats          int
	features       string
	out            string
	noVerify       bool
	quiet          bool
	publicKey      string
}

func issue(args []string) {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	var p issueParams
	fs.StringVar(&p.privateKeyPath, "private-key", "", "path to private key hex file (required)")
	fs.StringVar(&p.plan, "plan", string(license.PlanPro), "free|trial|pro|team|enterprise")
	fs.StringVar(&p.email, "email", "", "customer email")
	fs.StringVar(&p.order, "order", "", "order / invoice id")
	fs.IntVar(&p.days, "days", 365, "validity in days, 0 = perpetual")
	fs.StringVar(&p.expires, "expires", "", "explicit expiry date YYYY-MM-DD (UTC); overrides --days")
	fs.IntVar(&p.seats, "seats", 1, "number of seats")
	fs.StringVar(&p.features, "features", "", "comma separated feature list")
	fs.StringVar(&p.out, "out", "", "write license text to this file")
	fs.BoolVar(&p.noVerify, "no-verify", false, "skip post-issue self-verification")
	fs.BoolVar(&p.quiet, "quiet", false, "suppress human-readable output on stderr (token still on stdout)")
	fs.StringVar(&p.publicKey, "public-key", "",
		"override the public key used for post-issue self-verification (plan-B / rotation testing)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if p.privateKeyPath == "" {
		fmt.Fprintln(os.Stderr, "error: --private-key is required")
		os.Exit(2)
	}

	token, info, err := issueToken(p)
	if err != nil {
		fatal("failed to issue license", err)
	}

	fmt.Println(token)
	if p.out != "" {
		if err := os.WriteFile(p.out, []byte(token+"\n"), 0o644); err != nil {
			fatal("failed to write license file", err)
		}
		if !p.quiet {
			fmt.Fprintf(os.Stderr, "\nlicense written to: %s\n", p.out)
		}
	}
	// Self-verify right after issuing: the most common sales mistake is
	// emailing a customer a license that does not actually verify.
	//
	// --public-key lets key-rotation testing (plan B) self-check against the
	// NEW public key instead of the one compiled into the current binary.
	if p.publicKey != "" {
		license.PublicKeyHex = p.publicKey
	}
	if !p.noVerify {
		if _, err := license.Verify(token); err != nil {
			fmt.Fprintf(os.Stderr, "\nWARNING: the just-issued license FAILED self-verification: %v\n", err)
			fmt.Fprintf(os.Stderr, "    The embedded public key (%s) may not match this private key.\n",
				shortKey(license.PublicKeyHex))
			os.Exit(1)
		}
		if !p.quiet {
			fmt.Fprintf(os.Stderr, "OK self-verified: plan=%s expires=%s\n", info.Plan, expiryLabel(info))
		}
	}
}

// issueToken builds Info and signs it; shared by issue and batch.
func issueToken(p issueParams) (string, *license.Info, error) {
	priv, err := readPrivateKey(p.privateKeyPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load private key: %w", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	info := license.Info{
		Plan:     license.Plan(strings.ToLower(strings.TrimSpace(p.plan))),
		Email:    strings.TrimSpace(p.email),
		Order:    strings.TrimSpace(p.order),
		Seats:    p.seats,
		IssuedAt: now,
		Features: splitFeatures(p.features),
	}
	if !info.Plan.Valid() {
		return "", nil, fmt.Errorf("invalid plan %q (want free, trial, pro, team or enterprise)", p.plan)
	}
	if p.seats < 0 {
		return "", nil, fmt.Errorf("seats must be >= 0")
	}
	if p.days < 0 {
		return "", nil, fmt.Errorf("days must be >= 0")
	}

	switch {
	case strings.TrimSpace(p.expires) != "":
		// Sales thinks in "expires on 2027-11-09", not "365 days".
		// Parse as UTC end-of-day so the customer is not cut off early
		// on their expiry date.
		d, err := time.Parse("2006-01-02", strings.TrimSpace(p.expires))
		if err != nil {
			return "", nil, fmt.Errorf("invalid --expires %q (want YYYY-MM-DD)", p.expires)
		}
		info.ExpiresAt = d.Add(24*time.Hour - time.Second)
		if !info.ExpiresAt.After(now) {
			return "", nil, fmt.Errorf("--expires %s is not in the future", p.expires)
		}
	case p.days > 0:
		info.ExpiresAt = now.AddDate(0, 0, p.days)
	}

	token, err := license.Sign(priv, info)
	if err != nil {
		return "", nil, err
	}
	info.Key = token
	return token, &info, nil
}

func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	key := fs.String("key", "", "license token text")
	keyFile := fs.String("key-file", "", "read the token from a file")
	publicKey := fs.String("public-key", "", "override the public key hex")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	token := strings.TrimSpace(*key)
	if *keyFile != "" {
		raw, err := os.ReadFile(*keyFile)
		if err != nil {
			fatal("failed to read license file", err)
		}
		token = strings.TrimSpace(string(raw))
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: --key or --key-file is required")
		os.Exit(2)
	}
	if *publicKey != "" {
		license.PublicKeyHex = *publicKey
	}

	info, err := license.Verify(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INVALID license: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("License valid")
	fmt.Printf("   plan:    %s\n", info.Plan)
	if info.Email != "" {
		fmt.Printf("   email:   %s\n", info.Email)
	}
	if info.Order != "" {
		fmt.Printf("   order:   %s\n", info.Order)
	}
	if info.Seats > 0 {
		fmt.Printf("   seats:   %d\n", info.Seats)
	}
	fmt.Printf("   issued:  %s\n", info.IssuedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("   expires: %s\n", expiryLabel(info))
	feats := info.Features
	if len(feats) == 0 {
		feats = license.PlanFeatures(info.Plan)
	}
	if len(feats) > 0 {
		fmt.Printf("   feats:   %s\n", strings.Join(feats, ", "))
	}
	lim := license.Limits(info.Plan)
	if lim.Documents < 0 {
		fmt.Println("   limits:  unlimited documents / unlimited storage")
	} else {
		fmt.Printf("   limits:  %d documents / %d bytes storage\n", lim.Documents, lim.StorageBytes)
	}
}

// batch issues licenses from a CSV order file: the smallest useful
// automation for "handle N orders in one sitting".
//
// CSV header: email,plan,days,seats,order[,expires][,features]
// One <order>.key file per row, plus a tab-separated summary on stdout.
func batch(args []string) {
	fs := flag.NewFlagSet("batch", flag.ExitOnError)
	privateKeyPath := fs.String("private-key", "", "path to private key hex file (required)")
	ordersPath := fs.String("orders", "", "input CSV path, \"-\" for stdin (required)")
	outDir := fs.String("out-dir", "", "directory for <order>.key output files (required)")
	defaultDays := fs.Int("default-days", 365, "fallback validity when a row has no days")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *privateKeyPath == "" || *ordersPath == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "error: --private-key, --orders and --out-dir are required")
		os.Exit(2)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal("failed to create output directory", err)
	}

	var fh *os.File
	if *ordersPath == "-" {
		fh = os.Stdin
	} else {
		f, err := os.Open(*ordersPath)
		if err != nil {
			fatal("failed to open orders CSV", err)
		}
		defer f.Close()
		fh = f
	}

	rows, err := csv.NewReader(fh).ReadAll()
	if err != nil {
		fatal("failed to parse orders CSV", err)
	}
	if len(rows) < 2 {
		fatal("orders CSV has no data rows", nil)
	}

	header := indexHeader(rows[0])
	if _, ok := header["email"]; !ok {
		fatal("orders CSV must have an \"email\" column", nil)
	}

	okCount, failCount := 0, 0
	for i, row := range rows[1:] {
		lineNo := i + 2
		if allBlank(row) {
			continue
		}
		p := issueParams{
			privateKeyPath: *privateKeyPath,
			plan:           cell(row, header, "plan", string(license.PlanPro)),
			email:          cell(row, header, "email", ""),
			order:          cell(row, header, "order", ""),
			expires:        cell(row, header, "expires", ""),
			features:       cell(row, header, "features", ""),
			days:           *defaultDays,
			seats:          1,
			noVerify:       true,
		}
		if v := cell(row, header, "days", ""); v != "" {
			d, err := parseInt(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "line %d: invalid days %q, skipped\n", lineNo, v)
				failCount++
				continue
			}
			p.days = d
		}
		if v := cell(row, header, "seats", ""); v != "" {
			s, err := parseInt(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "line %d: invalid seats %q, skipped\n", lineNo, v)
				failCount++
				continue
			}
			p.seats = s
		}
		if p.email == "" {
			fmt.Fprintf(os.Stderr, "line %d: missing email, skipped\n", lineNo)
			failCount++
			continue
		}

		// One bad row must not abort the whole batch: this is exactly
		// where manual license issuing goes wrong.
		token, info, err := issueToken(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "line %d (%s): %v\n", lineNo, p.email, err)
			failCount++
			continue
		}
		if _, err := license.Verify(token); err != nil {
			fmt.Fprintf(os.Stderr, "line %d (%s): issued token failed verification: %v\n", lineNo, p.email, err)
			failCount++
			continue
		}

		name := p.order
		if name == "" {
			name = safeFileName(p.email)
		}
		outPath := filepath.Join(*outDir, safeFileName(name)+".key")
		if err := os.WriteFile(outPath, []byte(token+"\n"), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "line %d (%s): write %s: %v\n", lineNo, p.email, outPath, err)
			failCount++
			continue
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", info.Email, info.Plan, info.Order, expiryLabel(info), outPath)
		okCount++
	}
	fmt.Fprintf(os.Stderr, "\nissued %d, failed %d, output directory: %s\n", okCount, failCount, *outDir)
	if failCount > 0 {
		os.Exit(1)
	}
}

func indexHeader(row []string) map[string]int {
	m := make(map[string]int, len(row))
	for i, h := range row {
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}

func cell(row []string, header map[string]int, name, fallback string) string {
	i, ok := header[name]
	if !ok || i >= len(row) {
		return fallback
	}
	if v := strings.TrimSpace(row[i]); v != "" {
		return v
	}
	return fallback
}

func allBlank(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

// safeFileName strips characters that could escape the output directory.
func safeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '@':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "license"
	}
	return out
}

func readPrivateKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(raw))
	// Tolerate keygen output pasted as a whole: "private_key: <hex>".
	if i := strings.IndexByte(text, ':'); i >= 0 && strings.Contains(text[:i], "private_key") {
		text = strings.TrimSpace(text[i+1:])
	}
	b, err := hex.DecodeString(strings.ReplaceAll(text, " ", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key length %d, want %d bytes", len(b), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(b), nil
}

func splitFeatures(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func expiryLabel(info *license.Info) string {
	if info == nil || info.ExpiresAt.IsZero() {
		return "perpetual"
	}
	return fmt.Sprintf("%s (%d days left)",
		info.ExpiresAt.Format("2006-01-02"), info.DaysLeft())
}

func shortKey(hexKey string) string {
	if len(hexKey) <= 16 {
		return hexKey
	}
	return hexKey[:8] + "..." + hexKey[len(hexKey)-8:]
}

func fatal(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	}
	os.Exit(1)
}
