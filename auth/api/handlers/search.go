package handlers

import (
	"net/http"

	"github.com/go-ldap/ldap/v3"
	"github.com/gofiber/fiber/v3"

	"usrmgr/config"
)

// RegisterSearchHandler registers the search route with the API
func RegisterSearchHandler(router fiber.Router, config config.LDAPConfig) {
	router.Get("/search", func(c fiber.Ctx) error {
		// Get query parameters with defaults
		filter := c.Query("filter", "(objectClass=inetOrgPerson)")

		// Connect to LDAP
		conn, err := ldap.DialURL(config.Host)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP connection failed: " + err.Error(),
			})
		}
		defer conn.Close()

		// Bind to LDAP server
		err = conn.Bind(config.BindDN, config.Password)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP bind failed: " + err.Error(),
			})
		}

		// Search request
		searchRequest := ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			filter,
			[]string{"dn", "cn", "uid", "mail", "sn"},
			nil,
		)

		// Execute search
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP search failed: " + err.Error(),
			})
		}

		// Return results as JSON
		return c.JSON(sr.Entries)
	})
}
