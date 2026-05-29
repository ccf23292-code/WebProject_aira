package services

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractError 用统一错误类型表达预处理失败，便于 IngestService 落库 error_message。
type extractError struct {
	Code    string
	Message string
}

func (e *extractError) Error() string { return e.Message }

func newExtractError(code, message string) *extractError {
	return &extractError{Code: code, Message: message}
}

// ExtractTextFromFile 把上传文件转为纯文本/Markdown，供下游 LLM 清洗。
//   - .md / .txt        → 原样读取
//   - .pdf              → ledongthuc/pdf 抽文本（仅适用于数字 PDF）
//   - .docx             → 解 zip 取 word/document.xml 文本节点
//   - .jpg/.jpeg/.png   → 调用 vision client（Qwen-VL）做 OCR
//
// 其它扩展名返回"unsupported"错误。
func ExtractTextFromFile(ctx context.Context, path string, vision *VisionClient) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt":
		return readPlainFile(path)
	case ".pdf":
		return extractPDF(path)
	case ".docx":
		return extractDOCX(path)
	case ".jpg", ".jpeg", ".png":
		if vision == nil || !vision.Enabled() {
			return "", newExtractError(
				"vision_disabled",
				"图片识别未启用：请联系管理员配置 LLM_VISION_API_KEY",
			)
		}
		return vision.ExtractFromImage(ctx, path)
	default:
		return "", newExtractError("unsupported_format", fmt.Sprintf("暂不支持该文件类型: %s", ext))
	}
}

func readPlainFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", newExtractError("read_failed", "读取文件失败: "+err.Error())
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", newExtractError("empty_content", "文件内容为空")
	}
	return text, nil
}

// extractPDF 仅适用于数字 PDF。扫描版（图片）PDF 应转为图片或 DOCX 后再上传。
func extractPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", newExtractError("pdf_open_failed", "PDF 打开失败: "+err.Error())
	}
	defer f.Close()

	var builder strings.Builder
	totalPages := r.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			// 单页失败不阻断整本，继续抽下一页
			continue
		}
		builder.WriteString(text)
		builder.WriteString("\n\n")
	}

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "", newExtractError(
			"pdf_no_text",
			"未能从 PDF 中提取文本，可能是扫描版图片，请转换为 DOCX 或图片上传",
		)
	}
	return result, nil
}

// extractDOCX 解 .docx (zip)，读 word/document.xml，遍历 <w:t> 文本节点并按段落拼接。
func extractDOCX(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", newExtractError("docx_open_failed", "DOCX 打开失败: "+err.Error())
	}
	defer zr.Close()

	var docFile *zip.File
	for _, file := range zr.File {
		if file.Name == "word/document.xml" {
			docFile = file
			break
		}
	}
	if docFile == nil {
		return "", newExtractError("docx_invalid", "未在 DOCX 中找到 document.xml")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", newExtractError("docx_read_failed", "DOCX 读取失败: "+err.Error())
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var builder strings.Builder
	var inText bool
	var inParagraph bool

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", newExtractError("docx_parse_failed", "DOCX 解析失败: "+err.Error())
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "t":
				inText = true
			case "br", "tab":
				builder.WriteString(" ")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					builder.WriteString("\n")
					inParagraph = false
				}
			case "t":
				inText = false
			}
		case xml.CharData:
			if inText {
				builder.Write(t)
			}
		}
	}

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "", newExtractError("docx_empty", "DOCX 文本为空")
	}
	return result, nil
}

// httpStatusForExtractError 将 extractError 映射为 HTTP 错误码。
// 用户输入造成的错误返回 400；服务端环境问题返回 500。
func httpStatusForExtractError(err error) int {
	var ee *extractError
	if !errors.As(err, &ee) {
		return http.StatusInternalServerError
	}
	switch ee.Code {
	case "unsupported_format", "empty_content", "pdf_no_text", "docx_invalid", "docx_empty":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
