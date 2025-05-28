package api

import (
	"bufio"
	"context"
	"net/http"
	"proxyllama/auth"
	pxcx "proxyllama/context"
	"proxyllama/models"
	"proxyllama/proxy"
	"proxyllama/util"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// RegisterOllamaRoutes adds Ollama-related endpoints to the app
func RegisterOllamaRoutes(app *fiber.App) {
	app.Post("/api/chat", ReverseProxyHandler)
}

// ReverseProxyHandler forwards the request to Ollama and streams the response back to the client
func ReverseProxyHandler(c *fiber.Ctx) error {
	// Parse the incoming request
	var chatReq models.OllamaChatReq
	if err := c.BodyParser(&chatReq); err != nil {
		return handleError(err, fiber.StatusBadRequest, "Invalid request body")
	}

	// Always set streaming to true to prevent timeouts
	chatReq.Stream = true

	uid := c.UserContext().Value(auth.UserIDKey).(string)
	// Get or create conversation context
	cc, err := pxcx.GetCachedConversation(uid, *chatReq.ConversationId)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to process conversation")
	}

	headers := proxy.ContextHeaders{}
	c.Request().Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)
		if keyStr != "Host" && keyStr != "Connection" {
			headers[keyStr] = valueStr
		}
	})

	c.Context().SetUserValue(proxy.ReqHeadersKey, headers)

	ollamaReqBody, err := cc.PrepareOllamaRequest(c.Context(), chatReq)
	if err != nil {
		return handleError(err, fiber.StatusInternalServerError, "Failed to prepare Ollama request")
	}

	handler, statusCode, err := proxy.GetProxyHandler(c.Context(), ollamaReqBody, c.Path(), http.MethodPost, true, func() *models.OllamaChatResp {
		return &models.OllamaChatResp{}
	})
	if err != nil {
		return handleError(err, fiber.StatusBadGateway, "Error during streaming")
	}
	c.Status(statusCode)
	var res string

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		res, err = handler(w)
		if err != nil {
			handleError(err, fiber.StatusInternalServerError, "Error during handler execution")
			return
		}

		// Only store the assistant response if there was no context error
		if res != "" {
			// Apply refinement steps to the response in a separate goroutine
			// to avoid keeping the client connection open longer than necessary
			// go func(response, userID string, convID int) {
			// 	pxcx.RefineResponse(response, userMessage, userID, convID)
			// }(res, cc.UserID, cc.ConversationID)

			go func(r string, cctx *pxcx.ConversationContext) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
				defer cancel()
				_, err := cc.AddAssistantMessage(ctx, res)
				if err != nil {
					util.HandleError(err)
				} else {
					util.LogInfo("Successfully stored assistant message in conversation", logrus.Fields{"conversationId": cc.ConversationID})
				}
			}(res, cc)
		} else if res == "" {
			util.LogWarning("Empty assistant response, not storing")
		}

		// Log the completion of the chat streaming
		util.LogInfo("Chat streaming complete", logrus.Fields{
			"conversationId": cc.ConversationID,
			"userId":         cc.UserID,
		})
	})

	return nil
}
