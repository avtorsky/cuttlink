package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

type gzipReader struct {
	http.Request
	Reader io.Reader
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func compressMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") {
			ctx.Next()
			return
		}

		gz, err := gzip.NewWriterLevel(ctx.Writer, gzip.BestCompression)
		if err != nil {
			io.WriteString(ctx.Writer, err.Error())
			return
		}
		defer gz.Close()

		ctx.Writer.Header().Set("Content-Encoding", "gzip")
		ctx.Writer = &gzipWriter{ctx.Writer, gz}
		ctx.Next()
	}
}

func decompressMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !strings.Contains(ctx.Request.Header.Get("Content-Encoding"), "gzip") {
			ctx.Next()
			return
		}

		gz, err := gzip.NewReader(ctx.Request.Body)
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx.Request.Body = gz
		ctx.Next()
	}
}
