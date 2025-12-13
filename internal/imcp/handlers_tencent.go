package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/mark3labs/mcp-go/mcp"
)

// ==================== 腾讯云 CVM 处理函数 ====================

// handleSearchCVMByIP 处理根据 IP 搜索腾讯云 CVM 的请求
func (s *MCPServer) handleSearchCVMByIP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	ip, ok := args["ip"].(string)
	if !ok || ip == "" {
		return mcp.NewToolResultError("ip parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var matchedInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(ctx, opts)
		if err != nil {
			logx.Error("Failed to list instances: %v", err)
			break
		}

		for _, inst := range instances {
			matched := false
			for _, privateIP := range inst.PrivateIP {
				if privateIP == ip {
					matchedInstances = append(matchedInstances, inst)
					matched = true
					break
				}
			}
			if !matched {
				for _, publicIP := range inst.PublicIP {
					if publicIP == ip {
						matchedInstances = append(matchedInstances, inst)
						break
					}
				}
			}
		}

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedInstances) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到 IP 为 %s 的腾讯云 CVM 实例", ip)), nil
	}

	result := formatInstances(matchedInstances, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleSearchCVMByName 处理根据名称搜索腾讯云 CVM 的请求
func (s *MCPServer) handleSearchCVMByName(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var matchedInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(ctx, opts)
		if err != nil {
			logx.Error("Failed to list instances: %v", err)
			break
		}

		for _, inst := range instances {
			if inst.Name == name {
				matchedInstances = append(matchedInstances, inst)
			}
		}

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedInstances) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到名称为 %s 的腾讯云 CVM 实例", name)), nil
	}

	result := formatInstances(matchedInstances, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleListCVM 处理列出腾讯云 CVM 实例的请求
func (s *MCPServer) handleListCVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var allInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(ctx, opts)
		if err != nil {
			logx.Error("Failed to list instances: %v", err)
			break
		}

		allInstances = append(allInstances, instances...)

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	result := formatInstances(allInstances, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleGetCVM 处理获取腾讯云 CVM 实例详情的请求
func (s *MCPServer) handleGetCVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	instanceID, ok := args["instance_id"].(string)
	if !ok || instanceID == "" {
		return mcp.NewToolResultError("instance_id parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	instance, err := p.GetInstance(ctx, instanceID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到实例 ID 为 %s 的腾讯云 CVM 实例: %v", instanceID, err)), nil
	}

	result := formatInstances([]*model.Instance{instance}, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// ==================== 腾讯云 CDB 处理函数 ====================

// handleListCDB 处理列出腾讯云 CDB 实例的请求
func (s *MCPServer) handleListCDB(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}

	accountName, _ := args["account"].(string)
	region, _ := args["region"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
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

	result := formatDatabases(allDatabases, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}

// handleSearchCDBByName 处理根据名称搜索腾讯云 CDB 的请求
func (s *MCPServer) handleSearchCDBByName(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	accountName, _ := args["account"].(string)

	p, tencentConfig, err := s.getTencentProvider(accountName)
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
		return mcp.NewToolResultText(fmt.Sprintf("未找到名称为 %s 的腾讯云 CDB 实例", name)), nil
	}

	result := formatDatabases(matchedDatabases, tencentConfig.Name)
	return mcp.NewToolResultText(result), nil
}
