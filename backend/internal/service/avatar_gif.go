package service

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/gif"
)

const (
	maxAvatarPixels         = 16 * 1024 * 1024
	maxAvatarGIFFramePixels = 64 * 1024 * 1024
	maxAvatarGIFFrames      = 1000
)

// Inspect image descriptors before DecodeAll allocates the frame buffers.
// Compressed byte size alone does not bound a GIF's decoded memory usage.
func validateAvatarGIF(data []byte) error {
	if len(data) < 13 || (string(data[:6]) != "GIF87a" && string(data[:6]) != "GIF89a") {
		return ErrAvatarInvalid
	}
	width := int(binary.LittleEndian.Uint16(data[6:8]))
	height := int(binary.LittleEndian.Uint16(data[8:10]))
	if width == 0 || height == 0 {
		return ErrAvatarInvalid
	}
	if int64(width)*int64(height) > maxAvatarPixels {
		return ErrAvatarTooLarge
	}
	offset := 13
	if data[10]&0x80 != 0 {
		offset += 3 * (1 << ((data[10] & 7) + 1))
	}
	frames, pixels := 0, int64(0)
	for offset < len(data) {
		block := data[offset]
		offset++
		switch block {
		case 0x3b: // Trailer.
			if frames == 0 {
				return ErrAvatarInvalid
			}
			return nil
		case 0x21: // Extension label followed by data sub-blocks.
			offset++
		case 0x2c: // Image descriptor and optional local color table.
			if offset+9 > len(data) {
				return ErrAvatarInvalid
			}
			frameWidth := int(binary.LittleEndian.Uint16(data[offset+4 : offset+6]))
			frameHeight := int(binary.LittleEndian.Uint16(data[offset+6 : offset+8]))
			frames++
			pixels += int64(frameWidth) * int64(frameHeight)
			if frames > maxAvatarGIFFrames || pixels > maxAvatarGIFFramePixels {
				return ErrAvatarTooLarge
			}
			packed := data[offset+8]
			offset += 9
			if packed&0x80 != 0 {
				offset += 3 * (1 << ((packed & 7) + 1))
			}
			offset++ // LZW minimum code size.
		default:
			return ErrAvatarInvalid
		}
		for {
			if offset >= len(data) {
				return ErrAvatarInvalid
			}
			size := int(data[offset])
			offset++
			if size == 0 {
				break
			}
			if size > len(data)-offset {
				return ErrAvatarInvalid
			}
			offset += size
		}
	}
	return ErrAvatarInvalid
}

// Resize indexed frames directly, preserving their palettes (including
// transparency), offsets, disposal modes, delays, frame count and loop count.
// In particular, do not flatten partial frames into a static JPEG.
func compressAvatarGIF(src *gif.GIF) ([]byte, error) {
	originalWidth, originalHeight := src.Config.Width, src.Config.Height
	scale := min(1.0, 512.0/float64(max(originalWidth, originalHeight)))
	for {
		width := max(1, int(float64(originalWidth)*scale))
		height := max(1, int(float64(originalHeight)*scale))
		out := *src
		out.Config.Width, out.Config.Height = width, height
		out.Image = make([]*image.Paletted, len(src.Image))
		for i, frame := range src.Image {
			bounds := frame.Bounds()
			// Map both edges into the same logical screen coordinate system.
			left := bounds.Min.X * width / originalWidth
			top := bounds.Min.Y * height / originalHeight
			right := min(width, (bounds.Max.X*width+originalWidth-1)/originalWidth)
			bottom := min(height, (bounds.Max.Y*height+originalHeight-1)/originalHeight)
			dst := image.NewPaletted(image.Rect(left, top, right, bottom), frame.Palette)
			for y := top; y < bottom; y++ {
				sy := bounds.Min.Y + (y-top)*bounds.Dy()/dst.Rect.Dy()
				for x := left; x < right; x++ {
					sx := bounds.Min.X + (x-left)*bounds.Dx()/dst.Rect.Dx()
					dst.SetColorIndex(x, y, frame.ColorIndexAt(sx, sy))
				}
			}
			out.Image[i] = dst
		}
		var encoded bytes.Buffer
		if err := gif.EncodeAll(&encoded, &out); err != nil {
			return nil, ErrAvatarInvalid
		}
		if encoded.Len() <= targetAvatarBytes {
			return encoded.Bytes(), nil
		}
		if width == 1 && height == 1 {
			return nil, ErrAvatarTooLarge
		}
		scale *= 0.75
	}
}
