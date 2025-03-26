package aliyun

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	ossClient *OSSClient
}

func NewUploadHandler(ossClient *OSSClient) *UploadHandler {
	return &UploadHandler{ossClient: ossClient}
}

// InitUpload 初始化分片上传
// @Summary 初始化分片上传
// @Description 初始化分片上传，获取UploadID
// @Tags 上传
// @Accept json
// @Produce json
// @Param filename query string true "文件名"
// @Success 200 {object} oss.InitiateMultipartUploadResult
// @Failure 500 {object} map[string]string
// @Router /upload/init [get]
func (h *UploadHandler) InitUpload(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	result, err := h.ossClient.InitiateMultipartUpload(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateUploadPartURL 生成分片上传URL
// @Summary 生成分片上传URL
// @Description 生成分片上传的预签名URL
// @Tags 上传
// @Accept json
// @Produce json
// @Param uploadId query string true "上传ID"
// @Param objectKey query string true "对象键"
// @Param partNumber query int true "分片序号"
// @Success 200 {object} oss.UploadPartInfo
// @Failure 500 {object} map[string]string
// @Router /upload/part-url [get]
func (h *UploadHandler) GenerateUploadPartURL(c *gin.Context) {
	uploadID := c.Query("uploadId")
	objectKey := c.Query("objectKey")
	partNumber, err := strconv.Atoi(c.Query("partNumber"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid partNumber"})
		return
	}

	// 预签名URL有效期为1小时
	result, err := h.ossClient.GenerateUploadPartURL(uploadID, objectKey, partNumber, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CompleteUpload 完成分片上传
// @Summary 完成分片上传
// @Description 完成分片上传并返回文件URL
// @Tags 上传
// @Accept json
// @Produce json
// @Param request body CompleteUploadRequest true "完成上传请求"
// @Success 200 {object} oss.CompleteMultipartUploadResult
// @Failure 500 {object} map[string]string
// @Router /upload/complete [post]
func (h *UploadHandler) CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换parts
	var parts []oss.UploadPart
	for _, p := range req.Parts {
		parts = append(parts, oss.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	result, err := h.ossClient.CompleteMultipartUpload(req.UploadID, req.ObjectKey, parts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type CompleteUploadRequest struct {
	UploadID  string       `json:"uploadId"`
	ObjectKey string       `json:"objectKey"`
	Parts     []UploadPart `json:"parts"`
}

type UploadPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}
