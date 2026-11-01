// Command licensegen is Cortex's offline license issuing tool.
//
// It is used by Cortex maintainers/sales to generate a key pair and to issue
// signed license tokens for customers. The private key must never be embedded
// in the Cortex binary or committed to the repository.
//
// Usage:
//
//	licensegen keygen [--out-private private.key]
//	licensegen issue --private-key private.key --plan pro --email user@example.com \
//	    --days 365 --seats 1 --out license.key
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
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
  issue    Issue a signed license token for a customer

Examples:
  licensegen keygen --out-private _temp/license-private.key
  licensegen issue --private-key _temp/license-private.key --plan pro \
      --email user@example.com --days 365 --seats 1 --out _temp/license.key

Flags for issue:
  --private-key <path>   private key hex file (required)
  --plan <plan>          free|pro|team|enterprise (default pro)
  --email <email>        customer email
  --order <id>           order / invoice id
  --days <n>             validity in days, 0 = perpetual (default 365)
  --seats <n>            number of seats (default 1)
  --features <a,b,c>     comma separated feature list
  --out <path>           write license text to a file
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

func issue(args []string) {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	privateKeyPath := fs.String("private-key", "", "path to private key hex file (required)")
	plan := fs.String("plan", string(license.PlanPro), "free|pro|team|enterprise")
	email := fs.String("email", "", "customer email")
	order := fs.String("order", "", "order / invoice id")
	days := fs.Int("days", 365, "validity in days, 0 = perpetual")
	seats := fs.Int("seats", 1, "number of seats")
	features := fs.String("features", "", "comma separated feature list")
	out := fs.String("out", "", "write license text to this file")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *privateKeyPath == "" {
		fmt.Fprintln(os.Stderr, "error: --private-key is required")
		os.Exit(2)
	}

	priv, err := readPrivateKey(*privateKeyPath)
	if err != nil {
		fatal("failed to load private key", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	info := license.Info{
		Plan:     license.Plan(strings.ToLower(strings.TrimSpace(*plan))),
		Email:    *email,
		Order:    *order,
		Seats:    *seats,
		IssuedAt: now,
		Features: splitFeatures(*features),
	}
	if *days < 0 {
		fatal("--days must be >= 0", nil)
	}
	if *days > 0 {
		info.ExpiresAt = now.AddDate(0, 0, *days)
	}

	token, err := license.Sign(priv, info)
	if err != nil {
		fatal("failed to sign license", err)
	}

	fmt.Println(token)
	if *out != "" {
		if err := os.WriteFile(*out, []byte(token+"\n"), 0o644); err != nil {
			fatal("failed to write license file", err)
		}
		fmt.Fprintf(os.Stderr, "\nlicense written to: %s\n", *out)
	}
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

func fatal(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	}
	os.Exit(1)
}
