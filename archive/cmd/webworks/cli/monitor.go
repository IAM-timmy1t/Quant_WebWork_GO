package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/timot/Quant_WebWork_GO/internal/monitoring"
)

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorMetricsCmd)
	monitorCmd.AddCommand(monitorSecurityCmd)
	monitorCmd.AddCommand(monitorAlertsCmd)
	monitorCmd.AddCommand(monitorWatchCmd)

	// Add flags for monitor commands
	monitorMetricsCmd.Flags().StringP("type", "t", "", "metric type (cpu, memory, disk, network)")
	monitorMetricsCmd.Flags().StringP("format", "f", "table", "output format (table, json)")
	monitorMetricsCmd.Flags().IntP("limit", "l", 10, "number of data points")

	monitorSecurityCmd.Flags().StringP("severity", "s", "", "filter by severity (low, medium, high, critical)")
	monitorSecurityCmd.Flags().IntP("limit", "l", 10, "number of events to show")

	monitorWatchCmd.Flags().StringP("type", "t", "all", "what to watch (all, metrics, security, alerts)")
	monitorWatchCmd.Flags().IntP("interval", "i", 5, "refresh interval in seconds")
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor system status",
	Long:  `Monitor system metrics, security events, and alerts in real-time.`,
}

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current system status",
	Run: func(cmd *cobra.Command, args []string) {
		monitor := monitoring.NewMonitor(cfg)
		
		// Get system metrics
		metrics, err := monitor.GetSystemMetrics()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting system metrics: %v\n", err)
			os.Exit(1)
		}

		// Get security score
		securityScore, err := monitor.GetSecurityScore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting security score: %v\n", err)
			os.Exit(1)
		}

		// Get active services count
		services, err := monitor.GetAllServices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting services: %v\n", err)
			os.Exit(1)
		}

		activeServices := 0
		for _, svc := range services {
			if svc.Status == "online" {
				activeServices++
			}
		}

		// Print system status
		fmt.Println("System Status:")
		fmt.Printf("Time: %s\n\n", time.Now().Format(time.RFC3339))

		fmt.Println("Resource Usage:")
		fmt.Printf("  CPU: %.2f%%\n", metrics.CPU.Usage)
		fmt.Printf("  Memory: %.2f%% (%.2f GB / %.2f GB)\n",
			metrics.Memory.UsagePercent,
			float64(metrics.Memory.Used)/(1024*1024*1024),
			float64(metrics.Memory.Total)/(1024*1024*1024))
		fmt.Printf("  Disk: %.2f%% (%.2f GB free)\n",
			metrics.Disk.UsagePercent,
			float64(metrics.Disk.Free)/(1024*1024*1024))

		fmt.Printf("\nSecurity Score: %.2f/100\n", securityScore)
		fmt.Printf("Active Services: %d/%d\n", activeServices, len(services))
	},
}

var monitorMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show system metrics",
	Run: func(cmd *cobra.Command, args []string) {
		metricType, _ := cmd.Flags().GetString("type")
		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")

		monitor := monitoring.NewMonitor(cfg)
		metrics, err := monitor.GetDetailedMetrics(metricType, limit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting metrics: %v\n", err)
			os.Exit(1)
		}

		if format == "json" {
			json, err := json.MarshalIndent(metrics, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting metrics: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(json))
			return
		}

		// Print metrics in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tTYPE\tVALUE\tUNIT")
		for _, m := range metrics {
			fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\n",
				m.Timestamp.Format(time.RFC3339),
				m.Type,
				m.Value,
				m.Unit,
			)
		}
		w.Flush()
	},
}

var monitorSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Show security events",
	Run: func(cmd *cobra.Command, args []string) {
		severity, _ := cmd.Flags().GetString("severity")
		limit, _ := cmd.Flags().GetInt("limit")

		monitor := monitoring.NewMonitor(cfg)
		events, err := monitor.GetSecurityEvents(limit, severity)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting security events: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIMESTAMP\tSEVERITY\tTYPE\tDESCRIPTION\tSTATUS")
		for _, event := range events {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				event.Timestamp.Format(time.RFC3339),
				event.Severity,
				event.Type,
				event.Description,
				event.Status,
			)
		}
		w.Flush()
	},
}

var monitorAlertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Show active alerts",
	Run: func(cmd *cobra.Command, args []string) {
		monitor := monitoring.NewMonitor(cfg)
		alerts, err := monitor.GetActiveAlerts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting alerts: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTIMESTAMP\tSEVERITY\tSOURCE\tMESSAGE")
		for _, alert := range alerts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				alert.ID,
				alert.Timestamp.Format(time.RFC3339),
				alert.Severity,
				alert.Source,
				alert.Message,
			)
		}
		w.Flush()
	},
}

var monitorWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch system metrics and events in real-time",
	Run: func(cmd *cobra.Command, args []string) {
		watchType, _ := cmd.Flags().GetString("type")
		interval, _ := cmd.Flags().GetInt("interval")

		monitor := monitoring.NewMonitor(cfg)
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		// Clear screen and hide cursor
		fmt.Print("\033[2J\033[?25l")
		defer fmt.Print("\033[?25h") // Show cursor on exit

		for {
			// Move cursor to top
			fmt.Print("\033[H")

			switch watchType {
			case "all", "metrics":
				metrics, err := monitor.GetSystemMetrics()
				if err == nil {
					fmt.Println("System Metrics:")
					fmt.Printf("CPU: %.2f%%  Memory: %.2f%%  Disk: %.2f%%\n",
						metrics.CPU.Usage,
						metrics.Memory.UsagePercent,
						metrics.Disk.UsagePercent,
					)
				}
			}

			if watchType == "all" || watchType == "security" {
				events, err := monitor.GetSecurityEvents(5, "")
				if err == nil {
					fmt.Println("\nRecent Security Events:")
					for _, event := range events {
						fmt.Printf("[%s] %s: %s\n",
							event.Severity,
							event.Timestamp.Format("15:04:05"),
							event.Description,
						)
					}
				}
			}

			if watchType == "all" || watchType == "alerts" {
				alerts, err := monitor.GetActiveAlerts()
				if err == nil {
					fmt.Println("\nActive Alerts:")
					for _, alert := range alerts {
						fmt.Printf("[%s] %s: %s\n",
							alert.Severity,
							alert.Timestamp.Format("15:04:05"),
							alert.Message,
						)
					}
				}
			}

			fmt.Printf("\nLast updated: %s (Refresh every %ds - Press Ctrl+C to exit)\n",
				time.Now().Format("15:04:05"),
				interval,
			)

			<-ticker.C
		}
	},
}
