// Config returns the adapter configuration
func (a *SharedBaseAdapter) Config() SharedAdapterConfig {
	// Return a copy to prevent modification
	config := a.config

	// Redact sensitive information
	if hasSensitiveInfo := len(config.Options) > 0; hasSensitiveInfo {
		newOptions := make(map[string]interface{})
		for k, v := range config.Options {
			if k == "password" || k == "secret" || k == "api_key" || k == "token" {
				newOptions[k] = "******"
			} else {
				newOptions[k] = v
			}
		}
		config.Options = newOptions
	}

	return config
} 