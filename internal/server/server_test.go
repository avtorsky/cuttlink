package server

import (
	"bytes"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/services"
	"github.com/avtorsky/cuttlink/internal/storage"
	"github.com/gin-gonic/gin"
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

func TestServer__createRedirect(t *testing.T) {
	localStorage := storage.New()
	localProxyService := services.New(localStorage)
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
			s := &Server{service: localProxyService}
			r := SetUpRouter()
			r.POST("/", s.createRedirect)
			data := url.Values{}
			data.Set(tt.key, tt.value)
			request := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(data.Encode()))
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

func TestServer__redirect(t *testing.T) {
	localStorage := storage.New()
	localProxyService := services.New(localStorage)
	dst := "https://explorer.avtorskydeployed.online/"
	testKey := localProxyService.CreateRedirect(dst)
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
			s := &Server{service: localProxyService}
			r := SetUpRouter()
			r.GET("/:keyID", s.redirect)
			request := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, res.StatusCode)
			}
			if tt.code == 307 {
				dst := res.Header.Get("location")
				if dst != tt.location {
					t.Errorf("Expected location %s, got %s", tt.location, dst)
				}
			}
			defer res.Body.Close()
		})
	}
}
