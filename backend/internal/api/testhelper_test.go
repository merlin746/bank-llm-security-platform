package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/mq"
)

// ==================== 测试公共脚手架 ====================

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestEngine 构造一个与生产路由完全一致的 gin 引擎。
//
// 所有外部依赖（Redis / RabbitMQ）均传 nil：生产代码对 nil 依赖做了降级处理
// （见 cmd/server/main.go 与各 handler 中的 nil 判断），这正好让测试无需任何
// 中间件即可覆盖路由注册与 handler 逻辑。
func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	demoUsers.reset()

	h := NewHandler(nil, nil, nil, nil, (*mq.MQClient)(nil), "http://127.0.0.1:8000")
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

// apiResponse 对应 model.APIResponse 的统一响应结构。
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Msg     string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
}

// doRequest 发起一次请求并返回 recorder 与解析后的统一响应体。
func doRequest(
	t *testing.T,
	r *gin.Engine,
	method, path string,
	body interface{},
) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body failed: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp apiResponse
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response failed: %v (body=%s)", err, w.Body.String())
		}
	}
	return w, resp
}

// decodeData 将统一响应中的 data 反序列化到 dest。
func decodeData(t *testing.T, resp apiResponse, dest interface{}) {
	t.Helper()

	if len(resp.Data) == 0 {
		t.Fatalf("response contains no data field (code=%d msg=%s)", resp.Code, resp.Message)
	}
	if err := json.Unmarshal(resp.Data, dest); err != nil {
		t.Fatalf("decode data failed: %v (data=%s)", err, string(resp.Data))
	}
}

// assertStatus 断言响应状态码。
func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("unexpected HTTP status: got %d, want %d (body=%s)", w.Code, want, w.Body.String())
	}
}

// assertSuccess 断言 HTTP 200 且业务码为 0。
func assertSuccess(t *testing.T, w *httptest.ResponseRecorder, resp apiResponse) {
	t.Helper()

	assertStatus(t, w, http.StatusOK)
	if resp.Code != 0 {
		t.Fatalf("expected business code 0, got %d (msg=%s)", resp.Code, resp.Message)
	}
}

// doRequestRaw 发送原始字符串请求体（用于构造非法 JSON 等边界场景）。
func doRequestRaw(
	t *testing.T,
	r *gin.Engine,
	method, path, rawBody string,
) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(rawBody)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp apiResponse
	if w.Body.Len() > 0 {
		// 非法请求体下响应体本身仍是合法 JSON，但这里不强制解析失败即测试失败
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w, resp
}

// contains 判断 s 是否包含 substr。
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
