package imgToRtp

import (
	"bytes"
	"dm/utils/imgToRtp/rtp"
	"errors"
	"github.com/gen2brain/x264-go"
	"image"
	"time"
)

func ImgToRtp(imgData image.Image) (rtpData []byte, err error) {

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
		return nil, errors.New("解析图片数据..." + err.Error())
	}

	err = enc.Encode(imgData)
	if err != nil {
		return nil, errors.New("格式化图片数据..." + err.Error())
	}

	err = enc.Flush()
	if err != nil {
		return nil, errors.New("刷新x264工具..." + err.Error())
	}

	err = enc.Close()
	if err != nil {
		return nil, errors.New("关闭x264工具..." + err.Error())
	}

	//保存RTP数据
	rtpBuf := bytes.NewBuffer(make([]byte, 0))
	rtpPacket := rtp.NewDefaultPacketWithH264Type()
	nalus := PackH264FrameToNalus(buf.Bytes())

	//e := &rtph264.Encoder{
	//	PayloadType: 96,
	//}
	//e.Init()
	//
	//pkts, err := e.Encode(nalus, 0)

	//for _, v := range pkts{
	//	var pb []byte
	//	v.MarshalTo(pb)
	//	rtpBuf.Write(pb)
	//}

	for _, v := range nalus {
		rps := rtpPacket.ParserNaluToRtpPayload(v)

		// H264 30FPS : 90000 / 30 : diff = 3000
		rtpPacket.SetTimeStamp(rtpPacket.TimeStamp() + 3000)

		for _, q := range rps {
			rtpPacket.SetSequence(rtpPacket.Sequence() + 1)
			rtpPacket.SetPayload(q)

			rtpBuf.Write(rtpPacket.GetRtpBytes())

			//远程发送
			//conn.WriteToUDP(rtpPacket.GetRtpBytes(), &net.UDPAddr{IP: net.ParseIP("192.168.0.78"), Port: 1236})
		}

		time.Sleep(30 * time.Millisecond)
	}

	return rtpBuf.Bytes(), nil
}

func PackH264FrameToNalus(bytes []byte) [][]byte {
	l := len(bytes)
	var startPos []int
	var nalus [][]byte
	j := 0 // split nalu in bytes to nalus
	for i := 0; i < l-5; i++ {
		if bytes[i] == 0 && bytes[i+1] == 0 && bytes[i+2] == 1 {
			if i > 0 && bytes[i-1] == 0 { //parameter set startpos
				startPos = append(startPos, i-1)
			} else {
				startPos = append(startPos, i)
			}
			j++
			if j > 1 {
				b := bytes[startPos[j-2]:startPos[j-1]]
				nalus = append(nalus, b)
			}
		}
	}
	nalus = append(nalus, bytes[startPos[j-1]:])
	if len(nalus) != len(startPos) {
		panic("unknown error at split nalu in bytes to nalus ")
	}

	return nalus
}
