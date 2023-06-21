package Model

// H264Packet 是表示 H.264 数据包的结构体
type H264Packet struct {
	Index string //编号
	Data  []byte // H.264 数据
}
