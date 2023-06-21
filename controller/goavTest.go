package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avdevice"
	"github.com/giorgisio/goav/avfilter"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/swresample"
	"github.com/giorgisio/goav/swscale"
	log "github.com/sirupsen/logrus"
	"os"
	"unsafe"
)

// SaveFrame writes a single frame to disk as a PPM file
func SaveFrame(frame *avutil.Frame, width, height, frameNumber int) {
	// Open file
	fileName := fmt.Sprintf("./output/frame%d.ppm", frameNumber)
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("Error Reading")
	}
	defer file.Close()

	// Write header
	header := fmt.Sprintf("P6\n%d %d\n255\n", width, height)
	file.Write([]byte(header))

	// Write pixel data
	for y := 0; y < height; y++ {
		data0 := avutil.Data(frame)[0]
		buf := make([]byte, width*3)
		startPos := uintptr(unsafe.Pointer(data0)) + uintptr(y)*uintptr(avutil.Linesize(frame)[0])
		for i := 0; i < width*3; i++ {
			element := *(*uint8)(unsafe.Pointer(startPos + uintptr(i)))
			buf[i] = element
		}
		file.Write(buf)
	}
}

// 将 H.264 数据包解码为图片
func GoavH264ToImg(c *gin.Context) {

	filename := "./static/simplest_mediadata_test_sintel.h264"

	// 加载ffmpeg的网络库
	avformat.AvRegisterAll()
	// 加载ffmpeg的编解码库
	avcodec.AvcodecRegisterAll()
	log.Printf("AvFilter Version:\t%v", avfilter.AvfilterVersion())
	log.Printf("AvDevice Version:\t%v", avdevice.AvdeviceVersion())
	log.Printf("SWScale Version:\t%v", swscale.SwscaleVersion())
	log.Printf("AvUtil Version:\t%v", avutil.AvutilVersion())
	log.Printf("AvCodec Version:\t%v", avcodec.AvcodecVersion())
	log.Printf("Resample Version:\t%v", swresample.SwresampleLicense())

	// 打开视频流
	formatCtx := openInput(filename)
	if formatCtx == nil {
		return
	}
	// 检索流信息
	success := findStreamInfo(formatCtx)
	if !success {
		return
	}
	fmt.Println("检索流信息： ", success)
	// 打印 ffmpeg 的日志
	formatCtx.AvDumpFormat(0, filename, 0)
	// 读取一帧视频帧
	avCodecCtx := findFirstVideoStreamCodecContext(formatCtx)
	if avCodecCtx == nil {
		fmt.Println("没有发现视频帧： ")
		return
	}
	// 查找并打开解码器
	codecCtx := findAndOpenCodec(avCodecCtx)
	if codecCtx == nil {
		fmt.Println("没有发现解码器，或解码器不可用： ")
		return
	}

	// Allocate video frame
	pFrame := avutil.AvFrameAlloc()

	// Allocate an AVFrame structure
	pFrameRGB := avutil.AvFrameAlloc()

	// Determine required buffer size and allocate buffer
	numBytes := uintptr(avcodec.AvpictureGetSize(avcodec.AV_PIX_FMT_RGB24, codecCtx.Width(), codecCtx.Height()))
	buffer := avutil.AvMalloc(numBytes)

	// Assign appropriate parts of buffer to image planes in pFrameRGB
	// Note that pFrameRGB is an AVFrame, but AVFrame is a superset
	// of AVPicture
	avp := (*avcodec.Picture)(unsafe.Pointer(pFrameRGB))
	avp.AvpictureFill((*uint8)(buffer), avcodec.AV_PIX_FMT_RGB24, codecCtx.Width(), codecCtx.Height())

	// initialize SWS context for software scaling
	swsCtx := swscale.SwsGetcontext(
		codecCtx.Width(),
		codecCtx.Height(),
		(swscale.PixelFormat)(codecCtx.PixFmt()),
		codecCtx.Width(),
		codecCtx.Height(),
		avcodec.AV_PIX_FMT_RGB24,
		avcodec.SWS_BILINEAR,
		nil,
		nil,
		nil,
	)

	for i := 0; i < int(formatCtx.NbStreams()); i++ {
		switch formatCtx.Streams()[i].CodecParameters().AvCodecGetType() {
		case avformat.AVMEDIA_TYPE_VIDEO:
			// 循环读取视频帧并解码成 rgb , 默认就是yuv数据
			packet := avcodec.AvPacketAlloc()
			frameNumber := 1
			for formatCtx.AvReadFrame(packet) >= 0 {
				// Is this a packet from the video stream?
				if packet.StreamIndex() == i {
					// Is this a packet from the video stream?
					// Decode video frame
					response := codecCtx.AvcodecSendPacket(packet)
					if response < 0 {
						fmt.Printf("Error while sending a packet to the decoder: %s\n", avutil.ErrorFromCode(response))
					}
					for response >= 0 {
						responseFrame := codecCtx.AvcodecReceiveFrame((*avcodec.Frame)(unsafe.Pointer(pFrame)))
						if responseFrame == avutil.AvErrorEAGAIN || responseFrame == avutil.AvErrorEOF {
							break
						} else if responseFrame < 0 {
							//fmt.Printf("Error while receiving a frame from the decoder: %s\n", avutil.ErrorFromCode(response))
							break
						}
						// 从原生数据转换成RGB， 转换 5 个视频帧
						// Convert the image from its native format to RGB
						if frameNumber <= 5 {
							swscale.SwsScale2(swsCtx, avutil.Data(pFrame),
								avutil.Linesize(pFrame), 0, codecCtx.Height(),
								avutil.Data(pFrameRGB), avutil.Linesize(pFrameRGB))

							// 保存到本地硬盘
							fmt.Printf("Writing frame %d\n", frameNumber)
							SaveFrame(pFrameRGB, codecCtx.Width(), codecCtx.Height(), frameNumber)
						} else {
							return
						}
						frameNumber++
					}
					// 释放资源
					// Free the packet that was allocated by av_read_frame
					packet.AvFreePacket()
				}
			}

			// Free the RGB image
			avutil.AvFree(buffer)
			avutil.AvFrameFree(pFrameRGB)

			// Free the YUV frame
			avutil.AvFrameFree(pFrame)

			// Close the codecs
			codecCtx.AvcodecClose()
			(*avcodec.Context)(unsafe.Pointer(avCodecCtx)).AvcodecClose()

			// Close the video file
			formatCtx.AvformatCloseInput()
		default:
			fmt.Println("Didn't find a video stream")
		}
	}
}

/*
*
打开视频流
*/
func openInput(filename string) *avformat.Context {
	//
	formatCtx := avformat.AvformatAllocContext()
	// 打开视频流
	if avformat.AvformatOpenInput(&formatCtx, filename, nil, nil) != 0 {
		fmt.Printf("Unable to open file %s\n", filename)
		return nil
	}
	return formatCtx

}

/*
*
检索流信息
*/
func findStreamInfo(ctx *avformat.Context) bool {
	if ctx.AvformatFindStreamInfo(nil) < 0 {
		log.Println("Error: Couldn't find stream information.")
		// 关闭文件，释放 媒体文件/流的上下文
		ctx.AvformatCloseInput()
		return false
	}
	return true
}

/*
*
获取第一帧视频位置
*/
func findFirstVideoStreamIndex(ctx *avformat.Context) int {
	videoStreamIndex := -1
	for index, stream := range ctx.Streams() {
		switch stream.CodecParameters().AvCodecGetType() {
		case avformat.AVMEDIA_TYPE_VIDEO:
			return index
		}
	}
	return videoStreamIndex
}

/*
*
读取一帧视频帧
*/
func findFirstVideoStreamCodecContext(ctx *avformat.Context) *avformat.CodecContext {
	for _, stream := range ctx.Streams() {
		switch stream.CodecParameters().AvCodecGetType() {
		case avformat.AVMEDIA_TYPE_VIDEO:
			return stream.Codec()
		}
	}
	return nil
}

/*
*
根据索引获取视频帧
*/
func findVideoStreamCodecContext(ctx *avformat.Context, videoStreamIndex int) *avformat.CodecContext {
	if videoStreamIndex >= 0 {
		streams := ctx.Streams()
		stream := streams[videoStreamIndex]
		return stream.Codec()
	}
	return nil
}

/*
*
查找并打开编解码器
*/
func findAndOpenCodec(codecCtx *avformat.CodecContext) *avcodec.Context {
	codec := avcodec.AvcodecFindDecoder(avcodec.CodecId(codecCtx.GetCodecId()))
	if codec == nil {
		fmt.Println("Unsupported codec!")
		return nil
	}
	codecContext := codec.AvcodecAllocContext3()
	if codecContext.AvcodecCopyContext((*avcodec.Context)(unsafe.Pointer(codecCtx))) != 0 {
		fmt.Println("Couldn't copy codec context")
		return nil
	}
	if codecContext.AvcodecOpen2(codec, nil) < 0 {
		fmt.Println("Could not open codec")
		return nil
	}
	return codecContext
}
