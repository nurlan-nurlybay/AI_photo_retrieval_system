package utils

import (
	"encoding/binary"
	"math"
	"net/url"
	"strings"
)

// ExtFromMime returns a file extension (without dot) from a MIME type
// Defaults to "bin" if unknown
func ExtFromMime(mime string) string {
	if mime == "" {
		return "bin"
	}

	mime = strings.ToLower(mime)
	switch mime {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/heic":
		return "heic"
	case "image/heif":
		return "heif"
	case "image/tiff":
		return "tiff"
	case "image/bmp":
		return "bmp"
	default:
		// fallback like "image/x-icon" to "icon"
		parts := strings.Split(mime, "/")
		if len(parts) == 2 {
			return strings.TrimPrefix(parts[1], "x-")
		}
		return "bin"
	}
}

func Float32ToBytes(vec32 []float32) []byte {
	buf := make([]byte, 4*len(vec32))
	for i, f := range vec32 {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func BytesToFloat32(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

const maxErrLen = 256 

func TruncateErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > maxErrLen {
		return msg[:maxErrLen]
	}
	return msg
}

func ExtractS3Key(mediaURL string) (string, error) {
	u, err := url.Parse(mediaURL)
	if err != nil {
		return "", err
	}

	// remove leading slash in u.Path
	key := strings.TrimPrefix(u.Path, "/") // u.path = "/media/404/ba94.../original.jpg"
	return key, nil
}
