package tencent

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// Client 腾讯云客户端
type Client struct {
	SecretID  string
	SecretKey string
	Region    string
	cvmClient *cvm.Client
	cdbClient *cdb.Client
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
