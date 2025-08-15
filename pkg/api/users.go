package api

import (
	"BackendTemplate/pkg/database"
	"crypto/rand"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

var JwtKey []byte

func init() {
	var err error
	JwtKey, err = generateSecureKey(32)
	if err != nil {
		fmt.Println(err)
	}
}

// Claims 结构体定义 JWT 的负载
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 生成 JWT
func generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}
func generateSecureKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// 验证 JWT
func validateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// 登录处理函数
func LoginHandler(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// 假设用户名和密码验证成功
	var users database.Users
	if database.Engine.Where("username = ?", loginData.Username).Get(&users); users.Password == loginData.Password {
		token, err := generateJWT(loginData.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
			"token":       token,
			"permissions": 1, // 示例：1表示管理员权限
			"refresh":     "mock-refresh-token",
			"username":    loginData.Username,
		}})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

// JWT 验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization2")[len("Bearer "):]
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			c.Abort()
			return
		}
		claims, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Next()
	}
}

// 注销处理函数
func LogoutHandler(c *gin.Context) {
	// 这里可以处理注销逻辑，比如删除 refresh token
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "Logged out successfully"})
}

// 修改密码处理函数
func ChangePasswordHandler(c *gin.Context) {
	var passwordData struct {
		OldPassword string `form:"old_password"`
		NewPassword string `form:"new_password"`
	}
	if err := c.ShouldBind(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	}

	// 处理密码修改逻辑
	if passwordData.OldPassword != passwordData.NewPassword {
		username := c.MustGet("username").(string)
		var users database.Users
		if database.Engine.Where("username = ?", username).Get(&users); users.Password == passwordData.OldPassword {
			users.Password = passwordData.NewPassword
			database.Engine.Where("username = ?", username).Update(&users)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "Password changed successfully"})
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "Password changed failed"})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "Password changed failed"})
	}

}

//
//// 获取用户信息处理函数
//func GetUserInfoHandler(c *gin.Context) {
//	// 获取用户名从上下文中
//	username := c.MustGet("username").(string)
//
//	var user database.Users
//	_, err := database.Engine.Where("username = ?", username).Get(&user)
//	if err != nil {
//		return
//	}
//	userInfo := gin.H{
//		"username":    username,
//		"permissions": user.Permissions, // 示例：1表示管理员
//		"phone":       user.Phone,
//	}
//	c.JSON(http.StatusOK, gin.H{"code": 200, "data": userInfo})
//}
//
//// User 返回给客户端的结构
//type User struct {
//	ID          string `json:"id"` // 使用 username 作为 ID
//	Username    string `json:"username"`
//	Permissions int    `json:"permissions"`
//	Phone       string `json:"phone"`
//}
//
//func GetUserListHandler(c *gin.Context) {
//	var query struct {
//		Page     int    `form:"page"`
//		PageSize int    `form:"page_size"`
//		Search   string `form:"search"`
//	}
//	if err := c.ShouldBindQuery(&query); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
//		return
//	}
//
//	// 默认分页值
//	if query.Page == 0 {
//		query.Page = 1
//	}
//	if query.PageSize == 0 {
//		query.PageSize = 10
//	}
//
//	// 构建查询条件
//	session := database.Engine.NewSession()
//	defer session.Close()
//
//	if query.Search != "" {
//		// 模糊查询 username
//		session = session.Where("username LIKE ? COLLATE NOCASE", "%"+query.Search+"%")
//	}
//
//	// 获取总记录数
//	total, err := session.Count(new(database.Users))
//	if err != nil {
//		log.Fatalf("获取总记录数失败: %v", err)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
//		return
//	}
//	if query.Search != "" {
//		// 模糊查询 username
//		session = session.Where("username LIKE ? COLLATE NOCASE", "%"+query.Search+"%")
//	}
//	// 分页查询
//	users := []database.Users{}
//	err = session.Limit(query.PageSize, (query.Page-1)*query.PageSize).Find(&users)
//	if err != nil {
//		log.Fatalf("获取用户列表失败: %v", err)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
//		return
//	}
//	var UserList []User
//	for id, user := range users {
//		var user1 User
//		user1.ID = strconv.Itoa(id)
//		user1.Username = user.Username
//		user1.Permissions = user.Permissions
//		user1.Phone = user.Phone
//		UserList = append(UserList, user1)
//	}
//
//	// 返回用户列表和总数
//	c.JSON(http.StatusOK, gin.H{
//		"code": 200,
//		"data": gin.H{
//			"list":  UserList,
//			"total": total,
//		},
//	})
//}
//
//// 创建用户处理函数
//func CreateUserHandler(c *gin.Context) {
//	var userData struct {
//		Username      string `json:"username"`
//		Password      string `json:"password"`
//		PasswordAgain string `json:"password_again"`
//		Phone         string `json:"phone"`
//		Email         string `json:"email"`
//		Permissions   string `json:"permissions"`
//	}
//	if err := c.ShouldBindJSON(&userData); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
//		return
//	}
//	if userData.Password == userData.PasswordAgain {
//		var user database.Users
//		user.Username = userData.Username
//		user.Password = userData.Password
//		user.Permissions, _ = strconv.Atoi(userData.Permissions)
//		user.Phone = userData.Phone
//		user.Email = userData.Email
//
//		exists, _ := database.Engine.Where("username = ?", userData.Username).Exist(new(database.Users))
//		if !exists {
//			database.Engine.Insert(&user)
//			// 创建用户逻辑
//			c.JSON(http.StatusOK, gin.H{"code": 200, "data": "User created successfully"})
//		}
//	}
//
//}
//
//// 删除用户处理函数
//func DeleteUserHandler(c *gin.Context) {
//	var deleteData struct {
//		Username string `json:"username"`
//	}
//	if err := c.ShouldBindJSON(&deleteData); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
//		return
//	}
//	database.Engine.Where("username = ?", deleteData.Username).Delete(new(database.Users))
//	// 删除用户逻辑
//	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "User deleted successfully"})
//
//}
