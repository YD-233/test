package main

import (
	"BackendTemplate/pkg/config"
	"BackendTemplate/pkg/database"
	"BackendTemplate/pkg/routes"
	"BackendTemplate/pkg/utils"
	"embed"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var embedFS embed.FS

func main() {
	utils.InitFunction()
	gin.SetMode(gin.ReleaseMode)

	database.ConnectDateBase()
	defer database.Engine.Close()

	database.Engine.Update(&database.Listener{Status: 2})
	database.Engine.Update(&database.WebDelivery{Status: 2})

	r := gin.New()

	// 设置路由
	routes.SetupRoutes(r, embedFS)

	fmt.Println("Listening on port ", config.WebPort)
	r.Run("0.0.0.0:" + strconv.Itoa(config.WebPort)) // 启动服务
}
