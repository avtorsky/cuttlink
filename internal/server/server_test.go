package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/storage"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type TestServer struct {
	*httptest.Server
	storage  *storage.StorageDB
	kvstore  *storage.FileStorage
	filename string
}

func NewTestServer(t *testing.T) TestServer {
	file, err := os.CreateTemp("", "cuttlink-test-*.txt")
	assert.Nil(t, err)
	tfs, _ := storage.NewFileStorage(file.Name())
	ls, _ := storage.NewKV(tfs)
	s, err := New(ls)
	assert.Nil(t, err)
	gin.ForceConsoleColor()
	r := gin.New()
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		compressMiddleware(),
		decompressMiddleware(),
		cookieAuthentication(),
	)
	r.GET("/:keyID", s.redirect)
	r.POST("/", s.createShortURL)
	r.POST("/form-submit", s.createShortURLWebForm)
	r.POST("/api/shorten", s.createShortURLJSON)
	r.GET("/api/user/urls", s.getUserURLs)
	ts := httptest.NewServer(r)
	srv := TestServer{
		Server:   ts,
		storage:  ls,
		kvstore:  tfs,
		filename: file.Name(),
	}
	return srv
}

func (s *TestServer) Close() {
	s.Server.Close()
	s.kvstore.CloseFS()
	os.Remove(s.filename)
}

func TestServer__createShortURLWebForm(t *testing.T) {
	t.SkipNow()
	ts := NewTestServer(t)
	defer ts.Close()
	client := http.Client{}
	rURL := fmt.Sprintf("%s/form-submit", ts.URL)
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
			value:       "https://yatube.avtorskydeployed.online/",
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
			value:       "yatube.avtorskydeployed.online",
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
			value:       "https://yatube.avtorskydeployed.online/",
		},
		{
			name:        "post_invalid_content_type_500",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        500,
			key:         "url",
			value:       "https://yatube.avtorskydeployed.online/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := url.Values{}
			data.Set(tt.key, tt.value)
			req, err := http.NewRequest(tt.method, rURL, bytes.NewBufferString(data.Encode()))
			assert.Nil(t, err)
			req.Header.Set("Content-Type", tt.contentType)
			res, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, res.StatusCode, "http status codes should be equal")
			defer res.Body.Close()
		})
	}
}

func TestServer__createShortURLJSON(t *testing.T) {
	t.SkipNow()
	ts := NewTestServer(t)
	defer ts.Close()
	client := http.Client{}
	rURL := fmt.Sprintf("%s/api/shorten", ts.URL)

	type request struct {
		URL string `json:"url" binding:"required"`
	}

	type response struct {
		Result string `json:"result"`
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		code        int
		data        request
		result      response
	}{
		{
			name:        "post_ok_201",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        201,
			data:        request{URL: "https://yatube.avtorskydeployed.online/"},
			result:      response{Result: "http://localhost:8080/2"},
		},
		{
			name:        "post_empty_url_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        request{URL: ""},
			result:      response{Result: ""},
		},
		{
			name:        "post_url_without_scheme_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        request{URL: "yatube.avtorskydeployed.online"},
			result:      response{Result: ""},
		},
		{
			name:        "post_url_without_host_400",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        400,
			data:        request{URL: "https://"},
			result:      response{Result: ""},
		},
		{
			name:        "post_invalid_method_404",
			method:      http.MethodDelete,
			contentType: "application/json",
			code:        404,
			data:        request{URL: "https://yatube.avtorskydeployed.online/"},
			result:      response{Result: ""},
		},
		{
			name:        "post_invalid_content_type_500",
			method:      http.MethodPost,
			contentType: "application/xml",
			code:        500,
			data:        request{URL: "https://yatube.avtorskydeployed.online/"},
			result:      response{Result: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.data)
			assert.Nil(t, err)
			req, _ := http.NewRequest(tt.method, rURL, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", tt.contentType)
			res, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, res.StatusCode, "http status codes should be equal")
			defer res.Body.Close()

			if tt.code == http.StatusCreated {
				dataBytes, err := io.ReadAll(res.Body)
				assert.Nil(t, err)
				body := response{}
				assert.Nil(t, json.Unmarshal(dataBytes, &body))
				assert.Equal(t, tt.result, body, "response body should be equal")
			}
		})
	}
}

func TestServer__redirect(t *testing.T) {
	t.SkipNow()
	ts := NewTestServer(t)
	defer ts.Close()
	baseURL := "https://yatube.avtorskydeployed.online"
	key, err := ts.storage.Insert(baseURL, "6a15c16b-b941-48b3-be78-8e539838d612")
	assert.Nil(t, err)
	client := http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	tests := []struct {
		name     string
		method   string
		code     int
		shortURL string
		location string
	}{
		{
			name:     "get_ok_307",
			method:   http.MethodGet,
			code:     307,
			shortURL: fmt.Sprintf("/%s", key),
			location: baseURL,
		},
		{
			name:     "get_invalid_key_400",
			method:   http.MethodGet,
			code:     400,
			shortURL: "/ID",
			location: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", ts.URL, tt.shortURL)
			res, err := client.Get(url)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, res.StatusCode, "http status codes should be equal")
			defer res.Body.Close()

			if tt.code == http.StatusTemporaryRedirect {
				loc := res.Header.Get("location")
				assert.Equal(t, tt.location, loc, "http status codes should be equal")
			}
		})
	}
}

func TestServer__getUserURLs(t *testing.T) {
	t.SkipNow()
	ts := NewTestServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar: jar,
	}
	assert := assert.New(t)
	contentType := "application/json"

	type row struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}

	type request struct {
		URL string `json:"url" binding:"required"`
	}

	type response struct {
		Result string `json:"result"`
	}

	tests := []row{
		{
			ShortURL:    "http://localhost:8080/2",
			OriginalURL: "https://yatube.avtorskydeployed.online/",
		},
		{
			ShortURL:    "http://localhost:8080/3",
			OriginalURL: "https://explorer.avtorskydeployed.online/",
		},
	}

	aURL := fmt.Sprintf("%s/api/user/urls", ts.URL)
	res, err := client.Get(aURL)
	assert.Equal(http.StatusNoContent, res.StatusCode, "invalid http status code")
	assert.Nil(err)
	aBytes, err := io.ReadAll(res.Body)
	assert.Nil(err)
	body := make([]row, 0)
	json.Unmarshal(aBytes, &body)
	assert.Equal(make([]row, 0), body, "response body should be empty")
	defer res.Body.Close()

	for item := range tests {
		bURL := fmt.Sprintf("%s/api/shorten", ts.URL)
		data := request{URL: tests[item].OriginalURL}
		bBytes, err := json.Marshal(data)
		assert.Nil(err)
		res, err := client.Post(bURL, contentType, bytes.NewBuffer(bBytes))
		assert.Nil(err)
		assert.Equal(http.StatusCreated, res.StatusCode, "http status codes should be equal")
		bodyBytes, err := io.ReadAll(res.Body)
		assert.Nil(err)
		var body response
		err = json.Unmarshal(bodyBytes, &body)
		assert.Nil(err)
		tests[item].ShortURL = body.Result
		defer res.Body.Close()
	}

	res, err = client.Get(aURL)
	assert.Equal(http.StatusNoContent, res.StatusCode, "invalid http status code")
	assert.Nil(err)
	defer res.Body.Close()
}
