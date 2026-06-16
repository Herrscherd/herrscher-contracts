package contracts

import (
	"fmt"
	"strings"
)

// Setting is one config value a plugin declares it reads. It is the env/config
// analogue of Param: the plugin states its config surface once (in its Manifest),
// and the host resolves and validates it generically — required values that are
// absent fail the daemon at startup rather than surfacing as an opaque error
// deep inside the plugin.
type Setting struct {
	Key      string // the neutral key the plugin reads via PluginConfig.Get
	Env      string // the environment variable it binds from (empty: not env-bound)
	Help     string
	Required bool
	Default  string // applied when the env var is unset/empty
}

// Resolve builds a PluginConfig from declared settings: each Setting's value is
// read from getenv(Setting.Env), falling back to Default. A Required setting that
// is still empty is an error naming every missing one, so the host can refuse to
// start a misconfigured plugin with a clear message. getenv is injected so the
// contract stays free of os (and is testable).
func Resolve(settings []Setting, getenv func(string) string) (PluginConfig, error) {
	cfg := PluginConfig{Settings: make(map[string]string, len(settings))}
	var missing []string
	for _, s := range settings {
		v := ""
		if s.Env != "" {
			v = getenv(s.Env)
		}
		if v == "" {
			v = s.Default
		}
		if v == "" && s.Required {
			if s.Env != "" {
				missing = append(missing, fmt.Sprintf("%s (set %s)", s.Key, s.Env))
			} else {
				missing = append(missing, s.Key)
			}
			continue
		}
		cfg.Settings[s.Key] = v
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}
