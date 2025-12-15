package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleSearchECSByIP 处理根据 IP 搜索 ECS 的请求
func (s *MCPServer) handleSearchECSByIP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	ip, ok := args["ip"].(string)
	if !ok || ip == "" {
		return mcp.NewToolResultError("ip parameter is required"), nil
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)
	ipType, _ := args["ip_type"].(string) // private, public, eip

	client, aliyunConfig, err := s.getAliyunClient(accountName, region)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 使用增强的 IP 查询功能
	instance, err := client.GetECSInstanceByIP(ctx, ip, ipType)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到 IP 为 %s 的 ECS 实例: %v", ip, err)), nil
	}

	result := formatInstances([]*model.Instance{instance}, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleSearchECSByName 处理根据名称搜索 ECS 的请求
func (s *MCPServer) handleSearchECSByName(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)

	client, aliyunConfig, err := s.getAliyunClient(accountName, region)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 使用增强的名称查询功能
	instance, err := client.GetECSInstanceByName(ctx, name)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到名称为 %s 的 ECS 实例: %v", name, err)), nil
	}

	result := formatInstances([]*model.Instance{instance}, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleListECS 处理列出 ECS 实例的请求
func (s *MCPServer) handleListECS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)
	status, _ := args["status"].(string)
	chargeType, _ := args["instance_charge_type"].(string)

	client, aliyunConfig, err := s.getAliyunClient(accountName, region)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 使用增强的查询参数
	var allInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		params := &aliyun.ECSQueryParams{
			Status:             status,
			InstanceChargeType: chargeType,
			PageSize:           pageSize,
			PageNum:            pageNum,
		}

		instances, err := client.QueryECSInstances(ctx, params)
		if err != nil {
			logx.Error("Failed to query instances: %v", err)
			break
		}

		allInstances = append(allInstances, instances...)

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	result := formatInstances(allInstances, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleGetECS 处理获取 ECS 实例详情的请求
func (s *MCPServer) handleGetECS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	instanceID, ok := args["instance_id"].(string)
	if !ok || instanceID == "" {
		return mcp.NewToolResultError("instance_id parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, aliyunConfig, err := s.getAliyunProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	instance, err := p.GetInstance(ctx, instanceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到实例 ID 为 %s 的 ECS 实例: %v", instanceID, err)), nil
	}

	result := formatInstances([]*model.Instance{instance}, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleListRDS 处理列出 RDS 实例的请求
func (s *MCPServer) handleListRDS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)

	p, aliyunConfig, err := s.getAliyunProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var allDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(ctx, opts)
		if err != nil {
			logx.Error("Failed to list databases: %v", err)
			break
		}

		allDatabases = append(allDatabases, databases...)

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	result := formatDatabases(allDatabases, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleSearchRDSByName 处理根据名称搜索 RDS 的请求
func (s *MCPServer) handleSearchRDSByName(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, aliyunConfig, err := s.getAliyunProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var matchedDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(ctx, opts)
		if err != nil {
			logx.Error("Failed to list databases: %v", err)
			break
		}

		for _, db := range databases {
			if db.Name == name {
				matchedDatabases = append(matchedDatabases, db)
			}
		}

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedDatabases) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到名称为 %s 的 RDS 实例", name)), nil
	}

	result := formatDatabases(matchedDatabases, aliyunConfig.Name)
	return mcp.NewToolResultText(result), nil
}
