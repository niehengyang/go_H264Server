package controller

import (
	"bytes"
	"dm/utils/rtpToImg"
	"github.com/gen2brain/x264-go"
	"github.com/gin-gonic/gin"
	"image/jpeg"
)

func DecodeH264(c *gin.Context) {

	// 读取图片数据
	imgData, err := loadImage("./output/1683881609034.jpg")
	if err != nil {
		c.JSON(500, gin.H{"msg": "图片读取失败..." + err.Error()})
		return
	}

	// 创建x264编码器
	opts := &x264.Options{
		Width:     imgData.Bounds().Size().X,
		Height:    imgData.Bounds().Size().Y,
		FrameRate: 25,
		Tune:      "zerolatency",
		Preset:    "veryfast",
		Profile:   "baseline",
		LogLevel:  x264.LogDebug,
	}

	//保存H264数据
	buf := bytes.NewBuffer(make([]byte, 0))

	enc, err := x264.NewEncoder(buf, opts)
	if err != nil {
		c.JSON(500, gin.H{"msg": "解析图片数据..." + err.Error()})
		return
	}

	err = enc.Encode(imgData)
	if err != nil {
		c.JSON(500, gin.H{"msg": "格式化图片数据..." + err.Error()})
		return
	}

	err = enc.Flush()
	if err != nil {
		c.JSON(500, gin.H{"msg": "刷新x264工具..." + err.Error()})
		return
	}

	err = enc.Close()
	if err != nil {
		c.JSON(500, gin.H{"msg": "关闭x264工具..." + err.Error()})
		return
	}

	// setup H264->raw frames decoder
	h264RawDec, err := rtpToImg.NewH264Decoder()
	if err != nil {
		c.JSON(500, gin.H{"msg": "解码器..." + err.Error()})
		return
	}
	defer h264RawDec.Close()

	//convert NALUs into RGBA frames
	img, err := h264RawDec.Decode(buf.Bytes())
	if err != nil {
		c.JSON(500, gin.H{"msg": "解码..." + err.Error()})
		return
	}

	// wait for a frame
	if img == nil {
		c.JSON(500, gin.H{"msg": "未获取到图片..."})
		return
	}

	// 将图像数据编码为字节
	var imgBuf bytes.Buffer
	err = jpeg.Encode(&imgBuf, img, nil)
	if err != nil {
		c.JSON(500, gin.H{"msg": "将图像数据编码为字节..." + err.Error()})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", "inline; filename=image.jpg")

	// 将图像数据写入响应主体
	c.Writer.Write(imgBuf.Bytes())

}
