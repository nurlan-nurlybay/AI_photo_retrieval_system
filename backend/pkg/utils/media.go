package utils

import "strings"

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
