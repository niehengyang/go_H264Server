package controller

import (
	"bytes"
	"dm/utils/imgToRtp"
	"dm/utils/rtpToImg"
	"dm/utils/rtpToh264"
	"github.com/gin-gonic/gin"
	"image"
	"image/jpeg"
	"os"
)

func TestRtpToImg(c *gin.Context) {
	// 读取图片数据
	imgData, err := loadImage("./output/1683881609034.jpg")
	if err != nil {
		c.JSON(500, gin.H{"msg": "图片读取失败..." + err.Error()})
		return
	}

	//将图片转换成RTP数据
	resRtp, err := imgToRtp.ImgToRtp(imgData)
	if err != nil {
		c.JSON(500, gin.H{"msg": "Img转Rtp失败..." + err.Error()})
		return
	}

	//rtpParsePacket := rtpToImg2.NewRtpParsePacket()
	//
	////将rtp数据解析为H264
	//frameData, _ := rtpParsePacket.ReadRtp(resRtp)

	frameData := rtpToh264.RtpParser(resRtp)
	//rtpPkg := rtp.ParseRTPHeader(resRtp)
	//frameData := rtpToImg.UnpackRTP2H264(rtpPkg.Payload)

	// setup H264->raw frames decoder
	h264RawDec, err := rtpToImg.NewH264Decoder()
	if err != nil {
		c.JSON(500, gin.H{"msg": "解码器..." + err.Error()})
		return
	}
	defer h264RawDec.Close()

	//convert NALUs into RGBA frames
	img, err := h264RawDec.Decode(frameData)
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

// loadImage从文件加载图像
func loadImage(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}
