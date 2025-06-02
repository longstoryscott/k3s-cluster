package config

import (
	"os"
)

// LDAPConfig holds LDAP connection parameters
type LDAPConfig struct {
	Host     string
	BindDN   string
	Password string
	BaseDN   string
}

// GetLDAPConfig returns LDAP configuration from environment variables with fallbacks
func GetLDAPConfig() LDAPConfig {
	return LDAPConfig{
		Host:     getEnv("USRMGR_LDAP_URL", "ldap://openldap.auth.svc.cluster.local:389"),
		BindDN:   getEnv("USRMGR_LDAP_BIND_DN", "cn=admin,dc=longstorymedia,dc=com"),
		Password: getEnv("USRMGR_LDAP_BIND_PASSWORD", ""),
		BaseDN:   getEnv("USRMGR_BASE_DN", "dc=longstorymedia,dc=com"),
	}
}

// Helper to get environment variables with default fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
