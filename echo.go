package echohandles

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func Register(e *echo.Echo, tx *gorm.DB, tables []string) {
	e.GET("/table/:table", TableQueryHandle(tables, tx))
}

func RegisterWithGoup(g *echo.Group, tx *gorm.DB, tables []string) {
	g.GET("/table/:table", TableQueryHandle(tables, tx))
}
