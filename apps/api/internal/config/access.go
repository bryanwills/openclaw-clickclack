package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/openclaw/clickclack/apps/api/internal/authpolicy"
)

func normalizeAccessConfig(c *Config) error {
	teamDomain := strings.TrimSpace(c.AccessTeamDomain)
	audience := strings.TrimSpace(c.AccessAUD)
	if (teamDomain != "") != (audience != "") {
		return errors.New("CLICKCLACK_ACCESS_TEAM_DOMAIN and CLICKCLACK_ACCESS_AUD must be configured together")
	}
	if teamDomain != "" {
		canonicalTeamDomain, err := authpolicy.CanonicalPublicURL(teamDomain)
		if err != nil {
			return fmt.Errorf("CLICKCLACK_ACCESS_TEAM_DOMAIN: %w", err)
		}
		parsedTeamDomain, err := url.Parse(canonicalTeamDomain)
		if err != nil || parsedTeamDomain.Scheme != "https" {
			return errors.New("CLICKCLACK_ACCESS_TEAM_DOMAIN must be an HTTPS origin")
		}
		teamDomain = canonicalTeamDomain
	}
	c.AccessTeamDomain = teamDomain
	c.AccessAUD = audience
	return nil
}
