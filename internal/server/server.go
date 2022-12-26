package server

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/services"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

type Server struct {
	service  services.ProxyService
	endpoint string
	port     int
}

func New(service services.ProxyService, endpoint string, port int) Server {
	return Server{
		service:  service,
		endpoint: endpoint,
		port:     port,
	}
}

func (s *Server) Run() {
	gin.ForceConsoleColor()
	r := gin.Default()
	r.GET("/:keyID", s.redirect)
	r.POST("/", s.createRedirect)
	dst := fmt.Sprintf(":%d", s.port)
	http.ListenAndServe(dst, r)
}

func (s *Server) createRedirect(ctx *gin.Context) {
	headerContentType := ctx.Request.Header.Get("Content-Type")
	ctx.Writer.Header().Set("content-type", "text/plain")
	var url = ""
	if headerContentType == "application/x-www-form-urlencoded" {
		url = ctx.PostForm("url")
	} else if headerContentType == "text/plain; charset=utf-8" {
		urlBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Invalid payload")
			fmt.Println("Invalid payload.")
		}
		url = strings.TrimSuffix(string(urlBytes), "\n")
	} else {
		ctx.String(http.StatusInternalServerError, "Invalid Content-Type header")
		fmt.Println("Invalid Content-Type header.")
		return
	}
	if url == "" {
		ctx.String(http.StatusBadRequest, "Invalid URL")
		fmt.Println("Invalid URL.")
		return
	}
	key := s.service.CreateRedirect(url)
	resultLink := fmt.Sprintf("%s/%s", s.endpoint, key)
	ctx.Status(http.StatusCreated)
	ctx.Writer.Write([]byte(resultLink))
}

func (s *Server) redirect(ctx *gin.Context) {
	key := ctx.Param("keyID")
	url, err := s.service.GetLinkByKeyID(key)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid key")
		fmt.Println("Invalid key", key)
		return
	}
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}
