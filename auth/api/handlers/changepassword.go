package handlers

import (
	"net/http"

	"github.com/go-ldap/ldap/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"

	"usrmgr/config"
	"usrmgr/util"
)

type PasswordChangeRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// RegisterChangePasswordHandler registers the password change route with the API
func RegisterChangePasswordHandler(router fiber.Router, config config.LDAPConfig) {
	router.Post("/change-password", func(c fiber.Ctx) error {
		pcr := new(PasswordChangeRequest)

		util.LogInfo("Handling password change request", logrus.Fields{
			"remoteIP": c.IP(),
			"method":   c.Method(),
			"path":     c.Path(),
			"body":     string(c.Body()),
		})

		if err := c.Bind().Body(pcr); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		util.LogInfo("Received password change request", logrus.Fields{
			"username":    pcr.Username,
			"oldPassword": pcr.OldPassword,
			"newPassword": pcr.NewPassword,
		})

		// Validate request
		if pcr.Username == "" || pcr.OldPassword == "" || pcr.NewPassword == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Username, old password, and new password are required",
			})
		}

		// Connect to LDAP
		conn, err := ldap.DialURL(config.Host)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP connection failed",
			})
		}
		defer conn.Close()

		// First, bind as admin to search for the user
		err = conn.Bind(config.BindDN, config.Password)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP bind failed",
			})
		}

		// Search for the user
		userFilter := "(uid=" + ldap.EscapeFilter(pcr.Username) + ")"
		searchRequest := ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
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

		if len(sr.Entries) != 1 {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found or multiple users found",
			})
		}

		userDN := sr.Entries[0].DN

		// Connect and attempt to bind as the user with the old password to verify identity
		connUser, err := ldap.DialURL(config.Host)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP connection failed",
			})
		}
		defer connUser.Close()

		// Verify old password works
		err = connUser.Bind(userDN, pcr.OldPassword)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Current password is incorrect",
			})
		}

		// User authenticated successfully, now perform password modification as admin
		// Create a password modify request
		passwordModifyRequest := ldap.NewPasswordModifyRequest(
			userDN,
			pcr.OldPassword,
			pcr.NewPassword,
		)

		// Connect as admin to update the password
		connAdmin, err := ldap.DialURL(config.Host)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP connection failed for password update",
			})
		}
		defer connAdmin.Close()

		err = connAdmin.Bind(config.BindDN, config.Password)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "LDAP bind failed for password update",
			})
		}

		_, err = connAdmin.PasswordModify(passwordModifyRequest)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update password: " + err.Error(),
			})
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Password updated successfully",
		})
	})
}
