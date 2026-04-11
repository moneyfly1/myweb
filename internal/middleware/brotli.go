package middleware

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

var brotliWriterPool = sync.Pool{
	New: func() interface{} {
		return brotli.NewWriterLevel(io.Discard, brotli.DefaultCompression)
	},
}

type brotliResponseWriter struct {
	gin.ResponseWriter
	writer *brotli.Writer
}

func (w *brotliResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *brotliResponseWriter) WriteString(s string) (int, error) {
	return w.writer.Write([]byte(s))
}

// BrotliMiddleware compresses responses with Brotli when the client supports it.
// Falls back to the next middleware (gzip) if the client doesn't send Accept-Encoding: br.
func BrotliMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "br") {
			c.Next()
			return
		}

		// Skip small or already-compressed content types
		ct := c.GetHeader("Content-Type")
		if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "video/") || strings.HasPrefix(ct, "audio/") {
			c.Next()
			return
		}

		bw := brotliWriterPool.Get().(*brotli.Writer)
		bw.Reset(c.Writer)
		defer func() {
			bw.Close()
			brotliWriterPool.Put(bw)
		}()

		c.Header("Content-Encoding", "br")
		c.Header("Vary", "Accept-Encoding")
		c.Writer.Header().Del("Content-Length")

		brw := &brotliResponseWriter{ResponseWriter: c.Writer, writer: bw}
		c.Writer = brw

		c.Next()

		// Ensure status is written
		if c.Writer.Status() == 0 {
			c.Writer.WriteHeader(http.StatusOK)
		}
	}
}
