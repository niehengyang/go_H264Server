package rtpToImg

import (
	log "github.com/sirupsen/logrus"
)

func UnpackRTP2H264(rtpPayload []byte) []byte {
	if len(rtpPayload) <= 0 {
		return nil
	}

	var out []byte
	fu_indicator := rtpPayload[0]                           //获取第一个字节
	fu_header := rtpPayload[1]                              //获取第二个字节
	nalu_type := fu_indicator & 0x1f                        //获取FU indicator的类型域
	flag := fu_header & 0xe0                                //获取FU header的前三位，判断当前是分包的开始、中间或结束
	nal_fua := ((fu_indicator & 0xe0) | (fu_header & 0x1f)) //FU_A nal
	var FrameType string
	if nal_fua == 0x67 {
		FrameType = "SPS"
	} else if nal_fua == 0x68 {
		FrameType = "PPS"
	} else if nal_fua == 0x65 {
		FrameType = "IDR"
	} else if nal_fua == 0x61 {
		FrameType = "P Frame"
	} else if nal_fua == 0x41 {
		FrameType = "P Frame"
	}
	log.Printf("nalu_type: %x flag: %x FrameType: %s", nalu_type, flag, FrameType)
	if nalu_type == 0x1c { //判断NAL的类型为0x1c=28，说明是FU-A分片
		if flag == 0x80 { //分片NAL单元开始位
			/*
			   o := make([]byte, len(rtpPayload)+5-2) //I帧开头可能为00 00 00 01、00 00 01，组帧时只用00 00 01开头
			   o[0] = 0x00
			   o[1] = 0x00
			   o[2] = 0x00
			   o[3] = 0x01
			   o[4] = nal_fua*/
			o := make([]byte, len(rtpPayload)+4-2) //I帧开头可能为00 00 00 01、00 00 01，组帧时只用00 00 01开头
			o[0] = 0x00
			o[1] = 0x00
			o[2] = 0x01
			o[3] = nal_fua
			copy(o[4:], rtpPayload[2:])
			out = o
		} else { //中间分片包或者最后一个分片包
			o := make([]byte, len(rtpPayload)-2)
			copy(o[0:], rtpPayload[2:])
			out = o
		}
	} else if nalu_type == 0x1 { //单一NAL 单元模式
		o := make([]byte, len(rtpPayload)+4) //将整个rtpPayload一起放进去
		o[0] = 0x00
		o[1] = 0x00
		o[2] = 0x00
		o[3] = 0x01
		copy(o[4:], rtpPayload[0:])
		out = o
	} else {
		log.Printf("Unsport nalu type!")
	}
	return out
}
