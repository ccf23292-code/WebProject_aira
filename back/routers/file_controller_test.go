package routers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildPublicURLUsesRequestHost(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Host = "localhost:3001"
	req.Header.Set("X-Forwarded-Proto", "https")
	ctx.Request = req

	got := buildPublicURL(ctx, "/static/avatars/a.png")
	want := "https://localhost:3001/static/avatars/a.png"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFileControllerUploadStoresFileAndReturnsURL(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("type", "image"); err != nil {
		t.Fatalf("write type field: %v", err)
	}

	part, err := writer.CreateFormFile("file", "cover.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	expectedContent := []byte("fake-image-content")
	if _, err := part.Write(expectedContent); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	engine := gin.New()
	group := engine.Group("/api/files")
	NewFileController().RegisterRoutes(group)

	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", body)
	req.Host = "127.0.0.1:3001"
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Type       string `json:"type"`
			PublicPath string `json:"public_path"`
			URL        string `json:"url"`
			Filename   string `json:"filename"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Data.Type != "image" {
		t.Fatalf("expected upload type image, got %q", resp.Data.Type)
	}
	if !strings.HasPrefix(resp.Data.PublicPath, "/static/uploads/images/") {
		t.Fatalf("unexpected public path: %q", resp.Data.PublicPath)
	}
	if !strings.HasPrefix(resp.Data.URL, "http://127.0.0.1:3001/static/uploads/images/") {
		t.Fatalf("unexpected url: %q", resp.Data.URL)
	}

	relativePath := strings.TrimPrefix(resp.Data.PublicPath, "/static/")
	savedPath := filepath.Join(tempDir, defaultStorageRoot, filepath.FromSlash(relativePath))
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if _, err := io.Copy(io.Discard, bytes.NewReader(content)); err != nil {
		t.Fatalf("copy saved file content: %v", err)
	}
	if string(content) != string(expectedContent) {
		t.Fatalf("expected saved content %q, got %q", string(expectedContent), string(content))
	}
}
