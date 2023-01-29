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

type Server struct {
	storage     *storage.StorageDB
	serverHost  string
	serviceHost string
}

func New(storage *storage.StorageDB, serverHost string, serviceHost string) Server {
	return Server{
		storage:     storage,
		serverHost:  serverHost,
		serviceHost: serviceHost,
	}
}

func (s *Server) Run() {
	gin.ForceConsoleColor()
	r := gin.New()
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		compressMiddleware(),
		decompressMiddleware(),
	)
	r.GET("/:keyID", s.redirect)
	r.POST("/", s.createShortURL)
	r.POST("/form-submit", s.createShortURLWebForm)
	r.POST("/api/shorten", s.createShortURLJSON)
	http.ListenAndServe(s.serverHost, r)
}

func (s *Server) createShortURL(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
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
	key, err := s.storage.Insert(baseURL)
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
	key, err := s.storage.Insert(baseURL)
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
	key, err := s.storage.Insert(payload.URL)
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
