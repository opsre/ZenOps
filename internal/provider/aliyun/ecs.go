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

// ListECSInstances 查询 ECS 实例列表
func (c *Client) ListECSInstances(ctx context.Context, pageSize, pageNum int, filters map[string]string) ([]*model.Instance, error) {
	ecsClient, err := c.GetECSClient()
	if err != nil {
		return nil, err
	}

	if pageSize <= 0 {
		pageSize = 10
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	request := &ecs.DescribeInstancesRequest{
		RegionId:   tea.String(c.Region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNum)),
	}

	// 应用过滤条件
	if instanceName, ok := filters["instance_name"]; ok {
		request.InstanceName = tea.String(instanceName)
	}
	if status, ok := filters["status"]; ok {
		request.Status = tea.String(status)
	}

	logx.Debug("Querying Aliyun ECS instances, region %s, page_size %d, page_num %d",
		c.Region, pageSize, pageNum)

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

// GetECSInstance 获取 ECS 实例详情
func (c *Client) GetECSInstance(ctx context.Context, instanceID string) (*model.Instance, error) {
	ecsClient, err := c.GetECSClient()
	if err != nil {
		return nil, err
	}

	request := &ecs.DescribeInstancesRequest{
		RegionId:    tea.String(c.Region),
		InstanceIds: tea.String(fmt.Sprintf(`["%s"]`, instanceID)),
	}

	logx.Debug("Querying Aliyun ECS instance, instance_id %s, region %s",
		instanceID, c.Region)

	response, err := ecsClient.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if response.Body == nil || response.Body.Instances == nil || len(response.Body.Instances.Instance) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	instance := convertECSToInstance(response.Body.Instances.Instance[0], c.Region)

	logx.Info("Successfully queried Aliyun ECS instance, instance_id %s",
		instanceID)

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
		if t, err := time.Parse("2006-01-02T15:04:05Z", tea.StringValue(inst.CreationTime)); err == nil {
			instance.CreatedAt = t
		}
	}

	// 解析过期时间
	if inst.ExpiredTime != nil && tea.StringValue(inst.ExpiredTime) != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", tea.StringValue(inst.ExpiredTime)); err == nil {
			instance.ExpiredAt = &t
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
