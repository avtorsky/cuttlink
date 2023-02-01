package server

import (
	"bytes"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/storage"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func SetUpRouter() *gin.Engine {
	gin.ForceConsoleColor()
	router := gin.Default()
	return router
}

func TestServer__createShortURLWebForm(t *testing.T) {
	localStorage := storage.New()
	tests := []struct {
		name        string
		method      string
		contentType string
		code        int
		key         string
		value       string
	}{
		{
			name:        "post_ok_201",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        201,
			key:         "url",
			value:       "https://explorer.avtorskydeployed.online/",
		},
		{
			name:        "post_empty_url_400",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			key:         "url",
			value:       "",
		},
		{
			name:        "post_url_without_scheme_400",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			key:         "url",
			value:       "explorer.avtorskydeployed.online",
		},
		{
			name:        "post_url_without_host_400",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			key:         "url",
			value:       "https://",
		},
		{
			name:        "post_invalid_method_404",
			method:      http.MethodDelete,
			contentType: "application/x-www-form-urlencoded",
			code:        404,
			key:         "url",
			value:       "https://explorer.avtorskydeployed.online/",
		},
		{
			name:        "post_invalid_content_type_500",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        500,
			key:         "url",
			value:       "https://explorer.avtorskydeployed.online/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{storage: localStorage}
			r := SetUpRouter()
			r.POST("/form-submit", s.createShortURLWebForm)
			data := url.Values{}
			data.Set(tt.key, tt.value)
			request := httptest.NewRequest(tt.method, "/form-submit", bytes.NewBufferString(data.Encode()))
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, w.Code)
			}
			defer res.Body.Close()
		})
	}
}

func TestServer__createShortURLJSON(t *testing.T) {
	localStorage := storage.New()
	tests := []struct {
		name        string
		method      string
		contentType string
		code        int
		data        string
		result      string
	}{
		{
			name:        "post_ok_201",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        201,
			data:        "{\"url\": \"https://explorer.avtorskydeployed.online/\"}",
			result:      "{\"result\":\"http://localhost:8080/2\"}",
		},
		{
			name:        "post_empty_url_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        "{\"url\": \"\"}",
			result:      "",
		},
		{
			name:        "post_url_without_scheme_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        "{\"url\": \"explorer.avtorskydeployed.online/\"}",
			result:      "",
		},
		{
			name:        "post_url_without_host_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        "{\"url\": \"https://\"}",
			result:      "",
		},
		{
			name:        "post_invalid_method_404",
			method:      http.MethodDelete,
			contentType: "application/json",
			code:        404,
			data:        "{\"url\": \"https://explorer.avtorskydeployed.online/\"}",
			result:      "",
		},
		{
			name:        "post_invalid_content_type_500",
			method:      http.MethodPost,
			contentType: "application/xml",
			code:        500,
			data:        "{\"url\": \"https://explorer.avtorskydeployed.online/\"}",
			result:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				storage: localStorage,
				endpoint: "http://localhost:8080",
				port: 8080,
			}
			r := SetUpRouter()
			r.POST("/api/shorten", s.createShortURLJSON)
			request := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.data))
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, w.Code)
			}
			if tt.code == http.StatusCreated {
				responseBytes, _ := io.ReadAll(res.Body)
				response := string(responseBytes)
				if response != tt.result {
					t.Errorf("Expected result %s, got %s", tt.result, response)
				}
			}
			defer res.Body.Close()
		})
	}
}

func TestServer__redirect(t *testing.T) {
	localStorage := storage.New()
	dst := "https://explorer.avtorskydeployed.online/"
	testKey := localStorage.Insert(dst)
	tests := []struct {
		name     string
		method   string
		code     int
		url      string
		location string
	}{
		{
			name:     "get_ok_307",
			method:   http.MethodGet,
			code:     307,
			url:      fmt.Sprintf("/%s", testKey),
			location: dst,
		},
		{
			name:     "get_invalid_key_400",
			method:   http.MethodGet,
			code:     400,
			url:      "/ID",
			location: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{storage: localStorage}
			r := SetUpRouter()
			r.GET("/:keyID", s.redirect)
			request := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, res.StatusCode)
			}
			if tt.code == http.StatusTemporaryRedirect {
				dst := res.Header.Get("location")
				if dst != tt.location {
					t.Errorf("Expected location %s, got %s", tt.location, dst)
				}
			}
			defer res.Body.Close()
		})
	}
}
