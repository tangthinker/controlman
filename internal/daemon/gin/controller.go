package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tangthinker/controlman/internal/daemon"
)

type Controller struct {
	daemon *daemon.Daemon
}

func NewController(daemon *daemon.Daemon) *Controller {
	return &Controller{daemon: daemon}
}

func (c *Controller) Command(ctx *gin.Context) {
	var cmd daemon.Command
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := c.daemon.HandleCommand(cmd)
	ctx.JSON(http.StatusOK, response)
}
