package main

import (
	"dm/router"
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	S := gin.Default()
	RR := router.GinRouter(S)
	err := RR.Run(":8901")
	if err != nil {
		fmt.Println("服务器启动失败！")
	}
}
