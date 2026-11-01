package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lh123aa/cortex/internal/storage"
	"github.com/lh123aa/cortex/pkg/license"
	"github.com/spf13/cobra"
)

const pricingURL = license.DefaultPricingURL

// NewLicenseCmd manages the offline Ed25519 license installed on this machine.
func NewLicenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Manage your Cortex license (activate, status, deactivate)",
		Long: `Manage your offline Cortex license.

A license is a signed text token issued for your email/order. Activation
verifies the Ed25519 signature locally and stores the token in
~/.cortex/license.key. No network connection is required.`,
	}
	cmd.AddCommand(newLicenseActivateCmd())
	cmd.AddCommand(newLicenseStatusCmd())
	cmd.AddCommand(newLicenseDeactivateCmd())
	return cmd
}

func newLicenseActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activate <license-key>",
		Short: "Verify and install a license key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := license.Activate(args[0])
			if err != nil {
				return fmt.Errorf("activation failed: %w", err)
			}
			fmt.Println("  ✅ License activated")
			fmt.Println()
			printLicenseInfo(info)
			fmt.Printf("  Saved to:  %s\n", license.Path())
			return nil
		},
	}
}

func newLicenseStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the currently installed license",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := license.Path()
			info, err := license.Current()
			if err != nil {
				return fmt.Errorf("invalid license at %s: %w\nRun 'cortex license deactivate' to remove it, then activate a valid key", path, err)
			}

			fmt.Println("\n🔑 Cortex License")
			fmt.Println("=================")
			if info == nil {
				fmt.Println("  Plan:      Free")
				fmt.Println("  License:   not activated")
				fmt.Printf("  Path:      %s (missing)\n", path)
				fmt.Println()
				fmt.Println("  Limits:    5,000 documents · 1.0 GB storage")
				fmt.Printf("\n  Upgrade:   %s\n", pricingURL)
				fmt.Println("  Activate:  cortex license activate <license-key>")
				return nil
			}

			printLicenseInfo(info)
			fmt.Printf("  Path:      %s\n", path)
			return nil
		},
	}
}

func newLicenseDeactivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate",
		Short: "Remove the local license and return to the Free plan",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := license.Deactivate(); err != nil {
				return fmt.Errorf("deactivation failed: %w", err)
			}
			fmt.Println("  ✅ License deactivated. Cortex is now on the Free plan.")
			return nil
		},
	}
}

// printLicenseInfo renders a verified license in human-readable form.
func printLicenseInfo(info *license.Info) {
	lim := license.Limits(info.Plan)
	fmt.Printf("  Plan:      %s\n", displayPlan(info.Plan))
	if info.Email != "" {
		fmt.Printf("  Email:     %s\n", info.Email)
	}
	if info.Order != "" {
		fmt.Printf("  Order:     %s\n", info.Order)
	}
	if info.Seats > 0 {
		fmt.Printf("  Seats:     %d\n", info.Seats)
	}
	fmt.Printf("  Expires:   %s\n", displayExpiry(info))
	if len(info.Features) > 0 {
		fmt.Printf("  Features:  %s\n", strings.Join(info.Features, ", "))
	}
	if lim.Documents < 0 {
		fmt.Println("  Limits:    unlimited documents · unlimited storage")
	} else {
		fmt.Printf("  Limits:    %s documents · %s storage\n", formatCount(lim.Documents), formatBytes(lim.StorageBytes))
	}
}

func displayPlan(p license.Plan) string {
	if p == "" {
		return "Free"
	}
	s := string(p)
	return strings.ToUpper(s[:1]) + s[1:]
}

func displayExpiry(info *license.Info) string {
	if info == nil || info.ExpiresAt.IsZero() {
		return "Never (perpetual)"
	}
	days := int(time.Until(info.ExpiresAt).Hours() / 24)
	if days < 0 {
		return info.ExpiresAt.Format("2006-01-02") + " (expired)"
	}
	return fmt.Sprintf("%s (%d days left)", info.ExpiresAt.Format("2006-01-02"), days)
}

func formatCount(n int64) string {
	// Insert thousands separators for readability (500000 -> 500,000).
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// currentLicenseOrFree returns the installed license (nil when absent) while
// degrading gracefully for invalid/expired license files: a broken license
// behaves like Free and a warning is printed to stderr.
func currentLicenseOrFree() *license.Info {
	info, err := license.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Ignoring invalid license at %s: %v\n", license.Path(), err)
		fmt.Fprintln(os.Stderr, "     Run 'cortex license status' for details, or 'cortex license deactivate' to reset.")
		return nil
	}
	return info
}

// enforceQuota is the unified quota check used before indexing work starts.
func enforceQuota(documents int, storageBytes int64) error {
	return license.CheckQuota(currentLicenseOrFree(), documents, storageBytes)
}

// checkQuotaFromStorage enforces plan limits against an already open storage.
func checkQuotaFromStorage(st storage.Storage) error {
	documents, err := st.GetDocumentsCount("")
	if err != nil {
		return fmt.Errorf("failed to read document count: %w", err)
	}
	used, err := st.CalculateStorageUsed("")
	if err != nil {
		return fmt.Errorf("failed to calculate storage usage: %w", err)
	}
	return enforceQuota(documents, used)
}

// checkQuotaPreRun opens the configured storage and enforces plan limits.
// It is used as the `cortex index` PreRunE so the quota is checked before any
// indexing work begins.
func checkQuotaPreRun() error {
	cfg, logger, err := loadConfig()
	if err != nil {
		return err
	}
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		return err
	}
	defer st.Close()
	return checkQuotaFromStorage(st)
}
