package handlers

import (
	"net/http"

	"github.com/go-ldap/ldap/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"usrmgr/auth"
	"usrmgr/config"
	"usrmgr/util"
)

type DeleteUserRequest struct {
	Username string `json:"username"`
}

// RegisterDeleteUserHandler registers the delete user route with the API
func RegisterDeleteUserHandler(router fiber.Router, config config.LDAPConfig) {
	router.Delete("/user", func(c fiber.Ctx) error {
		claims := c.Context().Value(auth.TokenClaimsKey).(jwt.MapClaims)

		// Check if the user has permission to delete users
		groups, ok := claims["groups"]
		hasAdmin := false
		if ok {
			groupsInterface, ok := groups.([]any)
			if ok {
				for _, group := range groupsInterface {
					if groupStr, ok := group.(string); ok && groupStr == "admins" {
						hasAdmin = true
						break
					}
				}
			}
		}
		if !hasAdmin {
			util.LogWarning("Unauthorized delete user attempt", logrus.Fields{
				"remoteIP": c.IP(),
				"method":   c.Method(),
				"path":     c.Path(),
				"userID":   claims["sub"],
			})
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden",
			})
		}
		req := new(DeleteUserRequest)

		util.LogInfo("Handling delete user request", logrus.Fields{
			"remoteIP": c.IP(),
			"method":   c.Method(),
			"path":     c.Path(),
			"body":     string(c.Body()),
		})

		if err := c.Bind().Body(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Username == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Username is required",
			})
		}

		conn, err := ldap.DialURL(config.Host)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP connection failed",
			})
		}
		defer conn.Close()

		if err := conn.Bind(config.BindDN, config.Password); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP bind failed",
			})
		}

		// Search for the user DN
		userFilter := "(uid=" + ldap.EscapeFilter(req.Username) + ")"
		searchRequest := ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0, 0, false,
			userFilter,
			[]string{"dn"},
			nil,
		)
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "User search failed",
			})
		}
		if len(sr.Entries) == 0 {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		userDN := sr.Entries[0].DN
		deleteReq := ldap.NewDelRequest(userDN, nil)
		if err := conn.Del(deleteReq); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete user: " + err.Error(),
			})
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "User deleted successfully",
		})
	})
}
