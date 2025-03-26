package main

import (
	_ "embed"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"oss-upload-demo/oss/aliyun"
)

var aliyunOssConfig aliyun.OSSConfig

func init() {
	gin.SetMode(gin.DebugMode)
	// 加载当前.env文件
	_ = godotenv.Load(".env")

	aliyunOssConfig = aliyun.OSSConfig{
		Endpoint:        os.Getenv("ALIYUN_OSS_ENDPOINT"),
		AccessKeyID:     os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET"),
		BucketName:      os.Getenv("ALIYUN_OSS_BUCKET_NAME"),
	}
}

func main() {
	r := gin.Default()
	registerAliyunOssRoutes(r)
	r.Run(":8080")
}

//go:embed oss/aliyun/index.html
var aliyunOssHtml []byte

func registerAliyunOssRoutes(r *gin.Engine) {
	ossClient, err := aliyun.NewOSSClient(&aliyunOssConfig)
	if err != nil {
		panic(err)
	}

	group := r.Group("/aliyun")
	group.GET("/upload", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", aliyunOssHtml)
	})
	uploadHandler := aliyun.NewUploadHandler(ossClient)
	group.GET("/upload/init", uploadHandler.InitUpload)
	group.GET("/upload/part-url", uploadHandler.GenerateUploadPartURL)
	group.POST("/upload/complete", uploadHandler.CompleteUpload)
}
