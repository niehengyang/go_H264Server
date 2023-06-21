package router

import (
	"dm/service"
	"github.com/gin-gonic/gin"
)

func GinRouter(r *gin.Engine) *gin.Engine {
	rr := r.Group("/")

	// 注册WebSocket路由
	rr.GET("/ws", service.WebSocketHandler)

	rr.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "服务启动成功"})
	})

	rr = r.Group("/api")
	Routers(rr) //路由注册
	return r
}
