package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/storage"
	"github.com/avtorsky/cuttlink/internal/workers"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type PayloadJSON struct {
	URL string `json:"url" binding:"required"`
}

type ResponseJSON struct {
	Result string `json:"result"`
}

type URLPair struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

type URLPairRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url" binding:"required"`
}

type URLPairResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type Server struct {
	srv         *http.Server
	storage     storage.Storager
	serverHost  string
	serviceHost string
	removalCh   chan workers.RemovalTask
}

type ServerOption func(*Server) error

func WithServerHost(address string) ServerOption {
	return func(s *Server) error {
		s.serverHost = address
		return nil
	}
}

func WithServiceHost(bURL string) ServerOption {
	return func(s *Server) error {
		s.serviceHost = bURL
		return nil
	}
}

func New(storage storage.Storager, opts ...ServerOption) (Server, error) {
	const (
		defaultServerHost  = ":8080"
		defaultServiceHost = "http://localhost:8080"
	)

	ctx := context.Background()
	removalTasks := make(chan workers.RemovalTask, 10)
	removalWorker := workers.New(storage, removalTasks)
	go removalWorker.Run(ctx)

	s := Server{
		srv:         nil,
		storage:     storage,
		serverHost:  defaultServerHost,
		serviceHost: defaultServiceHost,
		removalCh:   removalTasks,
	}

	for _, opt := range opts {
		if err := opt(&s); err != nil {
			return Server{}, err
		}
	}

	gin.ForceConsoleColor()
	r := gin.New()
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		compressMiddleware(),
		decompressMiddleware(),
		cookieAuthentication(),
	)
	r.GET("/:id", s.redirect)
	r.POST("/", s.createShortURL)
	r.POST("/form-submit", s.createShortURLWebForm)
	r.POST("/api/shorten", s.createShortURLJSON)
	r.POST("/api/shorten/batch", s.createShortURLBatch)
	r.GET("/api/user/urls", s.getUserURLs)
	r.DELETE("/api/user/urls", s.deleteUserURLs)
	r.GET("/ping", s.pingDSN)

	srv := http.Server{
		Addr:    s.serverHost,
		Handler: r,
	}

	s.srv = &srv
	return s, nil
}

func (s *Server) ListenAndServe() {
	s.srv.ListenAndServe()
}

func (s *Server) createShortURL(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}

	var baseURL string
	switch headerContentType {
	case "application/x-gzip", "text/plain; charset=utf-8":
		dataBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
			return
		}
		baseURL = strings.TrimSpace(string(dataBytes))
	default:
		ctx.String(http.StatusInternalServerError, "Invalid Content-Type header")
		return
	}

	u, _ := url.Parse(baseURL)
	if u.Scheme == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL scheme")
		return
	} else if u.Host == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL host")
		return
	}

	key, err := s.storage.SetURL(ctx.Request.Context(), baseURL, sessionID)
	if err != nil {
		ctx.Writer.Header().Set("Content-Type", "text/plain")
		var dbError *storage.DuplicateURLError
		if errors.As(err, &dbError) {
			shortURL := fmt.Sprintf("%s/%s", s.serviceHost, dbError.Key)
			ctx.String(http.StatusConflict, shortURL)
			return
		}
		ctx.String(http.StatusInternalServerError, "Internal server I/O error")
		return
	}

	shortURL := fmt.Sprintf("%s/%s", s.serviceHost, key)
	ctx.Writer.Header().Set("Content-Type", "text/plain")
	ctx.String(http.StatusCreated, shortURL)
}

func (s *Server) createShortURLWebForm(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}

	var baseURL string
	switch headerContentType {
	case "application/x-gzip":
		dataBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
			return
		}
		baseURL = strings.TrimSpace(string(dataBytes))
	case "application/x-www-form-urlencoded":
		baseURL = ctx.PostForm("url")
	default:
		ctx.String(http.StatusInternalServerError, "Invalid Content-Type header")
		return
	}

	if baseURL == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL")
		return
	}
	u, _ := url.Parse(baseURL)
	if u.Scheme == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL scheme")
		return
	} else if u.Host == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL host")
		return
	}

	key, err := s.storage.SetURL(ctx.Request.Context(), baseURL, sessionID)
	if err != nil {
		ctx.Writer.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		var dbError *storage.DuplicateURLError
		if errors.As(err, &dbError) {
			shortURL := fmt.Sprintf("%s/%s", s.serviceHost, dbError.Key)
			ctx.String(http.StatusConflict, shortURL)
			return
		}
		ctx.String(http.StatusInternalServerError, "Internal server I/O error")
		return
	}

	shortURL := fmt.Sprintf("%s/%s", s.serviceHost, key)
	ctx.Writer.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.String(http.StatusCreated, shortURL)
}

func (s *Server) createShortURLJSON(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}

	var payload PayloadJSON
	if headerContentType != "application/json" {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Invalid Content-Type header",
		})
		return
	}
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid URL",
		})
		return
	}
	if _, err := url.ParseRequestURI(payload.URL); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid URL scheme",
		})
		return
	}
	u, _ := url.Parse(payload.URL)
	if u.Host == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid URL host",
		})
		return
	}

	key, err := s.storage.SetURL(ctx.Request.Context(), payload.URL, sessionID)
	if err != nil {
		ctx.Writer.Header().Set("Content-Type", "application/json")
		var dbError *storage.DuplicateURLError
		if errors.As(err, &dbError) {
			shortURL := ResponseJSON{
				Result: fmt.Sprintf("%s/%s", s.serviceHost, dbError.Key),
			}
			ctx.JSON(http.StatusConflict, shortURL)
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server I/O error",
		})
		return
	}

	shortURL := ResponseJSON{
		Result: fmt.Sprintf("%s/%s", s.serviceHost, key),
	}
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.JSON(http.StatusCreated, shortURL)

	result, err := json.Marshal(shortURL)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(result))
}

func (s *Server) createShortURLBatch(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}
	if headerContentType != "application/json" {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Invalid Content-Type header",
		})
		return
	}

	request := make([]URLPairRequest, 0)
	if err := ctx.BindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid payload",
		})
		return
	}
	size := len(request)
	urlBatch := make([]string, size)
	for i := range request {
		if _, err := url.ParseRequestURI(request[i].OriginalURL); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid URL scheme",
			})
			return
		}
		u, _ := url.Parse(request[i].OriginalURL)
		if u.Host == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid URL host",
			})
			return
		}
		urlBatch[i] = request[i].OriginalURL
	}

	urlBatch, err = s.storage.SetBatchURL(ctx.Request.Context(), urlBatch, sessionID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server I/O error",
		})
		return
	}

	response := make([]URLPairResponse, size)
	for i := range request {
		response[i] = URLPairResponse{
			CorrelationID: request[i].CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.serviceHost, urlBatch[i]),
		}
	}
	ctx.Writer.Header().Set("Content-Type", "application/json")
	if len(response) == 0 {
		ctx.JSON(http.StatusNoContent, response)
	}
	ctx.JSON(http.StatusCreated, response)
}

func (s *Server) redirect(ctx *gin.Context) {
	key := ctx.Param("id")
	baseURL, err := s.storage.GetURL(ctx.Request.Context(), key)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid key")
		return
	}

	if baseURL.IsDeleted {
		ctx.AbortWithStatus(http.StatusGone)
		return
	}
	ctx.Redirect(http.StatusTemporaryRedirect, baseURL.Value)
}

func (s *Server) getUserURLs(ctx *gin.Context) {
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}

	urlMap, err := s.storage.GetUserURLs(ctx.Request.Context(), sessionID)
	result := make([]URLPair, len(urlMap))
	if len(result) < 1 || err != nil {
		ctx.JSON(http.StatusNoContent, result)
		return
	}
	item := 0
	for key, url := range urlMap {
		result[item] = URLPair{
			OriginalURL: url,
			ShortURL:    fmt.Sprintf("%s/%s", s.serviceHost, key),
		}
		item++
	}
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.JSON(http.StatusOK, result)
}

func (s *Server) deleteUserURLs(ctx *gin.Context) {
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}

	var keys []string
	if err := json.NewDecoder(ctx.Request.Body).Decode(&keys); err != nil {
		ctx.String(http.StatusBadRequest, "URL keys parse error")
		return
	}
	s.removalCh <- workers.RemovalTask{
		Keys: keys,
		UUID: sessionID,
	}
	ctx.Status(http.StatusAccepted)
}

func (s *Server) pingDSN(ctx *gin.Context) {
	ctx.Writer.Header().Set("Content-Type", "text/plain")
	err := s.storage.Ping(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "DSN out of service timeout")
		return
	}
	ctx.String(http.StatusOK, "OK")
}
