package parser

var (
	videoExts = []string{
		".mp4", ".mov", ".avi", ".mkv",
		".webm", ".flv", ".wmv",
	}

	imageExts = []string{
		".jpg", ".jpeg", ".png", ".gif",
		".bmp", ".webp", ".tiff",
	}

	keywords = map[string]TokenKind{
		"set":            TokenSet,
		"push":           TokenPush,
		"use":            TokenUse,
		"on":             TokenOn,
		"export":         TokenExport,
		"trim":           TokenTrim,
		"thumbnail_from": TokenThumbnailFrom,
		"concat":         TokenConcat,
		"process":        TokenProcess,
		"true":           TokenBool,
		"false":          TokenBool,
	}
)
