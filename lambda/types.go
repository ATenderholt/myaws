package lambda

import (
	"io"
	"log"
)

type LayerVersionContentInput struct {
	S3Bucket        *string
	S3Key           *string
	S3ObjectVersion *string
	ZipFile         []byte
}

type Architecture string
type Runtime string

type PublishLayerVersionBody struct {
	Content                 LayerVersionContentInput
	LayerName               *string
	CompatibleArchitectures []Architecture
	CompatibleRuntimes      []Runtime
	Description             *string
	LicenseInfo             *string
}

type ZipContent struct {
	Offset  int64
	Content []byte
	Length  int64
}

func min(a int, b int64) int64 {
	a64 := int64(a)
	if a64 < b {
		return a64
	}

	return b
}

func (source ZipContent) ReadAt(p []byte, off int64) (n int, err error) {
	log.Printf("Attempting to read %d bytes from offset %d", len(p), off)

	if off >= source.Length {
		return 0, io.EOF
	}

	bytesToRead := min(len(p), source.Length-off)
	count := copy(p, source.Content[off:off+bytesToRead])

	if count < len(p) {
		return count, io.EOF
	}

	return count, nil
}
