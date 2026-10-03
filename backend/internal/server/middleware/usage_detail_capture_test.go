package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCaptureUsageRequestBodyRestoresFullBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := bytes.Repeat([]byte("a"), service.UsageDetailCaptureLimit+32)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))

	captured, truncated := captureUsageRequestBody(c)
	restored, err := io.ReadAll(c.Request.Body)

	require.NoError(t, err)
	require.True(t, truncated)
	require.Len(t, captured, service.UsageDetailCaptureLimit)
	require.Equal(t, body, restored)
}

func TestUsageDetailResponseWriterCapsCaptureWithoutTruncatingClientResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	writer := &usageDetailResponseWriter{ResponseWriter: c.Writer}
	body := bytes.Repeat([]byte("b"), service.UsageDetailCaptureLimit+32)

	written, err := writer.Write(body)
	captured, truncated := writer.captured()

	require.NoError(t, err)
	require.Equal(t, len(body), written)
	require.Equal(t, body, recorder.Body.Bytes())
	require.Len(t, captured, service.UsageDetailCaptureLimit)
	require.True(t, truncated)
}
