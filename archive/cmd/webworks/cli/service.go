package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/timot/Quant_WebWork_GO/internal/monitoring"
)

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceAddCmd)
	serviceCmd.AddCommand(serviceUpdateCmd)
	serviceCmd.AddCommand(serviceDeleteCmd)
	serviceCmd.AddCommand(serviceStatusCmd)

	// Add flags for service commands
	serviceAddCmd.Flags().StringP("name", "n", "", "service name")
	serviceAddCmd.Flags().StringP("url", "u", "", "service URL")
	serviceAddCmd.Flags().StringP("type", "t", "", "service type")
	serviceAddCmd.Flags().StringP("auth", "a", "", "authentication type (none, basic, jwt, apikey)")
	serviceAddCmd.MarkFlagRequired("name")
	serviceAddCmd.MarkFlagRequired("url")

	serviceUpdateCmd.Flags().StringP("name", "n", "", "service name")
	serviceUpdateCmd.Flags().StringP("url", "u", "", "service URL")
	serviceUpdateCmd.Flags().StringP("status", "s", "", "service status")
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
	Long:  `Add, remove, update, and monitor services in the Quant WebWorks gateway.`,
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services",
	Run: func(cmd *cobra.Command, args []string) {
		monitor := monitoring.NewMonitor(cfg)
		services, err := monitor.GetAllServices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting services: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tURL\tSTATUS\tHEALTH")
		for _, svc := range services {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f%%\n",
				svc.ID,
				svc.Name,
				svc.URL,
				svc.Status,
				svc.Health,
			)
		}
		w.Flush()
	},
}

var serviceAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new service",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		svcType, _ := cmd.Flags().GetString("type")
		auth, _ := cmd.Flags().GetString("auth")

		monitor := monitoring.NewMonitor(cfg)
		service, err := monitor.RegisterService(monitoring.RegisterServiceInput{
			Name:     name,
			URL:      url,
			Type:     svcType,
			AuthType: auth,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding service: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Service '%s' added successfully with ID: %s\n", service.Name, service.ID)
	},
}

var serviceUpdateCmd = &cobra.Command{
	Use:   "update [service-id]",
	Short: "Update a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceID := args[0]
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		status, _ := cmd.Flags().GetString("status")

		monitor := monitoring.NewMonitor(cfg)
		service, err := monitor.UpdateService(serviceID, monitoring.UpdateServiceInput{
			Name:   name,
			URL:    url,
			Status: status,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating service: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Service '%s' updated successfully\n", service.Name)
	},
}

var serviceDeleteCmd = &cobra.Command{
	Use:   "delete [service-id]",
	Short: "Delete a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceID := args[0]

		monitor := monitoring.NewMonitor(cfg)
		success, err := monitor.DeleteService(serviceID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting service: %v\n", err)
			os.Exit(1)
		}

		if success {
			fmt.Printf("Service '%s' deleted successfully\n", serviceID)
		}
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status [service-id]",
	Short: "Get detailed status of a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceID := args[0]

		monitor := monitoring.NewMonitor(cfg)
		service, err := monitor.GetService(serviceID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting service status: %v\n", err)
			os.Exit(1)
		}

		metrics, err := monitor.GetServiceMetrics(serviceID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting service metrics: %v\n", err)
			os.Exit(1)
		}

		// Print detailed status
		fmt.Printf("Service: %s (%s)\n", service.Name, service.ID)
		fmt.Printf("URL: %s\n", service.URL)
		fmt.Printf("Status: %s\n", service.Status)
		fmt.Printf("Health: %.2f%%\n", service.Health)
		fmt.Printf("\nMetrics:\n")
		fmt.Printf("  Response Time: %.2fms\n", metrics.ResponseTime)
		fmt.Printf("  Request Count: %d\n", metrics.RequestCount)
		fmt.Printf("  Error Rate: %.2f%%\n", metrics.ErrorRate)
		fmt.Printf("  Uptime: %.2f%%\n", metrics.UptimePercentage)
	},
}
