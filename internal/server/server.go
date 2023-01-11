package server

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/storage"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Server struct {
	storage  *storage.StorageDB
	endpoint string
	port     int
}

func New(storage *storage.StorageDB, endpoint string, port int) Server {
	return Server{
		storage:  storage,
		endpoint: endpoint,
		port:     port,
	}
}

func (s *Server) Run() {
	gin.ForceConsoleColor()
	r := gin.Default()
	r.GET("/:keyID", s.redirect)
	r.POST("/", s.createShortURL)
	r.POST("/form-submit", s.createShortURLWebForm)
	dst := fmt.Sprintf(":%d", s.port)
	http.ListenAndServe(dst, r)
}

func (s *Server) createShortURL(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	ctx.Writer.Header().Set("content-type", "text/plain")
	var baseURL string
	if headerContentType == "text/plain; charset=utf-8" {
		urlBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
			fmt.Println("Invalid payload.")
		}
		baseURL = strings.TrimSpace(string(urlBytes))
	} else {
		ctx.String(http.StatusInternalServerError, "Invalid Content-Type header")
		fmt.Println("Invalid Content-Type header.")
		return
	}
	u, _ := url.Parse(baseURL)
	if u.Scheme == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL scheme")
		fmt.Println("Invalid URL scheme.")
		return
	} else if u.Host == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL host")
		fmt.Println("Invalid URL host.")
		return
	}
	key := s.storage.Insert(baseURL)
	shortURL := fmt.Sprintf("%s/%s", s.endpoint, key)
	ctx.String(http.StatusCreated, shortURL)
}

func (s *Server) createShortURLWebForm(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	ctx.Writer.Header().Set("content-type", "application/x-www-form-urlencoded")
	var baseURL string
	if headerContentType == "application/x-www-form-urlencoded" {
		baseURL = ctx.PostForm("url")
	} else {
		ctx.String(http.StatusInternalServerError, "Invalid Content-Type header")
		fmt.Println("Invalid Content-Type header.")
		return
	}
	if baseURL == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL")
		fmt.Println("Invalid URL.")
		return
	}
	u, _ := url.Parse(baseURL)
	if u.Scheme == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL scheme")
		fmt.Println("Invalid URL scheme.")
		return
	} else if u.Host == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL host")
		fmt.Println("Invalid URL host.")
		return
	}
	key := s.storage.Insert(baseURL)
	shortURL := fmt.Sprintf("%s/%s", s.endpoint, key)
	ctx.String(http.StatusCreated, shortURL)
}

func (s *Server) redirect(ctx *gin.Context) {
	key := ctx.Param("keyID")
	baseURL, err := s.storage.Get(key)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid key")
		fmt.Println("Invalid key", key)
		return
	}
	ctx.Redirect(http.StatusTemporaryRedirect, baseURL)
}
