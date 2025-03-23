package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/timot/Quant_WebWork_GO/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "webworks",
	Short: "Quant WebWorks - Advanced Self-Hosting Gateway",
	Long: `Quant WebWorks is a comprehensive self-hosting gateway designed to securely
manage and bridge multiple external applications, websites, dashboards, and services.

Key features include:
- Unified Master 2FA Authentication
- Live Network & Security Dashboard
- One-Click App Bridging & Integration
- Security-First Architecture
- Event-Driven Communication`,
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.webworks.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	var err error
	cfg, err = config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
}
