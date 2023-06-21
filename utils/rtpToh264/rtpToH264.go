package rtpToh264

import "dm/utils/rtpToh264/parser/h264"

func RtpParser(RtpData []byte) (h264Data []byte) {

	// H264 解析器
	h264Parser := h264.NewParser()

	rtpParser := NewRtpParser()
	rtpPack := rtpParser.Parse(RtpData)
	h264Real := rtpPack.RealData()

	// 解析 h264
	h264Parser.WriteByte(h264Real)

	return h264Real
}
