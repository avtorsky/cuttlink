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
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(compressMiddleware())
	r.Use(decompressMiddleware())
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
	case "application/x-gzip":
		dataBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
		}
		baseURL = strings.TrimSpace(string(dataBytes))
	case "text/plain; charset=utf-8":
		dataBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
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
	key := s.storage.Insert(baseURL)
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
	key := s.storage.Insert(baseURL)
	shortURL := fmt.Sprintf("%s/%s", s.serviceHost, key)
	ctx.Writer.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.String(http.StatusCreated, shortURL)
}

func (s *Server) createShortURLJSON(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	var payload PayloadJSON
	switch headerContentType {
	case "application/json":
		err := ctx.BindJSON(&payload)
		if err != nil || payload.URL == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid URL",
			})
			return
		}
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Invalid Content-Type header",
		})
		return
	}
	u, _ := url.Parse(payload.URL)
	if u.Scheme == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid URL scheme",
		})
		return
	} else if u.Host == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid URL host",
		})
		return
	}
	key := s.storage.Insert(payload.URL)
	shortURL := ResponseJSON{
		Result: fmt.Sprintf("%s/%s", s.serviceHost, key),
	}
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.JSON(http.StatusCreated, shortURL)
	result, _ := json.Marshal(shortURL)
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
