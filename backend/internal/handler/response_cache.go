package handler

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

// responseCacheCaptureWriter 包装 gin.ResponseWriter，在透传写出的同时把响应体
// 捕获到缓冲区（上限 responseCacheCaptureMaxBytes），用于非流式 Chat Completions
// 成功响应的缓存。所有写方法委托给底层 writer，行为与未包装时一致。
type responseCacheCaptureWriter struct {
	gin.ResponseWriter
	max int64
	buf bytes.Buffer
}

func (w *responseCacheCaptureWriter) Write(p []byte) (int, error) {
	if w.max > 0 && int64(w.buf.Len()) < w.max {
		remaining := w.max - int64(w.buf.Len())
		if int64(len(p)) > remaining {
			w.buf.Write(p[:remaining])
		} else {
			w.buf.Write(p)
		}
	}
	return w.ResponseWriter.Write(p)
}

func (w *responseCacheCaptureWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseCacheCaptureWriter) WriteHeaderNow() {
	w.ResponseWriter.WriteHeaderNow()
}

func (w *responseCacheCaptureWriter) Flush() {
	if f, ok := w.ResponseWriter.(gin.ResponseWriter); ok {
		f.Flush()
		return
	}
	if f, ok := w.ResponseWriter.(interface{ Flush() }); ok {
		f.Flush()
	}
}
