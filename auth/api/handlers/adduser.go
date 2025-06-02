package handlers

import (
	"net/http"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"usrmgr/auth"
	"usrmgr/config"
	"usrmgr/util"
)

type AddUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	CN       string `json:"cn"`
	Mail     string `json:"mail"`
}

// RegisterAddUserHandler registers the add user route with the API
func RegisterAddUserHandler(router fiber.Router, config config.LDAPConfig) {
	router.Post("/user", func(c fiber.Ctx) error {
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
			util.LogWarning("Unauthorized add user attempt", logrus.Fields{
				"remoteIP": c.IP(),
				"method":   c.Method(),
				"path":     c.Path(),
				"userID":   claims["sub"],
			})
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden",
			})
		}
		req := new(AddUserRequest)

		util.LogInfo("Handling add user request", logrus.Fields{
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

		if req.Username == "" || req.Password == "" || req.CN == "" || req.Mail == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Username, password, cn, and mail are required",
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

		// Check if user already exists
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
		if len(sr.Entries) > 0 {
			return c.Status(http.StatusConflict).JSON(fiber.Map{
				"error": "User already exists",
			})
		}

		sn := req.CN // Using CN as surname for simplicity
		if strings.Contains(sn, " ") {
			parts := strings.SplitN(sn, " ", 2)
			sn = parts[1] // Take the second part as surname
		}

		userDN := "uid=" + req.Username + "," + config.BaseDN
		addReq := ldap.NewAddRequest(userDN, nil)
		addReq.Attribute("objectClass", []string{"inetOrgPerson", "organizationalPerson", "person", "top"})
		addReq.Attribute("uid", []string{req.Username})
		addReq.Attribute("cn", []string{req.CN})
		addReq.Attribute("sn", []string{sn}) // Using CN as surname for simplicity
		addReq.Attribute("mail", []string{req.Mail})
		addReq.Attribute("userPassword", []string{req.Password})

		if err := conn.Add(addReq); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to add user: " + err.Error(),
			})
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "User added successfully",
		})
	})
}
