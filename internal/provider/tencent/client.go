package tencent

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// Client 腾讯云客户端
type Client struct {
	SecretID  string
	SecretKey string
	Region    string
	cvmClient *cvm.Client
	cdbClient *cdb.Client
	cosClient *cos.Client
}

// NewClient 创建腾讯云客户端
func NewClient(secretID, secretKey, region string) *Client {
	return &Client{
		SecretID:  secretID,
		SecretKey: secretKey,
		Region:    region,
	}
}

// GetCVMClient 获取 CVM 客户端
func (c *Client) GetCVMClient() (*cvm.Client, error) {
	if c.cvmClient != nil {
		return c.cvmClient, nil
	}

	credential := common.NewCredential(c.SecretID, c.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"

	client, err := cvm.NewClient(credential, c.Region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create CVM client: %w", err)
	}

	c.cvmClient = client
	return client, nil
}

// GetCDBClient 获取 CDB 客户端
func (c *Client) GetCDBClient() (*cdb.Client, error) {
	if c.cdbClient != nil {
		return c.cdbClient, nil
	}

	credential := common.NewCredential(c.SecretID, c.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cdb.tencentcloudapi.com"

	client, err := cdb.NewClient(credential, c.Region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDB client: %w", err)
	}

	c.cdbClient = client
	return client, nil
}

// GetCOSClient 获取 COS 客户端 (用于 Service 级别的操作，如 ListBuckets)
func (c *Client) GetCOSClient() *cos.Client {
	if c.cosClient != nil {
		return c.cosClient
	}

	// 使用 service API 地址用于 GetService (ListBuckets)
	u, _ := url.Parse(fmt.Sprintf("https://cos.%s.myqcloud.com", c.Region))
	b := &cos.BaseURL{ServiceURL: u}

	client := cos.NewClient(b, &http.Client{
		Timeout: 100 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.SecretID,
			SecretKey: c.SecretKey,
		},
	})

	c.cosClient = client
	return client
}

// GetCOSBucketClient 获取指定 Bucket 的 COS 客户端
func (c *Client) GetCOSBucketClient(bucketName string) *cos.Client {
	// bucket URL 格式: https://<bucket-name>.cos.<region>.myqcloud.com
	u, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucketName, c.Region))
	b := &cos.BaseURL{BucketURL: u}

	return cos.NewClient(b, &http.Client{
		Timeout: 100 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.SecretID,
			SecretKey: c.SecretKey,
		},
	})
}
