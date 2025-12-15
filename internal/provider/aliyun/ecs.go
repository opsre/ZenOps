package aliyun

import (
	"context"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/eryajf/zenops/internal/model"
)

// ECSQueryParams ECS 实例查询参数
type ECSQueryParams struct {
	// 基础查询参数
	InstanceIDs  []string // 实例 ID 列表，最多支持 100 个
	InstanceName string   // 实例名称，支持通配符 (*)

	// IP 地址查询
	PrivateIPAddresses []string // 内网 IP 地址列表
	PublicIPAddresses  []string // 公网 IP 地址列表
	EipAddresses       []string // 弹性公网 IP 地址列表

	// 实例状态
	Status string // 实例状态: Pending, Running, Starting, Stopping, Stopped

	// 计费方式
	InstanceChargeType string // 计费方式: PostPaid(按量付费), PrePaid(包年包月)

	// 分页参数
	PageSize int // 每页数量，最大 100，默认 10
	PageNum  int // 页码，默认 1
}

// buildDescribeInstancesRequest 构建 DescribeInstances 请求
func (c *Client) buildDescribeInstancesRequest(params *ECSQueryParams) *ecs.DescribeInstancesRequest {
	request := &ecs.DescribeInstancesRequest{
		RegionId: tea.String(c.Region),
	}

	if params == nil {
		return request
	}

	// 基础查询参数
	if len(params.InstanceIDs) > 0 {
		// 构建 JSON 数组格式: ["id1", "id2", ...]
		idsJSON := "["
		for i, id := range params.InstanceIDs {
			if i > 0 {
				idsJSON += ","
			}
			idsJSON += fmt.Sprintf(`"%s"`, id)
		}
		idsJSON += "]"
		request.InstanceIds = tea.String(idsJSON)
	}

	if params.InstanceName != "" {
		request.InstanceName = tea.String(params.InstanceName)
	}

	// IP 地址参数
	if len(params.PrivateIPAddresses) > 0 {
		request.PrivateIpAddresses = tea.String(formatIPArray(params.PrivateIPAddresses))
	}
	if len(params.PublicIPAddresses) > 0 {
		request.PublicIpAddresses = tea.String(formatIPArray(params.PublicIPAddresses))
	}
	if len(params.EipAddresses) > 0 {
		request.EipAddresses = tea.String(formatIPArray(params.EipAddresses))
	}

	// 实例状态
	if params.Status != "" {
		request.Status = tea.String(params.Status)
	}

	// 计费方式
	if params.InstanceChargeType != "" {
		request.InstanceChargeType = tea.String(params.InstanceChargeType)
	}

	// 分页参数
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	pageNum := params.PageNum
	if pageNum <= 0 {
		pageNum = 1
	}
	request.PageSize = tea.Int32(int32(pageSize))
	request.PageNumber = tea.Int32(int32(pageNum))

	return request
}

// formatIPArray 格式化 IP 地址数组为 JSON 字符串
func formatIPArray(ips []string) string {
	if len(ips) == 0 {
		return ""
	}
	result := "["
	for i, ip := range ips {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, ip)
	}
	result += "]"
	return result
}

// QueryECSInstances 查询 ECS 实例列表（增强版）
// 支持更丰富的查询参数，适合 AI 和 MCP 精确查询
func (c *Client) QueryECSInstances(ctx context.Context, params *ECSQueryParams) ([]*model.Instance, error) {
	ecsClient, err := c.GetECSClient()
	if err != nil {
		return nil, err
	}

	if params == nil {
		params = &ECSQueryParams{}
	}

	request := c.buildDescribeInstancesRequest(params)

	logx.Debug("Querying Aliyun ECS instances with enhanced params, region %s, page_size %d, page_num %d",
		c.Region, params.PageSize, params.PageNum)

	response, err := ecsClient.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	if response.Body == nil || response.Body.Instances == nil {
		return []*model.Instance{}, nil
	}

	instances := make([]*model.Instance, 0, len(response.Body.Instances.Instance))
	for _, inst := range response.Body.Instances.Instance {
		instance := convertECSToInstance(inst, c.Region)
		instances = append(instances, instance)
	}

	logx.Info("Successfully queried Aliyun ECS instances, count %d, region %s",
		len(instances), c.Region)

	return instances, nil
}

// ListECSInstances 查询 ECS 实例列表（简化版，保持向后兼容）
func (c *Client) ListECSInstances(ctx context.Context, pageSize, pageNum int, filters map[string]string) ([]*model.Instance, error) {
	// 将简单参数转换为 ECSQueryParams
	params := &ECSQueryParams{
		PageSize: pageSize,
		PageNum:  pageNum,
	}

	// 应用过滤条件
	if instanceName, ok := filters["instance_name"]; ok {
		params.InstanceName = instanceName
	}
	if status, ok := filters["status"]; ok {
		params.Status = status
	}
	if chargeType, ok := filters["instance_charge_type"]; ok {
		params.InstanceChargeType = chargeType
	}

	return c.QueryECSInstances(ctx, params)
}

// GetECSInstanceByQuery 根据查询参数获取单个实例详情（增强版）
// 支持多种查询方式：实例 ID、实例名称、IP 地址等
func (c *Client) GetECSInstanceByQuery(ctx context.Context, params *ECSQueryParams) (*model.Instance, error) {
	if params == nil {
		return nil, fmt.Errorf("query params cannot be nil")
	}

	// 确保只返回一个结果
	params.PageSize = 1
	params.PageNum = 1

	instances, err := c.QueryECSInstances(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("instance not found with given query parameters")
	}

	return instances[0], nil
}

// GetECSInstance 获取 ECS 实例详情（通过实例 ID）
func (c *Client) GetECSInstance(ctx context.Context, instanceID string) (*model.Instance, error) {
	params := &ECSQueryParams{
		InstanceIDs: []string{instanceID},
	}

	instance, err := c.GetECSInstanceByQuery(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s: %w", instanceID, err)
	}

	logx.Info("Successfully queried Aliyun ECS instance, instance_id %s", instanceID)
	return instance, nil
}

// GetECSInstanceByName 根据实例名称获取实例详情
func (c *Client) GetECSInstanceByName(ctx context.Context, instanceName string) (*model.Instance, error) {
	params := &ECSQueryParams{
		InstanceName: instanceName,
	}

	instance, err := c.GetECSInstanceByQuery(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance by name %s: %w", instanceName, err)
	}

	logx.Info("Successfully queried Aliyun ECS instance by name, instance_name %s", instanceName)
	return instance, nil
}

// GetECSInstanceByIP 根据 IP 地址获取实例详情
func (c *Client) GetECSInstanceByIP(ctx context.Context, ipAddress string, ipType string) (*model.Instance, error) {
	params := &ECSQueryParams{}

	switch ipType {
	case "private":
		params.PrivateIPAddresses = []string{ipAddress}
	case "public":
		params.PublicIPAddresses = []string{ipAddress}
	case "eip":
		params.EipAddresses = []string{ipAddress}
	default:
		// 尝试所有类型
		params.PrivateIPAddresses = []string{ipAddress}
		instances, err := c.QueryECSInstances(ctx, params)
		if err == nil && len(instances) > 0 {
			return instances[0], nil
		}

		params.PrivateIPAddresses = nil
		params.PublicIPAddresses = []string{ipAddress}
		instances, err = c.QueryECSInstances(ctx, params)
		if err == nil && len(instances) > 0 {
			return instances[0], nil
		}

		params.PublicIPAddresses = nil
		params.EipAddresses = []string{ipAddress}
	}

	instance, err := c.GetECSInstanceByQuery(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance by IP %s: %w", ipAddress, err)
	}

	logx.Info("Successfully queried Aliyun ECS instance by IP, ip %s, type %s", ipAddress, ipType)
	return instance, nil
}

// convertECSToInstance 将阿里云 ECS 实例转换为统一的实例模型
func convertECSToInstance(inst *ecs.DescribeInstancesResponseBodyInstancesInstance, region string) *model.Instance {
	instance := &model.Instance{
		ID:           tea.StringValue(inst.InstanceId),
		Name:         tea.StringValue(inst.InstanceName),
		Provider:     "aliyun",
		Region:       region,
		Zone:         tea.StringValue(inst.ZoneId),
		InstanceType: tea.StringValue(inst.InstanceType),
		Status:       tea.StringValue(inst.Status),
		CPU:          int(tea.Int32Value(inst.Cpu)),
		Memory:       int(tea.Int32Value(inst.Memory)),
		OSType:       tea.StringValue(inst.OSType),
		OSName:       tea.StringValue(inst.OSName),
		Tags:         make(map[string]string),
		Metadata:     make(map[string]any),
	}

	// 解析创建时间
	if inst.CreationTime != nil {
		creationTime := tea.StringValue(inst.CreationTime)
		// 阿里云时间格式可能是 "2006-01-02T15:04Z" 或 "2006-01-02T15:04:05Z"
		for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04Z"} {
			if t, err := time.Parse(layout, creationTime); err == nil {
				instance.CreatedAt = t
				break
			}
		}
	}

	// 解析过期时间
	if inst.ExpiredTime != nil && tea.StringValue(inst.ExpiredTime) != "" {
		expiredTime := tea.StringValue(inst.ExpiredTime)
		// 阿里云时间格式可能是 "2006-01-02T15:04Z" 或 "2006-01-02T15:04:05Z"
		for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04Z"} {
			if t, err := time.Parse(layout, expiredTime); err == nil {
				instance.ExpiredAt = &t
				break
			}
		}
	}

	// 解析私网 IP
	if inst.VpcAttributes != nil && inst.VpcAttributes.PrivateIpAddress != nil {
		instance.PrivateIP = make([]string, 0, len(inst.VpcAttributes.PrivateIpAddress.IpAddress))
		for _, ip := range inst.VpcAttributes.PrivateIpAddress.IpAddress {
			if ip != nil {
				instance.PrivateIP = append(instance.PrivateIP, tea.StringValue(ip))
			}
		}
	}

	// 解析公网 IP
	if inst.PublicIpAddress != nil && inst.PublicIpAddress.IpAddress != nil {
		instance.PublicIP = make([]string, 0, len(inst.PublicIpAddress.IpAddress))
		for _, ip := range inst.PublicIpAddress.IpAddress {
			if ip != nil {
				instance.PublicIP = append(instance.PublicIP, tea.StringValue(ip))
			}
		}
	}

	// 解析 EIP
	if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		instance.PublicIP = append(instance.PublicIP, tea.StringValue(inst.EipAddress.IpAddress))
	}

	// 解析标签
	if inst.Tags != nil && inst.Tags.Tag != nil {
		for _, tag := range inst.Tags.Tag {
			if tag != nil {
				instance.Tags[tea.StringValue(tag.TagKey)] = tea.StringValue(tag.TagValue)
			}
		}
	}

	// 保存额外的元数据
	instance.Metadata["description"] = tea.StringValue(inst.Description)
	instance.Metadata["image_id"] = tea.StringValue(inst.ImageId)
	instance.Metadata["instance_charge_type"] = tea.StringValue(inst.InstanceChargeType)
	instance.Metadata["internet_charge_type"] = tea.StringValue(inst.InternetChargeType)
	instance.Metadata["internet_max_bandwidth_out"] = tea.Int32Value(inst.InternetMaxBandwidthOut)

	// 生成控制台跳转URL
	instance.ConsoleURL = fmt.Sprintf("https://ecs.console.aliyun.com/server/%s/detail?regionId=%s#/",
		instance.ID, region)

	return instance
}
