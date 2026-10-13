package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/gif"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func avatarGIFData(t *testing.T, large bool) ([]byte, *gif.GIF) {
	t.Helper()
	palette := make(color.Palette, 256)
	palette[0] = color.Transparent
	for i := 1; i < len(palette); i++ {
		palette[i] = color.RGBA{R: uint8(i), G: uint8(255 - i), B: uint8(i / 2), A: 255}
	}
	size := 16
	if large {
		size = 640
	}
	src := &gif.GIF{
		Delay: []int{7, 13, 21, 9}, LoopCount: 3,
		Disposal: []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious, gif.DisposalNone},
		Config:   image.Config{ColorModel: palette, Width: size, Height: size},
	}
	rng := rand.New(rand.NewSource(42))
	for i := range src.Delay {
		bounds := image.Rect(0, 0, size, size)
		if i == 2 {
			bounds = image.Rect(size/4, size/4, size, size)
		}
		frame := image.NewPaletted(bounds, palette)
		for j := range frame.Pix {
			frame.Pix[j] = uint8(rng.Intn(256))
		}
		src.Image = append(src.Image, frame)
	}
	var buf bytes.Buffer
	require.NoError(t, gif.EncodeAll(&buf, src))
	return buf.Bytes(), src
}

func TestInlineAvatarGIFCompressionPreservesAnimation(t *testing.T) {
	raw, src := avatarGIFData(t, true)
	require.Greater(t, len(raw), targetAvatarBytes)
	require.Less(t, len(raw), maxInlineAvatarBytes)
	input, err := normalizeInlineUserAvatarInput("data:image/gif;base64," + base64.StdEncoding.EncodeToString(raw))
	require.NoError(t, err)
	require.Equal(t, "image/gif", input.ContentType)
	require.LessOrEqual(t, input.ByteSize, targetAvatarBytes)
	compressed, err := base64.StdEncoding.DecodeString(input.URL[len("data:image/gif;base64,"):])
	require.NoError(t, err)
	require.Len(t, compressed, input.ByteSize)
	sum := sha256.Sum256(compressed)
	require.Equal(t, hex.EncodeToString(sum[:]), input.SHA256)
	result, err := gif.DecodeAll(bytes.NewReader(compressed))
	require.NoError(t, err)
	require.Len(t, result.Image, len(src.Image))
	require.Equal(t, src.Delay, result.Delay)
	require.Equal(t, src.LoopCount, result.LoopCount)
	require.Equal(t, src.Disposal, result.Disposal)
	require.Equal(t, src.BackgroundIndex, result.BackgroundIndex)
	require.Less(t, result.Config.Width, src.Config.Width)
	require.Greater(t, result.Image[2].Rect.Min.X, 0)
	for i, frame := range result.Image {
		_, _, _, alpha := frame.Palette[0].RGBA()
		require.Zero(t, alpha)
		require.True(t, frame.Bounds().In(image.Rect(0, 0, result.Config.Width, result.Config.Height)))
		require.Greater(t, len(frame.Palette), 1)
		if i == 0 {
			require.Equal(t, src.Image[0].ColorIndexAt(0, 0), frame.ColorIndexAt(0, 0))
		}
	}
	require.NotEqual(t, result.Image[0].Pix, result.Image[1].Pix)
	// Confirm only the rendered GIF is persisted, not the original upload.
	require.NotEqual(t, raw, compressed)
}

func TestInlineAvatarSmallGIFPreservedAndMIMENormalized(t *testing.T) {
	raw, _ := avatarGIFData(t, false)
	for _, mime := range []string{"image/gif", "image/png", "image/GIF"} {
		t.Run(mime, func(t *testing.T) {
			input, err := normalizeInlineUserAvatarInput("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw))
			require.NoError(t, err)
			require.Equal(t, "image/gif", input.ContentType)
			require.Equal(t, "data:image/gif;base64,"+base64.StdEncoding.EncodeToString(raw), input.URL)
			require.Equal(t, len(raw), input.ByteSize)
		})
	}
}

func TestInlineAvatarGIFUploadSizeBoundary(t *testing.T) {
	raw, _ := avatarGIFData(t, false)
	// Padding after the trailer is legal to decode, and tests the exact upload
	// limit independently of the image content's compressibility.
	raw = append(raw, make([]byte, maxInlineAvatarBytes-len(raw))...)
	input, err := normalizeInlineUserAvatarInput("data:image/gif;base64," + base64.StdEncoding.EncodeToString(raw))
	require.NoError(t, err)
	require.LessOrEqual(t, input.ByteSize, targetAvatarBytes)
	raw = append(raw, 0)
	_, err = normalizeInlineUserAvatarInput("data:image/gif;base64," + base64.StdEncoding.EncodeToString(raw))
	require.ErrorIs(t, err, ErrAvatarTooLarge)
}

func TestInlineAvatarRejectsMalformedGIF(t *testing.T) {
	raw, _ := avatarGIFData(t, false)
	for _, data := range [][]byte{[]byte("not a GIF"), raw[:len(raw)-5], append([]byte(nil), raw[:13]...)} {
		_, err := normalizeInlineUserAvatarInput("data:image/gif;base64," + base64.StdEncoding.EncodeToString(data))
		require.ErrorIs(t, err, ErrAvatarInvalid)
	}
}

func TestAvatarGIFRejectsExcessiveDecodedDimensions(t *testing.T) {
	raw, _ := avatarGIFData(t, false)
	raw[6], raw[7], raw[8], raw[9] = 255, 255, 255, 255
	_, err := normalizeInlineUserAvatarInput("data:image/gif;base64," + base64.StdEncoding.EncodeToString(raw))
	require.ErrorIs(t, err, ErrAvatarTooLarge)
}

func TestAvatarGIFRejectsExcessiveFrameCountBeforeDecoding(t *testing.T) {
	raw, _ := avatarGIFData(t, false)
	animation, err := gif.DecodeAll(bytes.NewReader(raw))
	require.NoError(t, err)
	palette := animation.Image[0].Palette
	animation.Image = make([]*image.Paletted, maxAvatarGIFFrames+1)
	animation.Delay = make([]int, len(animation.Image))
	animation.Disposal = nil
	for i := range animation.Image {
		animation.Image[i] = image.NewPaletted(image.Rect(0, 0, 1, 1), palette)
	}
	var encoded bytes.Buffer
	require.NoError(t, gif.EncodeAll(&encoded, animation))
	require.ErrorIs(t, validateAvatarGIF(encoded.Bytes()), ErrAvatarTooLarge)
}
