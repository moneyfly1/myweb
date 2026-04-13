package middleware

import (
	"compress/gzip"
	"io"
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

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type compressResponseWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *compressResponseWriter) WriteString(s string) (int, error) {
	return w.writer.Write([]byte(s))
}

// CompressionMiddleware Brotli 优先，fallback Gzip，互斥不会双重压缩
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ae := c.GetHeader("Accept-Encoding")

		// Skip WebSocket
		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			c.Next()
			return
		}

		if strings.Contains(ae, "br") {
			bw := brotliWriterPool.Get().(*brotli.Writer)
			bw.Reset(c.Writer)

			c.Header("Content-Encoding", "br")
			c.Header("Vary", "Accept-Encoding")
			c.Writer.Header().Del("Content-Length")

			c.Writer = &compressResponseWriter{ResponseWriter: c.Writer, writer: bw}
			c.Next()

			bw.Close()
			brotliWriterPool.Put(bw)
			return
		}

		if strings.Contains(ae, "gzip") {
			gw := gzipWriterPool.Get().(*gzip.Writer)
			gw.Reset(c.Writer)

			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")
			c.Writer.Header().Del("Content-Length")

			c.Writer = &compressResponseWriter{ResponseWriter: c.Writer, writer: gw}
			c.Next()

			gw.Close()
			gzipWriterPool.Put(gw)
			return
		}

		c.Next()
	}
}
