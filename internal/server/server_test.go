package server

import (
	"bytes"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/services"
	"github.com/avtorsky/cuttlink/internal/storage"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

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
			name:        "post_invalid_method_400",
			method:      http.MethodDelete,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
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
			data := url.Values{}
			data.Set(tt.key, tt.value)
			request := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(data.Encode()))
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.routeRedirect)
			h.ServeHTTP(w, request)
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
	testKey := localProxyService.CreateRedirect("https://explorer.avtorskydeployed.online/")
	tests := []struct {
		name   string
		method string
		code   int
		url    string
	}{
		{
			name:   "get_ok_301",
			method: http.MethodGet,
			code:   307,
			url:    fmt.Sprintf("/%s", testKey),
		},
		{
			name:   "get_invalid_key_400",
			method: http.MethodGet,
			code:   400,
			url:    "/ID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{service: localProxyService}
			request := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.routeRedirect)
			h.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, w.Code)
			}
			defer res.Body.Close()
		})
	}
}
