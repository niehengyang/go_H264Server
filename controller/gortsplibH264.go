package controller

import (
	"dm/utils/rtpToImg"
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"image"
	"image/jpeg"
	"os"
	"strconv"
	"time"
)

func saveToFile(img image.Image) error {
	// create file
	fname := "./output/" + strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + ".jpg"
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Println("saving", fname)

	// convert to jpeg
	return jpeg.Encode(f, img, &jpeg.Options{
		Quality: 60,
	})
}

func GortsplibH264(c *gin.Context) {

	rtpC := gortsplib.Client{}

	// parse URL
	u, err := url.Parse("rtsp://wowzaec2demo.streamlock.net/vod/mp4:BigBuckBunny_115k.mp4")
	if err != nil {
		c.JSON(500, gin.H{"msg": "视频地址解析失败" + err.Error()})
		panic(any(err))
	}

	// connect to the server
	err = rtpC.Start(u.Scheme, u.Host)
	if err != nil {
		c.JSON(500, gin.H{"msg": "视频连接失败" + err.Error()})
		panic(any(err))
	}
	defer rtpC.Close()

	// find published tracks
	tracks, baseURL, _, err := rtpC.Describe(u)
	if err != nil {
		c.JSON(500, gin.H{"msg": "find published tracks" + err.Error()})
		panic(any(err))
	}

	// find the H264 track
	track := func() *gortsplib.TrackH264 {
		for _, track := range tracks {
			if track, ok := track.(*gortsplib.TrackH264); ok {
				return track
			}
		}
		return nil
	}()
	if track == nil {
		c.JSON(500, gin.H{"msg": "H264 track not found"})
		panic(any(err))
	}

	// setup RTP/H264->H264 decoder
	rtpDec := track.CreateDecoder()

	// setup H264->raw frames decoder
	h264RawDec, err := rtpToImg.NewH264Decoder()
	if err != nil {
		c.JSON(500, gin.H{"msg": "帧解析失败" + err.Error()})
		panic(any(err))
	}
	defer h264RawDec.Close()

	// if SPS and PPS are present into the SDP, send them to the decoder
	sps := track.SafeSPS()
	if sps != nil {
		h264RawDec.Decode(sps)
	}
	pps := track.SafePPS()
	if pps != nil {
		h264RawDec.Decode(pps)
	}

	//called when a RTP packet arrives
	saveCount := 0
	rtpC.OnPacketRTP = func(ctx *gortsplib.ClientOnPacketRTPCtx) {
		// convert RTP packets into NALUs
		nalus, _, err := rtpDec.Decode(ctx.Packet)
		if err != nil {
			return
		}

		for _, nalu := range nalus {
			// convert NALUs into RGBA frames
			img, err := h264RawDec.Decode(nalu)
			if err != nil {
				c.JSON(500, gin.H{"msg": "NALUs解析失败" + err.Error()})
				panic(any(err))
			}

			// wait for a frame
			if img == nil {
				continue
			}

			// convert frame to JPEG and save to file
			err = saveToFile(img)
			if err != nil {
				c.JSON(500, gin.H{"msg": "图片保存失败" + err.Error()})
				panic(any(err))
			}

			saveCount++
			if saveCount == 100 {
				log.Printf("saved 100 images, exiting")
				os.Exit(1)
			}
		}
	}

	// setup and read the H264 track only
	err = rtpC.SetupAndPlay(gortsplib.Tracks{track}, baseURL)
	if err != nil {
		c.JSON(500, gin.H{"msg": "构建失败" + err.Error()})
		panic(any(err))
	}

	// wait until a fatal error
	panic(any(rtpC.Wait()))
}
