package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/XiaoXianHW/ArcNode/gateway/storage"
)

func (s *Server) handleListCustomKeywords(c *gin.Context) {
	kws, err := s.Store.ListCustomKeywords()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if kws == nil {
		kws = []storage.CustomKeyword{}
	}
	c.JSON(http.StatusOK, gin.H{"keywords": kws})
}

func (s *Server) handleAddCustomKeyword(c *gin.Context) {
	var body struct {
		Category string `json:"category"`
		Keyword  string `json:"keyword"`
		Scope    string `json:"scope"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Category == "" || body.Keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category and keyword required"})
		return
	}
	kw, err := s.Store.AddCustomKeyword(body.Category, body.Keyword, body.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.reloadCustomKeywords()
	c.JSON(http.StatusOK, kw)
}

func (s *Server) handleDeleteCustomKeyword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.Store.DeleteCustomKeyword(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.reloadCustomKeywords()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) reloadCustomKeywords() {
	m, err := s.Store.CustomKeywordMap()
	if err != nil {
		return
	}
	s.Classifier.SetCustom(m)
}
