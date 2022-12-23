package server

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/services"
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
	http.HandleFunc("/", s.routeRedirect)
	addr := fmt.Sprintf(":%d", s.port)
	http.ListenAndServe(addr, nil)
}

func (s *Server) createRedirect(rw http.ResponseWriter, r *http.Request) {
	headerContentType := r.Header.Get("Content-Type")
	rw.Header().Set("content-type", "text/plain")
	var url = ""
	if headerContentType == "application/x-www-form-urlencoded" {
		r.ParseForm()
		url = r.FormValue("url")
	} else if headerContentType == "text/plain; charset=utf-8" {
		urlBytes, err := io.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Println("Invalid payload.")
		}
		url = strings.TrimSuffix(string(urlBytes), "\n")
	} else {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Println("Invalid Content-Type header.")
		return
	}
	if url == "" {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Println("Invalid URL.")
		return
	}
	key := s.service.CreateRedirect(url)
	resultLink := fmt.Sprintf("%s/%s", s.endpoint, key)
	rw.WriteHeader(http.StatusCreated)
	rw.Write([]byte(resultLink))
}

func (s *Server) redirect(rw http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")
	url, err := s.service.GetLinkByKeyID(key)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Println("Invalid key", key)
		return
	}
	http.Redirect(rw, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) routeRedirect(rw http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.redirect(rw, r)
	} else if r.Method == http.MethodPost {
		s.createRedirect(rw, r)
	} else {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Println("Invalid HTTP request method.")
	}
}
