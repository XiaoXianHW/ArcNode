package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/category"
	"github.com/XiaoXianHW/ArcNode/gateway/middleware"
	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

type Server struct {
	Store      *storage.Store
	Classifier *category.Classifier
	Token      string
	SegmentGap int64
	WebFS      fs.FS
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api/v1")
	api.Use(middleware.BearerAuth(s.Token))
	{
		api.POST("/init", s.handleInit)
		api.POST("/events", s.handleEvents)
		api.GET("/devices", s.handleListDevices)
		api.GET("/devices/:id", s.handleGetDevice)
		api.GET("/events", s.handleQueryEvents)
		api.GET("/segments", s.handleQuerySegments)
		api.GET("/stats/categories", s.handleCategoryStats)
		api.GET("/stats/apps", s.handleAppStats)
		api.GET("/stats/shortcuts", s.handleShortcutStats)
		api.GET("/stats/summary", s.handleSummary)
		api.GET("/categories", s.handleCategoryRules)
	}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	if s.WebFS != nil {
		s.mountWeb(r)
	}
	return r
}

func (s *Server) mountWeb(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if p == "" || isSPARoute(p) {
			serveFile(c, s.WebFS, "index.html", "text/html; charset=utf-8")
			return
		}
		f, err := s.WebFS.Open(p)
		if err != nil {
			serveFile(c, s.WebFS, "index.html", "text/html; charset=utf-8")
			return
		}
		_ = f.Close()
		http.FileServer(http.FS(s.WebFS)).ServeHTTP(c.Writer, c.Request)
	})
}

func isSPARoute(p string) bool {
	if strings.HasPrefix(p, "api/") || strings.HasPrefix(p, "assets/") {
		return false
	}
	return !strings.Contains(p, ".")
}

func serveFile(c *gin.Context, fsys fs.FS, name, contentType string) {
	f, err := fsys.Open(name)
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}
	defer f.Close()
	c.Header("Content-Type", contentType)
	if _, err := copyFile(c.Writer, f); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func copyFile(w http.ResponseWriter, f fs.File) (int64, error) {
	return copyReader(w, f)
}
