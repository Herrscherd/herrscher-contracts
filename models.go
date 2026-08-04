package contracts

import (
	"fmt"
	"strings"
)

// Route dit qui sert un modèle. C'est la seule notion de « fournisseur » que le
// système porte, et elle est binaire : soit le CLI vendor parle à son propre
// service avec le login présent sur la machine, soit il parle à la passerelle
// Neublox, qui route vers l'amont réel côté serveur. herrscher n'a donc jamais à
// connaître Z.ai, Alibaba ou DeepSeek.
type Route string

const (
	// RouteNative : le CLI utilise l'authentification de l'utilisateur.
	RouteNative Route = "native"
	// RouteGateway : le CLI est pointé vers la passerelle Neublox.
	RouteGateway Route = "gateway"
)

// RoutePolicy borne les routes qu'un host accepte de servir. Le build public de
// l'app pose PolicyGatewayOnly : un modèle natif n'y est pas masqué, il est
// absent du catalogue, donc impossible à sélectionner, persister ou reprendre.
type RoutePolicy string

const (
	// PolicyAll est le zéro value : un host qui ne configure rien se comporte
	// comme avant ce changement. Le build interne reste sur cette valeur.
	PolicyAll RoutePolicy = ""
	// PolicyGatewayOnly n'expose que les modèles servis par la passerelle.
	PolicyGatewayOnly RoutePolicy = "gateway-only"
)

// Allows dit si la politique accepte cette route.
func (p RoutePolicy) Allows(r Route) bool {
	if p == PolicyGatewayOnly {
		return r == RouteGateway
	}
	return true
}

// ModelSpec est un modèle qu'un backend déclare savoir exécuter. Il vit dans le
// Manifest — donc lisible sans instancier de backend, ce dont l'app a besoin pour
// peupler son sélecteur avant qu'une session existe.
type ModelSpec struct {
	// ID est l'identifiant stable persisté dans l'état de session. C'est lui,
	// et non Label ni Arg, qui permet de retrouver la route au resume.
	ID string
	// Label est ce que l'utilisateur voit. Aucune notion de vendor, de
	// fournisseur ou de protocole ne doit y apparaître.
	Label string
	// Arg est la valeur passée à --model au CLI. Elle diffère souvent de l'ID
	// (cursor encode l'effort dedans, la passerelle renomme).
	Arg string
	// Efforts liste les niveaux acceptés. Vide = pas d'axe effort séparé.
	Efforts []string
	Route   Route
	// InputPrice est le prix affiché en USD par million de tokens d'entrée.
	// Pour une route gateway c'est NOTRE prix, pas celui de l'amont. 0 =
	// inconnu, le coût s'affiche alors en tokens seuls.
	InputPrice float64
}

// ValidateModels vérifie l'intégrité du catalogue d'un backend. kind nomme le
// backend dans les messages d'erreur. Appelé par le host au démarrage : un
// catalogue incohérent doit tuer le daemon avec un message clair, pas produire
// un sélecteur silencieusement faux.
func ValidateModels(kind string, models []ModelSpec) error {
	seen := make(map[string]bool, len(models))
	for i, m := range models {
		if strings.TrimSpace(m.ID) == "" {
			return fmt.Errorf("backend %q: model #%d has an empty ID", kind, i)
		}
		if seen[m.ID] {
			return fmt.Errorf("backend %q: duplicate model ID %q", kind, m.ID)
		}
		seen[m.ID] = true
		if strings.TrimSpace(m.Label) == "" {
			return fmt.Errorf("backend %q: model %q has an empty Label", kind, m.ID)
		}
		if strings.TrimSpace(m.Arg) == "" {
			return fmt.Errorf("backend %q: model %q has an empty Arg", kind, m.ID)
		}
		if m.Route != RouteNative && m.Route != RouteGateway {
			return fmt.Errorf("backend %q: model %q has unknown route %q", kind, m.ID, m.Route)
		}
	}
	return nil
}

// FilterModels garde les modèles que la politique autorise. Le résultat est
// toujours non-nil pour que les appelants puissent le sérialiser en JSON sans
// produire `null`.
func FilterModels(models []ModelSpec, p RoutePolicy) []ModelSpec {
	out := make([]ModelSpec, 0, len(models))
	for _, m := range models {
		if p.Allows(m.Route) {
			out = append(out, m)
		}
	}
	return out
}

// GatewayCreds est la paire (base URL, jeton) qu'une route gateway exige. Elle
// est atomique PAR CONSTRUCTION : NewGatewayCreds est le seul moyen d'en obtenir
// une valide, et il refuse toute moitié manquante.
//
// La raison est légale, pas esthétique. Poser la base URL sans jeton envoie le
// trafic vers la passerelle tout en le débitant sur l'abonnement claude.ai de
// l'utilisateur — la forme qu'Anthropic interdit explicitement aux développeurs
// tiers. Rendre cet état inexprimable vaut mieux que le tester après coup.
type GatewayCreds struct {
	BaseURL string
	Token   string
}

// NewGatewayCreds construit une paire complète, ou échoue.
func NewGatewayCreds(baseURL, token string) (GatewayCreds, error) {
	baseURL, token = strings.TrimSpace(baseURL), strings.TrimSpace(token)
	switch {
	case baseURL == "" && token == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: neither base URL nor token available")
	case baseURL == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: token without a base URL")
	case token == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: base URL without a token — refusing to run on the user's own subscription")
	}
	return GatewayCreds{BaseURL: baseURL, Token: token}, nil
}
