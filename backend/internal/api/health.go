 package api
 
 import (
 	"net/http"
 
 	"github.com/gin-gonic/gin"
 
 	"github.com/chainwise/backend/internal/model"
 )
 
 func (h *Handler) Health(c *gin.Context) {
 	status := gin.H{
 		"service": "chainwise-backend",
 		"status":  "ok",
 	}
 
 	// 检查 Redis 连接
 	if h.cacheClient != nil {
 		if err := h.cacheClient.Ping(c.Request.Context()); err != nil {
 			status["redis"] = "disconnected"
 		} else {
 			status["redis"] = "connected"
 		}
 	}
 
 	c.JSON(http.StatusOK, model.Success(status))
 }
