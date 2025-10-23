package domain

import (
	"encoding/binary"
	"math"
	"net/url"
	"strings"
	"time"
)

type Embedding struct {
	MediaID   int64
	UserID    int64
	Model     string
	VecBytes  []byte
	Status    string
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Float32ToBytes converts a []float32 slice into a []byte using little-endian encoding.
func Float32ToBytes(vec32 []float32) []byte {
	buf := make([]byte, 4*len(vec32))
	for i, f := range vec32 {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// BytesToFloat32 converts a []byte back into a []float32 slice.
func BytesToFloat32(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

const maxErrLen = 256 // or whatever your DB field can handle

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
	key := strings.TrimPrefix(u.Path, "/") // u.path = "media/404/ba94.../original.jpg"
	return key, nil
}
