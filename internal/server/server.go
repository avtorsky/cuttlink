package server

import (
	"encoding/json"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/storage"
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

type respPair struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

type Server struct {
	storage     *storage.StorageDB
	serverHost  string
	serviceHost string
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

func New(storage *storage.StorageDB, opts ...ServerOption) (Server, error) {
	const (
		defaultserverHost  = ":8080"
		defaultserviceHost = "http://localhost:8080"
	)
	srv := Server{
		storage:     storage,
		serverHost:  defaultserverHost,
		serviceHost: defaultserviceHost,
	}
	for _, opt := range opts {
		if err := opt(&srv); err != nil {
			return Server{}, err
		}
	}
	return srv, nil
}

func (s *Server) Run() {
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
	http.ListenAndServe(s.serverHost, r)
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
	key, err := s.storage.Insert(baseURL, sessionID)
	if err != nil {
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
	key, err := s.storage.Insert(baseURL, sessionID)
	if err != nil {
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
	key, err := s.storage.Insert(payload.URL, sessionID)
	if err != nil {
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

func (s *Server) redirect(ctx *gin.Context) {
	key := ctx.Param("keyID")
	baseURL, err := s.storage.Get(key)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid key")
		return
	}
	ctx.Redirect(http.StatusTemporaryRedirect, baseURL)
}

func (s *Server) getUserURLs(ctx *gin.Context) {
	sessionID, err := getUUID(ctx)
	if err != nil {
		return
	}
	urlMap, err := s.storage.GetUserURLs(sessionID)
	result := make([]respPair, len(urlMap))
	if len(result) < 1 || err != nil {
		ctx.JSON(http.StatusNoContent, result)
		return
	}
	item := 0
	for key, url := range urlMap {
		result[item] = respPair{
			OriginalURL: url,
			ShortURL:    fmt.Sprintf("%s/%s", s.serviceHost, key),
		}
		item++
	}
	fmt.Println(result)
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.JSON(http.StatusOK, result)
}
