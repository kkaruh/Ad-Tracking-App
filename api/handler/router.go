package router

import (
	"adds_app/api/middleware"
	"adds_app/controller"

	"github.com/gin-gonic/gin"
)

func Routes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.GlobalErrorHandler())
	router.GET("/ads", controller.GetAds)
	router.POST("/ads/click", controller.PostAdds)
	router.GET("/ads/analytics/:id", controller.GetAnalytics)
	return router
}
