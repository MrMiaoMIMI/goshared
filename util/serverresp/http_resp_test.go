package serverresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MrMiaoMIMI/goshared/util/servererr"
	"github.com/gin-gonic/gin"
)

func TestSuccessPageTypedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	SuccessPage(ctx, []string{"a", "b"}, 2, 1, 20)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp Response[PageData[string]]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != servererr.Success || resp.Data.Total != 2 || len(resp.Data.List) != 2 {
		t.Fatalf("response = %#v", resp)
	}
}

func TestErrorUsesBizError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	Error(ctx, servererr.Wrap(servererr.ErrBadRequest, "bad input", errors.New("missing field")))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}

	var resp Response[any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != servererr.ErrBadRequest || resp.Message != "bad input" {
		t.Fatalf("response = %#v", resp)
	}
}
