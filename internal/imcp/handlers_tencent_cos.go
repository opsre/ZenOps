package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleListCOS 处理列出 COS 存储桶的请求
func (s *MCPServer) handleListCOS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)

	tencentConfig, err := s.getTencentConfigByName(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get provider: %v", err)), nil
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize provider: %v", err)), nil
	}

	var allBuckets []*model.OSSBucket
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		buckets, err := p.ListOSSBuckets(ctx, opts)
		if err != nil {
			logx.Error("Failed to list COS buckets: %v", err)
			break
		}

		allBuckets = append(allBuckets, buckets...)

		if len(buckets) < pageSize {
			break
		}
		pageNum++
	}

	result := formatCOSBuckets(allBuckets, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleGetCOS 处理获取 COS 存储桶详情的请求
func (s *MCPServer) handleGetCOS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	bucketName, ok := args["bucket_name"].(string)
	if !ok || bucketName == "" {
		return mcp.NewToolResultError("bucket_name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	tencentConfig, err := s.getTencentConfigByName(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get provider: %v", err)), nil
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize provider: %v", err)), nil
	}

	bucket, err := p.GetOSSBucket(ctx, bucketName)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到存储桶 %s: %v", bucketName, err)), nil
	}

	result := formatCOSBuckets([]*model.OSSBucket{bucket}, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// formatCOSBuckets 格式化 COS 存储桶列表
func formatCOSBuckets(buckets []*model.OSSBucket, accountName string) string {
	if len(buckets) == 0 {
		return "未找到任何 COS 存储桶"
	}

	result := fmt.Sprintf("## 腾讯云 COS 存储桶列表 (账号: %s)\n\n", accountName)
	result += fmt.Sprintf("总数: %d\n\n", len(buckets))

	for _, bucket := range buckets {
		result += fmt.Sprintf("### %s\n", bucket.Name)
		result += fmt.Sprintf("- **区域**: %s\n", bucket.Region)
		if bucket.StorageClass != "" {
			result += fmt.Sprintf("- **存储类型**: %s\n", bucket.StorageClass)
		}
		result += fmt.Sprintf("- **创建时间**: %s\n", bucket.CreatedAt)
		if bucket.ACL != "" {
			result += fmt.Sprintf("- **访问控制**: %s\n", bucket.ACL)
		}

		if bucket.ConsoleURL != "" {
			result += fmt.Sprintf("- **控制台**: %s\n", bucket.ConsoleURL)
		}
		result += "\n"
	}

	return result
}

// getTencentConfigByName 获取指定名称的腾讯云账号配置（辅助方法）
func (s *MCPServer) getTencentConfigByName(accountName string) (*TencentAccountConfig, error) {
	// 使用 helpers.go 中的方法从数据库或配置获取
	cfg, err := s.getTencentConfigByNameFromDB(accountName)
	if err != nil {
		return nil, err
	}

	return &TencentAccountConfig{
		Name:    cfg.Name,
		AK:      cfg.AK,
		SK:      cfg.SK,
		Regions: cfg.Regions,
	}, nil
}

// TencentAccountConfig 腾讯云账号配置
type TencentAccountConfig struct {
	Name    string
	AK      string
	SK      string
	Regions []string
}
