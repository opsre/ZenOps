package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	aliyunprovider "github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleListOSS 处理列出 OSS 存储桶的请求
func (s *MCPServer) handleListOSS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)

	aliyunConfig, err := s.getAliyunConfigByName(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 创建临时客户端
	var ossClient *aliyunprovider.Client
	for _, region := range aliyunConfig.Regions {
		c, err := aliyunprovider.NewClient(aliyunConfig.AK, aliyunConfig.SK, region)
		if err == nil {
			ossClient = c
			break
		}
	}
	if ossClient == nil {
		return mcp.NewToolResultError("failed to create OSS client"), nil
	}

	var allBuckets []*model.OSSBucket
	pageNum := 1
	pageSize := 100

	for {
		buckets, err := ossClient.ListOSSBuckets(ctx, pageSize, pageNum, nil)
		if err != nil {
			logx.Error("Failed to list OSS buckets: %v", err)
			break
		}

		allBuckets = append(allBuckets, buckets...)

		if len(buckets) < pageSize {
			break
		}
		pageNum++
	}

	result := formatOSSBuckets(allBuckets, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleGetOSS 处理获取 OSS 存储桶详情的请求
func (s *MCPServer) handleGetOSS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	bucketName, ok := args["bucket_name"].(string)
	if !ok || bucketName == "" {
		return mcp.NewToolResultError("bucket_name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	aliyunConfig, err := s.getAliyunConfigByName(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 创建临时客户端
	var ossClient *aliyunprovider.Client
	for _, region := range aliyunConfig.Regions {
		c, err := aliyunprovider.NewClient(aliyunConfig.AK, aliyunConfig.SK, region)
		if err == nil {
			ossClient = c
			break
		}
	}
	if ossClient == nil {
		return mcp.NewToolResultError("failed to create OSS client"), nil
	}

	bucket, err := ossClient.GetOSSBucket(ctx, bucketName)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到存储桶 %s: %v", bucketName, err)), nil
	}

	result := formatOSSBuckets([]*model.OSSBucket{bucket}, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// formatOSSBuckets 格式化 OSS 存储桶列表
func formatOSSBuckets(buckets []*model.OSSBucket, accountName string) string {
	if len(buckets) == 0 {
		return "未找到任何 OSS 存储桶"
	}

	result := fmt.Sprintf("## 阿里云 OSS 存储桶列表 (账号: %s)\n\n", accountName)
	result += fmt.Sprintf("总数: %d\n\n", len(buckets))

	for _, bucket := range buckets {
		result += fmt.Sprintf("### %s\n", bucket.Name)
		result += fmt.Sprintf("- **区域**: %s\n", bucket.Region)
		result += fmt.Sprintf("- **存储类型**: %s\n", bucket.StorageClass)
		result += fmt.Sprintf("- **创建时间**: %s\n", bucket.CreatedAt)
		if bucket.ACL != "" {
			result += fmt.Sprintf("- **访问控制**: %s\n", bucket.ACL)
		}

		// 显示端点信息
		if extranetEndpoint, ok := bucket.Metadata["extranet_endpoint"].(string); ok && extranetEndpoint != "" {
			result += fmt.Sprintf("- **公网端点**: %s\n", extranetEndpoint)
		}
		if intranetEndpoint, ok := bucket.Metadata["intranet_endpoint"].(string); ok && intranetEndpoint != "" {
			result += fmt.Sprintf("- **内网端点**: %s\n", intranetEndpoint)
		}

		if bucket.ConsoleURL != "" {
			result += fmt.Sprintf("- **控制台**: %s\n", bucket.ConsoleURL)
		}
		result += "\n"
	}

	return result
}

// getAliyunConfigByName 获取指定名称的阿里云账号配置（辅助方法）
func (s *MCPServer) getAliyunConfigByName(accountName string) (*AliyunAccountConfig, error) {
	if len(s.config.Providers.Aliyun) == 0 {
		return nil, fmt.Errorf("no aliyun account configured")
	}

	if accountName == "" {
		for _, acc := range s.config.Providers.Aliyun {
			if acc.Enabled {
				return &AliyunAccountConfig{
					Name:    acc.Name,
					AK:      acc.AK,
					SK:      acc.SK,
					Regions: acc.Regions,
				}, nil
			}
		}
		acc := s.config.Providers.Aliyun[0]
		return &AliyunAccountConfig{
			Name:    acc.Name,
			AK:      acc.AK,
			SK:      acc.SK,
			Regions: acc.Regions,
		}, nil
	}

	for _, acc := range s.config.Providers.Aliyun {
		if acc.Name == accountName {
			return &AliyunAccountConfig{
				Name:    acc.Name,
				AK:      acc.AK,
				SK:      acc.SK,
				Regions: acc.Regions,
			}, nil
		}
	}

	return nil, fmt.Errorf("aliyun account '%s' not found", accountName)
}

// AliyunAccountConfig 阿里云账号配置
type AliyunAccountConfig struct {
	Name    string
	AK      string
	SK      string
	Regions []string
}
