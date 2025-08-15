package database

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"reflect"
	"xorm.io/xorm"
)

var Engine *xorm.Engine
var DatabaseName string = "./database.db"

type Users struct {
	Username string
	Password string
	//Email       string
	//Phone       string
	//Permissions int
}
type Clients struct {
	Uid        string
	FirstStart string
	ExternalIP string
	InternalIP string
	Username   string
	Computer   string
	Process    string
	Pid        string
	Address    string
	Arch       string
	Note       string
	Sleep      string
	Online     string
	Color      string
}
type Notes struct {
	Uid  string
	Note string
}
type Shell struct {
	Uid          string
	ShellContent string
}

type Downloads struct {
	Uid            string
	FileName       string
	FilePath       string
	FileSize       int
	DownloadedSize int
}
type Listener struct {
	Type           string
	ListenAddress  string
	ConnectAddress string
	Status         int
}
type WebDelivery struct {
	ListenerConfig string
	OS             string
	Arch           string
	ListeningPort  string
	Status         int
	ServerAddress  string
	FileName       string
}

func ConnectDateBase() {
	var err error
	Engine, err = xorm.NewEngine("sqlite3", DatabaseName)
	if err != nil {
		log.Fatalf("连接sqlite3数据库失败: %v", err)
	}
	err = Engine.Sync2(new(Users), new(Clients), new(Notes), new(Shell), new(Downloads), new(Listener), new(WebDelivery))
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	var user Users
	exists, err := Engine.Where("username = ?", "admin").Get(&user)
	if !exists {
		// 如果不存在 admin 用户，插入默认的 admin 用户
		defaultUser := &Users{
			Username: "admin",
			Password: "admin123",
			//Email:       "admin@example.com",
			//Phone:       "1234567890",
			//Permissions: 1,
		}

		err = InsertData(Engine, defaultUser)
		if err != nil {
			log.Fatalf("插入默认 admin 用户失败: %v", err)
		}
	}
}

// InsertData 函数用于插入任意表的数据
func InsertData(engine *xorm.Engine, table interface{}) error {
	// 使用反射获取表的信息
	valueOfTable := reflect.ValueOf(table)
	if valueOfTable.Kind() != reflect.Ptr || valueOfTable.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("table parameter must be a pointer to a struct")
	}

	// 同步数据库结构
	err := engine.Sync2(table)
	if err != nil {
		return fmt.Errorf("failed to sync database structure: %v", err)
	}

	// 插入数据
	_, err = engine.Insert(table)
	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	return nil
}
func ExecuteSQL(engine *xorm.Engine, sql string, args ...interface{}) error {
	_, err := engine.Exec(sql, args)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %v", err)
	}
	return nil
}
func QuerySQL(engine *xorm.Engine, sql string, args ...interface{}) ([]map[string]string, error) {
	results, err := engine.QueryString(sql, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query SQL: %v", err)
	}
	return results, nil
}
