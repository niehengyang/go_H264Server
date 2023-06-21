package rtpToImg

import (
	"image"
)

func RtpToImg(frameRtpData []byte) (image.Image, error) {
	frameData := UnpackRTP2H264(frameRtpData)
	//nalus, _ := SplitNALUs(frameData)

	// setup H264->raw frames decoder
	h264RawDec, err := NewH264Decoder()
	if err != nil {
		return nil, err
	}
	defer h264RawDec.Close()

	// convert NALUs into RGBA frames
	img, err := h264RawDec.Decode(frameData)
	if err != nil {
		return nil, err
	}

	// wait for a frame
	if img == nil {
		return nil, nil
	}

	return img, nil
}
