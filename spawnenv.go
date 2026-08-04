package contracts

import (
	"fmt"
	"sort"
	"strings"
)

// Ces trois fonctions transportent les variables d'environnement qu'un host
// injecte dans le process enfant d'un backend au spawn.
//
// Elles vivent dans contracts, et non dans chaque backend, parce que le host
// ENCODE et les backends DÉCODENT : séparer les deux moitiés entre dépôts les
// ferait diverger sans qu'aucune suite de tests ne puisse le voir.
//
// Le transport passe par PluginConfig.Settings, qui est un map[string]string.
// Il n'existe pas de canal typé jusqu'au plugin, et en ajouter un obligerait à
// changer la signature de toutes les factories.

// MergeEnv superpose extra sur base, au format "K=V" de os.Environ(). Une clé
// injectée REMPLACE celle héritée plutôt que de s'y ajouter : un exec.Cmd avec
// deux entrées pour la même variable a un comportement dépendant de la
// plateforme, et une valeur périmée héritée du daemon détournerait
// silencieusement la session.
//
// extra vide retourne base tel quel — c'est la non-régression de la route native.
func MergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, e := range base {
		if k, _, found := strings.Cut(e, "="); found {
			if _, override := extra[k]; override {
				continue
			}
		}
		out = append(out, e)
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+extra[k])
	}
	return out
}

// ParseEnvSetting décode la valeur du réglage "env" : des paires K=V séparées
// par des sauts de ligne. Seul le PREMIER '=' sépare la clé de la valeur — les
// jetons sont souvent en base64 et en contiennent.
func ParseEnvSetting(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// EncodeEnvSetting est l'inverse de ParseEnvSetting. Les clés sont triées pour
// que la valeur soit déterministe : sans ça deux spawns identiques
// transporteraient des chaînes différentes, et la fonction ne serait pas testable.
func EncodeEnvSetting(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, env[k])
	}
	return b.String()
}
