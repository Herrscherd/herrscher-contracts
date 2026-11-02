package contracts

import (
	"strings"
)

// Setting is one config value a plugin declares it reads. It is the env/config
// analogue of Param: the plugin states its config surface once (in its Manifest),
// and the host resolves and validates it generically. Required values that are
// absent fail the daemon at startup rather than surfacing as an opaque error
// deep inside the plugin.
type Setting struct {
	// Key is the name the plugin reads the value by through PluginConfig.Get.
	Key string
	// Env is the environment variable the value is read from. Empty means the
	// setting has no environment source.
	Env string
	// Help describes the setting for the operator.
	Help string
	// Required makes an empty value a startup failure.
	Required bool
	// Default is the value used when Env is unset or empty.
	Default string
}

// Resolve builds a PluginConfig from declared settings: each Setting's value is
// read from getenv(Setting.Env), falling back to Default. A Required setting that
// is still empty makes it return a [*MissingSettingsError] naming every missing
// one, so the host can refuse to start a misconfigured plugin with a clear
// message. getenv is injected so the contract stays free of os (and is testable).
func Resolve(settings []Setting, getenv func(string) string) (PluginConfig, error) {
	cfg := PluginConfig{Settings: make(map[string]string, len(settings))}
	var missing []string
	for _, setting := range settings {
		value := ""
		if setting.Env != "" {
			value = getenv(setting.Env)
		}
		if value == "" {
			value = setting.Default
		}
		if value == "" && setting.Required {
			if setting.Env != "" {
				missing = append(missing, setting.Key+" (set "+setting.Env+")")
			} else {
				missing = append(missing, setting.Key)
			}
			continue
		}
		cfg.Settings[setting.Key] = value
	}
	if len(missing) > 0 {
		return cfg, &MissingSettingsError{Missing: missing}
	}
	return cfg, nil
}

// MissingSettingsError reports the required settings that [Resolve] found empty.
type MissingSettingsError struct {
	// Missing names each empty required setting, with its environment variable when it has one.
	Missing []string
}

// Error lists the missing settings in one line.
func (e *MissingSettingsError) Error() string {
	return "missing required config: " + strings.Join(e.Missing, ", ")
}
