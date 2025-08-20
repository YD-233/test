package oss

import (
	"BackendTemplate/pkg/encrypt"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Client struct {
	Cli             *oss.Client
	Bucket          *oss.Bucket
	Endpoint        string
	AccessKeyId     string
	AccessKeySecret string
	BucketName      string
}

var Service *Client

// var c *oss.Bucket

func InitClient(endPoint, accessKeyId, accessKeySecret, bucketName string) error {
	var ossClient *oss.Client
	var err error

	ossClient, err = oss.New(endPoint, accessKeyId, accessKeySecret)
	if err != nil {
		return err
	}

	var ossBucket *oss.Bucket
	ossBucket, err = ossClient.Bucket(bucketName)
	if err != nil {
		return err
	}

	Service = &Client{
		Cli:             ossClient,
		Bucket:          ossBucket,
		Endpoint:        endPoint,
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
		BucketName:      bucketName,
	}
	return nil
}
func List(c *Client) []oss.ObjectProperties {

	lsRes, err := c.Bucket.ListObjects(oss.MaxKeys(3), oss.Prefix(""))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	//fmt.Println(lsRes)
	return lsRes.Objects

}
func Send(c *Client, name string, content []byte) {
	encodeData, err := encrypt.EncodeBase64(content)

	// 1.通过字符串上传对象
	f := strings.NewReader(string(encodeData))
	// var err error
	err = c.Bucket.PutObject(name, f)
	if err != nil {
		log.Println("[-]", "上传失败")
		return
	}

}
func Get(c *Client, name string) []byte {
	//fmt.Println(name)
	body, err := c.Bucket.GetObject(name)
	if err != nil {
		return nil
	}
	// 数据读取完成后，获取的流必须关闭，否则会造成连接泄漏，导致请求无连接可用，程序无法正常工作。
	defer body.Close()
	// println(body)
	data, err := io.ReadAll(body)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	//fmt.Println(string(data))
	decodeData, err := encrypt.DecodeBase64(data)
	return decodeData
}

func Del(c *Client, name string) {
	err := c.Bucket.DeleteObject(name)
	if err != nil {
		panic(err)
	}

}
