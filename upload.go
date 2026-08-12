package tuitui

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const MaxUploadBytes = 100 * 1024 * 1024

type UploadData struct {
	Data        []byte
	Filename    string
	ContentType string
}
type UploadOptions struct {
	Filename    string
	ContentType string
}
type UploadResult struct {
	FID       string
	Filename  string
	FileSize  int
	MediaType string
}

type uploader struct {
	http   *httpAPI
	config resolvedConfig
}

type preparedUpload struct {
	data        []byte
	filename    string
	contentType string
}

func (u *uploader) upload(ctx context.Context, source any, options *UploadOptions) (UploadResult, error) {
	prepared, err := u.prepare(ctx, source, options)
	if err != nil {
		return UploadResult{}, err
	}
	if len(prepared.data) > MaxUploadBytes {
		return UploadResult{}, fmt.Errorf("[tuitui] file too large: %.2fMB > 100.00MB limit", float64(len(prepared.data))/1024/1024)
	}
	mediaType := detectUploadMediaType(prepared.contentType, prepared.filename)
	response, err := u.http.postMultipart(ctx, "/media/upload?type="+mediaType, prepared.filename, prepared.contentType, prepared.data)
	if err != nil {
		return UploadResult{}, err
	}
	fid, _ := response["media_id"].(string)
	if fid == "" {
		return UploadResult{}, fmt.Errorf("[tuitui] upload response is missing media_id")
	}
	return UploadResult{FID: fid, Filename: prepared.filename, FileSize: len(prepared.data), MediaType: mediaType}, nil
}

func (u *uploader) prepare(ctx context.Context, source any, options *UploadOptions) (preparedUpload, error) {
	resolved := UploadOptions{}
	if options != nil {
		resolved = *options
	}
	switch source := source.(type) {
	case UploadData:
		filename := firstNonEmpty(resolved.Filename, source.Filename, generatedFilename())
		return preparedUpload{append([]byte(nil), source.Data...), filename, firstNonEmpty(resolved.ContentType, source.ContentType, mimeType(filename))}, nil
	case *UploadData:
		if source == nil {
			return preparedUpload{}, fmt.Errorf("[tuitui] upload source is required")
		}
		return u.prepare(ctx, *source, options)
	case []byte:
		filename := firstNonEmpty(resolved.Filename, generatedFilename())
		return preparedUpload{append([]byte(nil), source...), filename, firstNonEmpty(resolved.ContentType, mimeType(filename))}, nil
	case string:
		if strings.HasPrefix(source, "data:") {
			data, contentType, err := decodeDataURL(source)
			if err != nil {
				return preparedUpload{}, err
			}
			extension := strings.TrimPrefix(filepath.Ext("x."+strings.TrimPrefix(contentType, "image/")), ".")
			if extension == "" {
				extension = "bin"
			}
			return preparedUpload{data, firstNonEmpty(resolved.Filename, fmt.Sprintf("media_%d.%s", time.Now().UnixMilli(), extension)), firstNonEmpty(resolved.ContentType, contentType)}, nil
		}
		if strings.HasPrefix(strings.ToLower(source), "http://") || strings.HasPrefix(strings.ToLower(source), "https://") {
			return u.prepareRemote(ctx, source, resolved)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return preparedUpload{}, fmt.Errorf("[tuitui] local file not found: %s: %w", source, err)
		}
		filename := firstNonEmpty(resolved.Filename, filepath.Base(source))
		return preparedUpload{data, filename, firstNonEmpty(resolved.ContentType, mimeType(filename))}, nil
	case io.Reader:
		data, err := io.ReadAll(io.LimitReader(source, MaxUploadBytes+1))
		if err != nil {
			return preparedUpload{}, err
		}
		filename := firstNonEmpty(resolved.Filename, generatedFilename())
		return preparedUpload{data, filename, firstNonEmpty(resolved.ContentType, mimeType(filename))}, nil
	default:
		return preparedUpload{}, fmt.Errorf("[tuitui] unsupported upload source %T", source)
	}
}

func (u *uploader) prepareRemote(ctx context.Context, source string, options UploadOptions) (preparedUpload, error) {
	var response *http.Response
	var err error
	if u.config.fetchWithSSRF != nil {
		response, err = u.config.fetchWithSSRF(source)
	} else {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if requestErr != nil {
			return preparedUpload{}, requestErr
		}
		req.Close = true
		transport := &http.Transport{DisableKeepAlives: true, ForceAttemptHTTP2: false}
		defer transport.CloseIdleConnections()
		response, err = (&http.Client{Transport: transport, Timeout: u.config.httpTimeout}).Do(req)
	}
	if err != nil {
		return preparedUpload{}, fmt.Errorf("[tuitui] failed to download %s: %w", source, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return preparedUpload{}, fmt.Errorf("[tuitui] failed to download %s: %d", source, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxUploadBytes+1))
	if err != nil {
		return preparedUpload{}, err
	}
	filename := firstNonEmpty(options.Filename, remoteFilename(source, response.Header), "media")
	return preparedUpload{data, filename, firstNonEmpty(options.ContentType, response.Header.Get("Content-Type"), mimeType(filename))}, nil
}

func decodeDataURL(source string) ([]byte, string, error) {
	comma := strings.IndexByte(source, ',')
	if comma < 5 {
		return nil, "", fmt.Errorf("[tuitui] invalid data URL format")
	}
	metadata, payload := source[5:comma], source[comma+1:]
	parts := strings.Split(metadata, ";")
	contentType := firstNonEmpty(parts[0], "application/octet-stream")
	if len(parts) > 1 && parts[len(parts)-1] == "base64" {
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", fmt.Errorf("[tuitui] invalid base64 data URL: %w", err)
		}
		return data, contentType, nil
	}
	decoded, err := url.PathUnescape(payload)
	return []byte(decoded), contentType, err
}

var imageExtension = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|gif)$`)

func detectUploadMediaType(contentType, filename string) string {
	normalized := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if normalized != "" && normalized != "application/octet-stream" {
		switch normalized {
		case "image/jpg", "image/jpeg", "image/png", "image/gif":
			return "image"
		default:
			return "file"
		}
	}
	if imageExtension.MatchString(filename) {
		return "image"
	}
	return "file"
}

func mimeType(filename string) string {
	if value := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); value != "" {
		return value
	}
	return "application/octet-stream"
}

func remoteFilename(source string, headers http.Header) string {
	disposition := headers.Get("Content-Disposition")
	if _, params, err := mime.ParseMediaType(disposition); err == nil && params["filename"] != "" {
		return params["filename"]
	}
	parsed, err := url.Parse(source)
	if err == nil && filepath.Base(parsed.Path) != "." && filepath.Base(parsed.Path) != "/" {
		return filepath.Base(parsed.Path)
	}
	return "media"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func generatedFilename() string { return fmt.Sprintf("media_%d", time.Now().UnixMilli()) }

type FileAPI struct{ uploader *uploader }

func (f *FileAPI) Upload(ctx context.Context, source any, options *UploadOptions) (UploadResult, error) {
	return f.uploader.upload(ctx, source, options)
}
