package aliyun

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	CallbackURL     string
}

type OSSClient struct {
	client *oss.Client
	bucket *oss.Bucket
	config *OSSConfig
}

func NewOSSClient(config *OSSConfig) (*OSSClient, error) {
	client, err := oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(config.BucketName)
	if err != nil {
		return nil, err
	}

	return &OSSClient{
		client: client,
		bucket: bucket,
		config: config,
	}, nil
}

// InitiateMultipartUploadResult 分片上传初始化结果
type InitiateMultipartUploadResult struct {
	UploadID   string `json:"uploadId"`
	BucketName string `json:"bucketName"`
	ObjectKey  string `json:"objectKey"`
}

// InitiateMultipartUpload 初始化分片上传
func (o *OSSClient) InitiateMultipartUpload(objectKey string) (*InitiateMultipartUploadResult, error) {
	// 生成唯一对象键
	uniqueKey := generateObjectKey(objectKey)

	// 初始化分片上传
	imur, err := o.bucket.InitiateMultipartUpload(uniqueKey)
	if err != nil {
		return nil, err
	}

	return &InitiateMultipartUploadResult{
		UploadID:   imur.UploadID,
		BucketName: imur.Bucket,
		ObjectKey:  imur.Key,
	}, nil
}

// generateObjectKey 生成唯一的对象键
func generateObjectKey(originalName string) string {
	return fmt.Sprintf("uploads/%s/%d_%s", time.Now().Format("2006_01_02"), time.Now().UnixNano(), originalName)
}

// UploadPartInfo 分片上传信息
type UploadPartInfo struct {
	UploadID       string `json:"uploadId"`
	BucketName     string `json:"bucketName"`
	ObjectKey      string `json:"objectKey"`
	PartNumber     int    `json:"partNumber"`
	UploadURL      string `json:"uploadUrl"`
	ExpirationTime int64  `json:"expirationTime"`
}

// GenerateUploadPartURL 生成分片上传预签名URL
func (o *OSSClient) GenerateUploadPartURL(uploadID, objectKey string, partNumber int, expires time.Duration) (*UploadPartInfo, error) {
	sec := int64(expires.Seconds())

	options := []oss.Option{
		oss.AddParam("uploadId", uploadID),
		oss.AddParam("partNumber", strconv.Itoa(partNumber)),
		oss.ACReqMethod("PUT"),
		oss.Expires(time.Now().Add(time.Duration(sec) * time.Second)),
		oss.ContentType("application/octet-stream"),
		oss.Origin("http://localhost:8080"),
		oss.AddParam("response-content-type", "application/json"),
		oss.AddParam("response-expires", "0"),
		oss.AddParam("response-cache-control", "no-cache"),
		oss.AddParam("response-access-control-allow-headers", "*"),
		oss.AddParam("response-access-control-expose-headers", "ETag,x-oss-request-id"),
	}

	signedURL, err := o.bucket.SignURL(objectKey, oss.HTTPPut, sec, options...)
	if err != nil {
		return nil, err
	}

	return &UploadPartInfo{
		UploadID:       uploadID,
		BucketName:     o.config.BucketName,
		ObjectKey:      objectKey,
		PartNumber:     partNumber,
		UploadURL:      signedURL,
		ExpirationTime: time.Now().Add(time.Duration(sec) * time.Second).Unix(),
	}, nil
}

// CompleteMultipartUploadResult 完成分片上传结果
type CompleteMultipartUploadResult struct {
	Location   string `json:"location"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	ETag       string `json:"eTag"`
	PrivateURL string `json:"privateURL"`
	PublicURL  string `json:"publicURL"`
	Expiration int64  `json:"expiration"`
}

// CompleteMultipartUpload 完成分片上传
func (o *OSSClient) CompleteMultipartUpload(uploadID, objectKey string, parts []oss.UploadPart) (*CompleteMultipartUploadResult, error) {
	// 确保 parts 按照 PartNumber 排序
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.config.BucketName,
		Key:      objectKey,
		UploadID: uploadID,
	}

	// 转换 parts 为 OSS SDK 需要的格式
	uploadParts := make([]oss.UploadPart, len(parts))
	for i, part := range parts {
		uploadParts[i] = oss.UploadPart{
			PartNumber: part.PartNumber,
			ETag:       strings.Trim(part.ETag, "\""), // 确保移除 ETag 中的引号
		}
	}

	cmur, err := o.bucket.CompleteMultipartUpload(imur, uploadParts)
	if err != nil {
		return nil, fmt.Errorf("complete multipart upload failed: %w", err)
	}

	// 生成可访问的URL
	url := fmt.Sprintf("https://%s.%s/%s", o.config.BucketName, o.config.Endpoint, objectKey)
	// 生成公开URL， 通过私有访问URL和密钥， 生成一个临时的公共访问URL
	publicURL, err := o.GeneratePublicURL(objectKey, time.Second*10)
	if err != nil {
		return nil, fmt.Errorf("generate public URL failed: %w", err)
	}

	return &CompleteMultipartUploadResult{
		Location:   cmur.Location,
		Bucket:     cmur.Bucket,
		Key:        cmur.Key,
		ETag:       cmur.ETag,
		PrivateURL: url,
		PublicURL:  publicURL,
		Expiration: 0,
	}, nil
}

// GeneratePublicURL 生成公开URL
func (o *OSSClient) GeneratePublicURL(objectKey string, exp time.Duration) (string, error) {
	options := []oss.Option{
		//oss.ResponseContentType("application/octet-stream"),
		oss.ResponseContentDisposition(fmt.Sprintf("attachment; filename=\"%s\"", objectKey)),
	}
	// 生成签名URL，设置过期时间（例如1小时后过期）
	expires := time.Now().Add(exp)
	signedURL, err := o.bucket.SignURL(objectKey, oss.HTTPGet, int64(expires.Second()), options...) // 3600秒=1小时
	if err != nil {
		return "", err
	}

	return signedURL, nil
}
