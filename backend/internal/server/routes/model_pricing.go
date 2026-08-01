package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterModelPricingRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, panelRateLimiter *middleware.PanelRateLimiter) {
	pricing := v1.Group("/model-pricing")
	pricing.Use(gin.HandlerFunc(jwtAuth))
	pricing.Use(panelRateLimiter.Global())
	pricing.GET("", h.ModelPlaza.GetPricing)
}
