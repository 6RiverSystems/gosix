package ginmiddleware

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

func CaptureResponseBody(c *gin.Context, passThrough bool) (*bytes.Buffer, gin.ResponseWriter) {
	bc := &bodyCapture{
		ResponseWriter: c.Writer,
		passThrough:    passThrough,
		body:           &bytes.Buffer{},
	}
	c.Writer = bc
	return bc.body, bc.ResponseWriter
}

type bodyCapture struct {
	gin.ResponseWriter
	passThrough bool
	// could use io.MultiWriter here but not worth it
	body *bytes.Buffer
}

func (bc *bodyCapture) Write(b []byte) (int, error) {
	// Buffer.Write never returns an error
	bc.body.Write(b)
	if bc.passThrough {
		return bc.ResponseWriter.Write(b)
	} else {
		return len(b), nil
	}
}

func (bc *bodyCapture) WriteString(s string) (int, error) {
	// Buffer.Write never returns an error
	bc.body.WriteString(s)
	if bc.passThrough {
		return bc.ResponseWriter.WriteString(s)
	} else {
		return len(s), nil
	}
}
