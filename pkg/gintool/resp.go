package gintool

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Error(cxt *gin.Context, err error) {
	cxt.JSON(http.StatusOK, gin.H{
		"suc": false,
		"msg": err.Error(),
	})
}
func SucMsg(cxt *gin.Context, msg string) {
	cxt.JSON(http.StatusOK, gin.H{
		"suc": true,
		"msg": msg,
	})
}

func SucData(cxt *gin.Context, data interface{}) {
	cxt.JSON(http.StatusOK, gin.H{
		"suc":  true,
		"data": data,
	})
}

func Suc(cxt *gin.Context) {
	cxt.JSON(http.StatusOK, gin.H{
		"suc": true,
	})
}
