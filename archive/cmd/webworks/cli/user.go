package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/auth"
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userRolesCmd)

	// Add flags for user commands
	userAddCmd.Flags().StringP("username", "u", "", "username")
	userAddCmd.Flags().StringP("email", "e", "", "email address")
	userAddCmd.Flags().StringP("role", "r", "", "user role (admin, operator, viewer)")
	userAddCmd.Flags().StringP("password", "p", "", "password")
	userAddCmd.MarkFlagRequired("username")
	userAddCmd.MarkFlagRequired("email")
	userAddCmd.MarkFlagRequired("role")
	userAddCmd.MarkFlagRequired("password")

	userUpdateCmd.Flags().StringP("email", "e", "", "email address")
	userUpdateCmd.Flags().StringP("role", "r", "", "user role")
	userUpdateCmd.Flags().StringP("password", "p", "", "password")

	userListCmd.Flags().StringP("role", "r", "", "filter by role")
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  `Add, remove, update, and manage users in the Quant WebWorks gateway.`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		role, _ := cmd.Flags().GetString("role")

		authenticator := auth.NewAuthenticator(cfg)
		users, err := authenticator.GetUsers(role)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting users: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tROLE\tLAST ACTIVE")
		for _, user := range users {
			lastActive := "Never"
			if !user.LastActive.IsZero() {
				lastActive = user.LastActive.Format(time.RFC3339)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				user.ID,
				user.Username,
				user.Email,
				user.Role,
				lastActive,
			)
		}
		w.Flush()
	},
}

var userAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new user",
	Run: func(cmd *cobra.Command, args []string) {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		role, _ := cmd.Flags().GetString("role")
		password, _ := cmd.Flags().GetString("password")

		authenticator := auth.NewAuthenticator(cfg)
		user, err := authenticator.CreateUser(auth.CreateUserInput{
			Username: username,
			Email:    email,
			Role:     role,
			Password: password,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User '%s' added successfully with ID: %s\n", user.Username, user.ID)
		
		// If user is an admin, generate and display TOTP secret
		if user.Role == "admin" {
			secret, err := authenticator.GenerateTOTPSecret(user.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to generate TOTP secret: %v\n", err)
				return
			}
			fmt.Printf("\nTOTP Secret for admin user: %s\n", secret)
			fmt.Println("Please store this secret securely and use it to configure your 2FA app.")
		}
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update [user-id]",
	Short: "Update a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		userID := args[0]
		email, _ := cmd.Flags().GetString("email")
		role, _ := cmd.Flags().GetString("role")
		password, _ := cmd.Flags().GetString("password")

		authenticator := auth.NewAuthenticator(cfg)
		user, err := authenticator.UpdateUser(userID, auth.UpdateUserInput{
			Email:    email,
			Role:     role,
			Password: password,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User '%s' updated successfully\n", user.Username)
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete [user-id]",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		userID := args[0]

		authenticator := auth.NewAuthenticator(cfg)
		success, err := authenticator.DeleteUser(userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting user: %v\n", err)
			os.Exit(1)
		}

		if success {
			fmt.Printf("User '%s' deleted successfully\n", userID)
		}
	},
}

var userRolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "List available user roles and their permissions",
	Run: func(cmd *cobra.Command, args []string) {
		roles := []struct {
			Name        string
			Description string
			Permissions []string
		}{
			{
				Name:        "admin",
				Description: "Full system access",
				Permissions: []string{
					"Manage users and roles",
					"Configure system settings",
					"Manage services",
					"View all metrics and logs",
					"Access security features",
				},
			},
			{
				Name:        "operator",
				Description: "Service and monitoring management",
				Permissions: []string{
					"Manage services",
					"View metrics and logs",
					"Handle alerts",
					"Update service configuration",
				},
			},
			{
				Name:        "viewer",
				Description: "Read-only access",
				Permissions: []string{
					"View services status",
					"View metrics and logs",
					"View system dashboard",
				},
			},
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ROLE\tDESCRIPTION\tPERMISSIONS")
		for _, role := range roles {
			permissions := ""
			for i, perm := range role.Permissions {
				if i > 0 {
					permissions += ", "
				}
				permissions += perm
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				role.Name,
				role.Description,
				permissions,
			)
		}
		w.Flush()
	},
}

