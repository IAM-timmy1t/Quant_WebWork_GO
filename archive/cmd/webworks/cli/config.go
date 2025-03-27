package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/config"
	"gopkg.in/yaml.v2"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)

	// Add flags for config commands
	configGetCmd.Flags().StringP("format", "f", "plain", "output format (plain, json, yaml)")
	configSetCmd.Flags().StringP("type", "t", "string", "value type (string, int, bool, json)")
	configExportCmd.Flags().StringP("format", "f", "yaml", "export format (yaml, json)")
	configImportCmd.Flags().StringP("format", "f", "yaml", "import format (yaml, json)")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage system configuration",
	Long: `Manage Quant WebWorks configuration settings.
	
Configuration sections include:
- System: Core system settings
- Security: Security and authentication settings
- Monitoring: Monitoring and alerting thresholds
- Services: Default service configuration
- Network: Network and proxy settings
- Storage: Data storage and retention settings`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		format, _ := cmd.Flags().GetString("format")

		value, err := cfg.Get(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting config value: %v\n", err)
			os.Exit(1)
		}

		switch format {
		case "json":
			json, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting value: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(json))
		case "yaml":
			yaml, err := yaml.Marshal(value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting value: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(string(yaml))
		default:
			fmt.Printf("%v\n", value)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		valueStr := args[1]
		valueType, _ := cmd.Flags().GetString("type")

		var value interface{}
		var err error

		switch valueType {
		case "int":
			value, err = parseIntValue(valueStr)
		case "bool":
			value, err = parseBoolValue(valueStr)
		case "json":
			value, err = parseJSONValue(valueStr)
		default:
			value = valueStr
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing value: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Set(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config value: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully set %s = %v\n", key, value)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list [section]",
	Short: "List configuration values",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		section := ""
		if len(args) > 0 {
			section = args[0]
		}

		settings, err := cfg.List(section)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing config: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE\tTYPE\tDESCRIPTION")
		
		for key, setting := range settings {
			valueType := fmt.Sprintf("%T", setting.Value)
			description := setting.Description
			if description == "" {
				description = "-"
			}

			fmt.Fprintf(w, "%s\t%v\t%s\t%s\n",
				key,
				setting.Value,
				valueType,
				description,
			)
		}
		w.Flush()
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := cfg.Validate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating config: %v\n", err)
			os.Exit(1)
		}

		if len(issues) == 0 {
			fmt.Println("Configuration is valid.")
			return
		}

		fmt.Println("Configuration validation issues:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SEVERITY\tKEY\tISSUE")
		
		for _, issue := range issues {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				issue.Severity,
				issue.Key,
				issue.Message,
			)
		}
		w.Flush()

		if hasError(issues) {
			os.Exit(1)
		}
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export configuration to file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		format, _ := cmd.Flags().GetString("format")

		// Ensure directory exists
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		var data []byte
		var err error

		switch format {
		case "json":
			data, err = json.MarshalIndent(cfg.GetAll(), "", "  ")
		case "yaml":
			data, err = yaml.Marshal(cfg.GetAll())
		default:
			fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", format)
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting config: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(file, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration exported to %s\n", file)
	},
}

var configImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import configuration from file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		format, _ := cmd.Flags().GetString("format")

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		var config map[string]interface{}

		switch format {
		case "json":
			err = json.Unmarshal(data, &config)
		case "yaml":
			err = yaml.Unmarshal(data, &config)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", format)
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Import(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error importing config: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration imported from %s\n", file)
	},
}

// Helper functions

func parseIntValue(value string) (int64, error) {
	return config.ParseInt(value)
}

func parseBoolValue(value string) (bool, error) {
	value = strings.ToLower(value)
	switch value {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

func parseJSONValue(value string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON value: %v", err)
	}
	return result, nil
}

func hasError(issues []config.ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

