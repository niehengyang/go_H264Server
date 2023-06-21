package router

import (
	"dm/controller"
	"github.com/gin-gonic/gin"
)

func Routers(r *gin.RouterGroup) {
	rr := r.Group("")
	//rr.GET("/videoStreamToImg", controller.H264ToImg)

	rr.GET("/decodeH264", controller.DecodeH264)
	rr.GET("/goavH264ToImg", controller.GoavH264ToImg)
	rr.GET("/gortsplibH264", controller.GortsplibH264)
	rr.GET("/rtpToImg", controller.TestRtpToImg)
	return
}
