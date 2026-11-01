package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/spf13/cobra"
)

// 用户与角色管理（Team 版 RBAC）。
//
// 角色：viewer（只读）< editor（可读写）< admin（可管理用户与密钥）

var (
	userRole     string
	userPassword string
)

func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users and roles (admin, editor, viewer)",
		Long: `Manage local Cortex users and their roles.

Roles (from least to most privileged):
  viewer   read-only: search, view documents and stats
  editor   read + write: index, memory writes, delete entries
  admin    everything, plus user management and API key listing

Examples:
  cortex user list
  cortex user add alice --role editor --password 'S3cret!pass'
  cortex user set-role alice admin
  cortex user disable alice`,
	}
	cmd.AddCommand(
		newUserListCmd(),
		newUserAddCmd(),
		newUserSetRoleCmd(),
		newUserDisableCmd(),
		newUserEnableCmd(),
	)
	return cmd
}

func withAuthService(fn func(svc *auth.AuthService) error) error {
	cfg, logger, err := loadConfig()
	if err != nil {
		return err
	}
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		return err
	}
	defer st.Close()

	return fn(auth.NewAuthServiceWithStorage(st))
}

func newUserListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List users with their roles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthService(func(svc *auth.AuthService) error {
				users, err := svc.ListUsers()
				if err != nil {
					return err
				}
				if len(users) == 0 {
					fmt.Println("No users yet. Create one with: cortex user add <name> --role editor --password <pwd>")
					return nil
				}
				tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "USERNAME\tROLE\tTIER\tACTIVE\tLAST LOGIN")
				for _, u := range users {
					last := "-"
					if !u.LastLoginAt.IsZero() {
						last = u.LastLoginAt.Format("2006-01-02 15:04")
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%s\n",
						u.Username, models.NormalizeRole(u.Role), u.Tier, u.IsActive, last)
				}
				return tw.Flush()
			})
		},
	}
}

func newUserAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <username>",
		Short: "Create a user with a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimSpace(args[0])
			if username == "" {
				return fmt.Errorf("username is required")
			}
			if !models.IsValidRole(userRole) {
				return fmt.Errorf("invalid role %q (want admin|editor|viewer)", userRole)
			}
			if userPassword == "" {
				return fmt.Errorf("--password is required (or set CORTEX_USER_PASSWORD)")
			}

			return withAuthService(func(svc *auth.AuthService) error {
				u, err := svc.CreateUserWithRole(username, userPassword, userRole)
				if err != nil {
					return err
				}
				fmt.Printf("✅ Created user %q with role %s\n", u.Username, models.NormalizeRole(u.Role))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&userRole, "role", models.RoleEditor, "role: admin|editor|viewer")
	cmd.Flags().StringVar(&userPassword, "password", os.Getenv("CORTEX_USER_PASSWORD"),
		"password (or set CORTEX_USER_PASSWORD to avoid shell history)")
	return cmd
}

func newUserSetRoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-role <username> <role>",
		Short: "Change a user's role",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username, role := args[0], args[1]
			return withAuthService(func(svc *auth.AuthService) error {
				u, err := svc.SetUserRole(username, role)
				if err != nil {
					return err
				}
				fmt.Printf("✅ %s is now %s\n", u.Username, models.NormalizeRole(u.Role))
				return nil
			})
		},
	}
}

func newUserDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <username>",
		Short: "Disable a user (keeps data)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthService(func(svc *auth.AuthService) error {
				u, err := svc.GetUserByUsername(args[0])
				if err != nil {
					return err
				}
				if u == nil {
					return auth.ErrUserNotFound
				}
				if err := svc.DeactivateUser(u.ID); err != nil {
					return err
				}
				fmt.Printf("✅ Disabled user %q\n", u.Username)
				return nil
			})
		},
	}
}

func newUserEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <username>",
		Short: "Re-enable a disabled user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthService(func(svc *auth.AuthService) error {
				u, err := svc.GetUserByUsername(args[0])
				if err != nil {
					return err
				}
				if u == nil {
					return auth.ErrUserNotFound
				}
				if err := svc.ActivateUser(u.ID); err != nil {
					return err
				}
				fmt.Printf("✅ Enabled user %q\n", u.Username)
				return nil
			})
		},
	}
}
