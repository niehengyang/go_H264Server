package utils

import (
	log "github.com/sirupsen/logrus"
	"image"
	"sync"
)

const (
	DEFAULT_CAP int = 1024 * 1024 * 4

	// NALU类型
	NALU_TYPE_SLICE    = 1
	NALU_TYPE_DPA      = 2
	NALU_TYPE_DPB      = 3
	NALU_TYPE_DPC      = 4
	NALU_TYPE_IDR      = 5
	NALU_TYPE_SEI      = 6
	NALU_TYPE_SPS      = 7
	NALU_TYPE_PPS      = 8
	NALU_TYPE_AUD      = 9
	NALU_TYPE_EOSEQ    = 10
	NALU_TYPE_EOSTREAM = 11
	NALU_TYPE_FILL     = 12

	// 优先级
	NALU_PRIORITY_DISPOSABLE = 0
	NALU_PRIRITY_LOW         = 1
	NALU_PRIORITY_HIGH       = 2
	NALU_PRIORITY_HIGHEST    = 3
)

type NaluHead struct {
	StartCodeLen int
	Forbidden    int
	Reference    int
	UnitType     int
}

type H264Buffer struct {
	mutex       sync.Mutex
	data        []byte
	dataMaxSize int
	rPtr        int
	naluHeads   []*NaluHead // NALU块信息
	iFrameIndex []int       // I帧索引列表
}

func NewH264Buffer(cap int) *H264Buffer {
	return &H264Buffer{
		data:        make([]byte, 0, cap),
		dataMaxSize: cap,
		rPtr:        0,
		naluHeads:   make([]*NaluHead, 0),
		iFrameIndex: make([]int, 0),
	}
}

func (buf *H264Buffer) Push(d []byte) {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	log.Info("H264Buffer push data")
	if len(d) > buf.dataMaxSize {
		return
	}
	// 计算超出的字节数
	over := len(buf.data) + len(d) - buf.dataMaxSize
	if over > 0 {
		// 左移超出的字节数
		buf.data = buf.data[over:]
		buf.rPtr = buf.rPtr - over
	}
	buf.data = append(buf.data, d...)
	// 更新nalu信息
	buf.updateNALU()
	log.Info("I frame indexs: ", buf.iFrameIndex)
}

func (buf *H264Buffer) getNALUHead(headIdx int) *NaluHead {
	return &NaluHead{
		Forbidden: int(buf.data[headIdx] >> 7),
		Reference: int((buf.data[headIdx] << 1) >> 6),
		UnitType:  int((buf.data[headIdx] << 3) >> 3),
	}
}

func (buf *H264Buffer) updateNALU() {
	buf.naluHeads = make([]*NaluHead, 0)
	buf.iFrameIndex = make([]int, 0)
	for i := 0; i < len(buf.data)-5; i++ {
		if buf.data[i] == 0 && buf.data[i+1] == 0 {
			if buf.data[i+2] == 1 {
				head := buf.getNALUHead(i + 3)
				head.StartCodeLen = 3
				// 起始码长度为3的情况
				buf.naluHeads = append(buf.naluHeads, head)
				if head.UnitType == NALU_TYPE_IDR {
					buf.iFrameIndex = append(buf.iFrameIndex, i)
				}
			}
			if buf.data[i+2] == 0 && buf.data[i+3] == 1 {
				head := buf.getNALUHead(i + 4)
				head.StartCodeLen = 4
				// 起始码长度为4的情况
				buf.naluHeads = append(buf.naluHeads, head)
				if head.UnitType == NALU_TYPE_IDR {
					buf.iFrameIndex = append(buf.iFrameIndex, i)
				}
			}
		}
	}
}

func (buf *H264Buffer) NaluInfo() {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	log.Info("|Index\t|\tForbidden\t|\tReference\t|\tUnitType|")
	for i, v := range buf.naluHeads {
		reference := "UNKNOW"
		switch v.Reference {
		case NALU_PRIORITY_DISPOSABLE:
			reference = "DISPOSABLE"
		case NALU_PRIRITY_LOW:
			reference = "LOW"
		case NALU_PRIORITY_HIGH:
			reference = "HIGH"
		case NALU_PRIORITY_HIGHEST:
			reference = "HIGHEST"
		}
		naluType := "UNKNOW"
		switch v.UnitType {
		case NALU_TYPE_SLICE:
			naluType = "SLICE"
		case NALU_TYPE_DPA:
			naluType = "DPA"
		case NALU_TYPE_DPB:
			naluType = "DPB"
		case NALU_TYPE_DPC:
			naluType = "DPC"
		case NALU_TYPE_IDR:
			naluType = "IDR"
		case NALU_TYPE_SEI:
			naluType = "SEI"
		case NALU_TYPE_SPS:
			naluType = "SPS"
		case NALU_TYPE_PPS:
			naluType = "PPS"
		case NALU_TYPE_AUD:
			naluType = "AUD"
		case NALU_TYPE_EOSEQ:
			naluType = "EOSEQ"
		case NALU_TYPE_EOSTREAM:
			naluType = "EOSTREAM"
		case NALU_TYPE_FILL:
			naluType = "FILL"
		}
		log.Info("|Index: ", i, "\t\t| StartCodeLen: ", v.StartCodeLen, "\t| Forbidden: ", v.Forbidden,
			"\t| Reference: ", reference, "\t| UnitType: ", naluType)

	}
}

func (buf *H264Buffer) Pop() []byte {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	d := buf.data[buf.rPtr:]
	buf.rPtr = len(buf.data) - 1
	return d
}

func (buf *H264Buffer) PopLastGOP() []byte {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	if len(buf.iFrameIndex) == 0 {
		log.Info("no I Frame")
		return make([]byte, 0)
	}
	idx := buf.iFrameIndex[len(buf.iFrameIndex)-1]
	d := buf.data[idx:]
	log.Info("GOP size: ", len(d))
	return d
}

func (buf *H264Buffer) Data() []byte {
	h264Data := buf.data
	return h264Data
}

func (buf *H264Buffer) ExtractFrameFromH264Nalu() ([][]byte, error) {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()

	var videoFrames [][]byte

	h264Data := buf.data
	naluHeads := buf.naluHeads
	iFrameIndex := buf.iFrameIndex
	var start, end int
	log.Info("NALU块信息量: ", len(naluHeads))
	for i := 0; i < len(iFrameIndex); i++ {
		end = iFrameIndex[i]
		var frameBuff = make([]byte, end-start)
		copy(frameBuff, h264Data[start:end])
		videoFrames = append(videoFrames, frameBuff)
		start = end
	}
	return videoFrames, nil
}

// 将 H.264 数据包解码为图片
func DecodeH264ToImg(h264Data []byte) (image.Image, error) {

	var frameImg image.Image

	return frameImg, nil
}
