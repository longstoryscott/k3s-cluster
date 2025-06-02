package config

// OAuthConfig holds Oauth validation parameters
type OAuthConfig struct {
	JWKSUri      string `json:"jwksUri" mapstructure:"jwks_uri"`
	ClientID     string `json:"clientId" mapstructure:"client_id"`
	ClientSecret string `json:"clientSecret" mapstructure:"client_secret"`
}

// GetLDAPConfig returns LDAP configuration from environment variables with fallbacks
func GetOauthConfig() OAuthConfig {
	return OAuthConfig{
		JWKSUri:      getEnv("USRMGR_JWKS_URI", ""),
		ClientID:     getEnv("USRMGR_OAUTH_CLIENT_ID", ""),
		ClientSecret: getEnv("USRMGR_OAUTH_CLIENT_SECRET", ""),
	}
}
