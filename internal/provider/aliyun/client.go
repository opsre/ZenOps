package aliyun

import (
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	rds "github.com/alibabacloud-go/rds-20140815/v14/client"
	"github.com/alibabacloud-go/tea/tea"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Client 阿里云客户端
type Client struct {
	AccessKeyID     string
	AccessKeySecret string
	Region          string
	ecsClient       *ecs.Client
	rdsClient       *rds.Client
	ossClient       *oss.Client
}

// NewClient 创建阿里云客户端
func NewClient(accessKeyID, accessKeySecret, region string) (*Client, error) {
	if accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("access key id or secret is empty")
	}

	if region == "" {
		region = "cn-hangzhou" // 默认区域
	}

	client := &Client{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Region:          region,
	}

	return client, nil
}

// GetECSClient 获取 ECS 客户端
func (c *Client) GetECSClient() (*ecs.Client, error) {
	if c.ecsClient != nil {
		return c.ecsClient, nil
	}

	endpoint := fmt.Sprintf("ecs.%s.aliyuncs.com", c.Region)

	config := &openapi.Config{
		AccessKeyId:     tea.String(c.AccessKeyID),
		AccessKeySecret: tea.String(c.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	client, err := ecs.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS client: %w", err)
	}

	c.ecsClient = client
	return client, nil
}

// GetRDSClient 获取 RDS 客户端
func (c *Client) GetRDSClient() (*rds.Client, error) {
	if c.rdsClient != nil {
		return c.rdsClient, nil
	}

	endpoint := fmt.Sprintf("rds.%s.aliyuncs.com", c.Region)

	// RDS v2 使用旧版 openapi config
	config := &openapi.Config{
		AccessKeyId:     tea.String(c.AccessKeyID),
		AccessKeySecret: tea.String(c.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	client, err := rds.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	c.rdsClient = client
	return client, nil
}

// GetOSSClient 获取 OSS 客户端
func (c *Client) GetOSSClient() (*oss.Client, error) {
	if c.ossClient != nil {
		return c.ossClient, nil
	}

	// OSS 使用全局 endpoint (不带region前缀)
	// ListBuckets 等全局操作需要使用全局endpoint
	endpoint := "https://oss-cn-hangzhou.aliyuncs.com"

	client, err := oss.New(endpoint, c.AccessKeyID, c.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	c.ossClient = client
	return client, nil
}
