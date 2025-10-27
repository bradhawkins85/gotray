//go:build windows

package menu

import (
	"bytes"
	"encoding/binary"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"github.com/example/gotray/internal/logging"
)

func platformNormalizeIcon(data []byte) []byte {
	if len(data) < 4 {
		return nil
	}

	if isICO(data) {
		return data
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		logging.Debugf("failed to decode tray icon image: %v", err)
		return nil
	}

	pngData := data
	if format != "png" {
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			logging.Debugf("failed to convert tray icon to png: %v", err)
			return nil
		}
		pngData = buf.Bytes()
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		logging.Debugf("tray icon image has invalid bounds: %dx%d", width, height)
		return nil
	}

	icoData, err := wrapPNGAsICO(pngData, width, height)
	if err != nil {
		logging.Debugf("failed to wrap tray icon PNG as ico: %v", err)
		return nil
	}

	logging.Debugf("normalized tray icon (%dx%d) from %s to ico container", width, height, format)
	return icoData
}

func wrapPNGAsICO(pngData []byte, width, height int) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}

	writeDimension := func(value int) error {
		size := byte(value)
		if value <= 0 || value >= 256 {
			size = 0
		}
		return buf.WriteByte(size)
	}

	if err := writeDimension(width); err != nil {
		return nil, err
	}
	if err := writeDimension(height); err != nil {
		return nil, err
	}

	if err := buf.WriteByte(0); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(0); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(32)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(pngData))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(6+16)); err != nil {
		return nil, err
	}

	if _, err := buf.Write(pngData); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func isICO(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00
}
