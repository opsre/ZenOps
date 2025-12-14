package tencent

import (
	"context"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// 腾讯云区域ID映射 (用于控制台URL)
var tencentRegionIDMap = map[string]int{
	"ap-beijing":       1,
	"ap-shanghai":      4,
	"ap-guangzhou":     5,
	"ap-nanjing":       33,
	"ap-hongkong":      8,
	"ap-singapore":     9,
	"ap-bangkok":       23,
	"ap-mumbai":        21,
	"ap-seoul":         18,
	"ap-tokyo":         25,
	"na-siliconvalley": 15,
	"na-ashburn":       22,
	"eu-frankfurt":     17,
	"ap-chengdu":       16,
	"ap-chongqing":     19,
	"ap-taipei":        39,
	"ap-jakarta":       72,
	"eu-moscow":        24,
	"sa-saopaulo":      27,
}

// ListCVMInstances 列出 CVM 实例
func (p *TencentProvider) ListCVMInstances(ctx context.Context, opts *provider.QueryOptions) ([]*model.Instance, error) {
	var allInstances []*model.Instance

	// 如果指定了区域,只查询该区域
	if opts.Region != "" {
		client, exists := p.clients[opts.Region]
		if !exists {
			return nil, fmt.Errorf("region %s not configured", opts.Region)
		}

		instances, err := p.listCVMInstancesInRegion(ctx, client, opts)
		if err != nil {
			return nil, err
		}
		allInstances = append(allInstances, instances...)
	} else {
		// 查询所有区域
		for region, client := range p.clients {
			logx.Debug("Querying CVM instances in region %s", region)

			instances, err := p.listCVMInstancesInRegion(ctx, client, opts)
			if err != nil {
				logx.Warn("Failed to query region %s, error %v", region, err)
				continue
			}

			allInstances = append(allInstances, instances...)
		}
	}

	return allInstances, nil
}

// listCVMInstancesInRegion 查询单个区域的 CVM 实例
func (p *TencentProvider) listCVMInstancesInRegion(ctx context.Context, client *Client, opts *provider.QueryOptions) ([]*model.Instance, error) {
	cvmClient, err := client.GetCVMClient()
	if err != nil {
		return nil, err
	}

	request := cvm.NewDescribeInstancesRequest()

	// 设置分页参数
	if opts.PageSize > 0 {
		limit := int64(opts.PageSize)
		request.Limit = &limit
	}

	if opts.PageNum > 1 {
		offset := int64((opts.PageNum - 1) * opts.PageSize)
		request.Offset = &offset
	}

	response, err := cvmClient.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	var instances []*model.Instance
	for _, inst := range response.Response.InstanceSet {
		instances = append(instances, convertCVMToInstance(inst, client.Region))
	}

	return instances, nil
}

// GetCVMInstance 获取 CVM 实例详情
func (p *TencentProvider) GetCVMInstance(ctx context.Context, instanceID string) (*model.Instance, error) {
	// 遍历所有区域查找实例
	for region, client := range p.clients {
		logx.Debug("Searching instance in region %s, instance_id %s", region, instanceID)

		cvmClient, err := client.GetCVMClient()
		if err != nil {
			logx.Warn("Failed to get CVM client, region %s, error %v", region, err)
			continue
		}

		request := cvm.NewDescribeInstancesRequest()
		request.InstanceIds = []*string{&instanceID}

		response, err := cvmClient.DescribeInstances(request)
		if err != nil {
			logx.Warn("Failed to describe instance, region %s, error %v", region, err)
			continue
		}

		if len(response.Response.InstanceSet) > 0 {
			return convertCVMToInstance(response.Response.InstanceSet[0], region), nil
		}
	}

	return nil, fmt.Errorf("instance %s not found in any region", instanceID)
}

// convertCVMToInstance 将腾讯云 CVM 实例转换为统一的 Instance 模型
func convertCVMToInstance(inst *cvm.Instance, region string) *model.Instance {
	instance := &model.Instance{
		Provider: "tencent",
		Region:   region,
		Tags:     make(map[string]string),
		Metadata: make(map[string]any),
	}

	// 基本信息
	if inst.InstanceId != nil {
		instance.ID = *inst.InstanceId
	}
	if inst.InstanceName != nil {
		instance.Name = *inst.InstanceName
	}
	if inst.InstanceType != nil {
		instance.InstanceType = *inst.InstanceType
	}
	if inst.InstanceState != nil {
		instance.Status = *inst.InstanceState
	}

	// 可用区
	if inst.Placement != nil && inst.Placement.Zone != nil {
		instance.Zone = *inst.Placement.Zone
	}

	// 内网 IP
	if len(inst.PrivateIpAddresses) > 0 {
		for _, ip := range inst.PrivateIpAddresses {
			if ip != nil {
				instance.PrivateIP = append(instance.PrivateIP, *ip)
			}
		}
	}

	// 公网 IP
	if len(inst.PublicIpAddresses) > 0 {
		for _, ip := range inst.PublicIpAddresses {
			if ip != nil {
				instance.PublicIP = append(instance.PublicIP, *ip)
			}
		}
	}

	// CPU 和内存
	if inst.CPU != nil {
		instance.CPU = int(*inst.CPU)
	}
	if inst.Memory != nil {
		instance.Memory = int(*inst.Memory)
	}

	// 操作系统
	if inst.OsName != nil {
		instance.OSType = *inst.OsName
	}

	// 创建时间
	if inst.CreatedTime != nil {
		instance.Metadata["created_time"] = *inst.CreatedTime
		// 解析创建时间到 CreatedAt 字段
		if t, err := time.Parse("2006-01-02T15:04:05Z", *inst.CreatedTime); err == nil {
			instance.CreatedAt = t
		}
	}

	// VPC 信息
	if inst.VirtualPrivateCloud != nil {
		if inst.VirtualPrivateCloud.VpcId != nil {
			instance.Metadata["vpc_id"] = *inst.VirtualPrivateCloud.VpcId
		}
		if inst.VirtualPrivateCloud.SubnetId != nil {
			instance.Metadata["subnet_id"] = *inst.VirtualPrivateCloud.SubnetId
		}
	}

	// 计费模式
	if inst.InstanceChargeType != nil {
		instance.Metadata["charge_type"] = *inst.InstanceChargeType
	}

	// 过期时间
	if inst.ExpiredTime != nil {
		instance.Metadata["expired_time"] = *inst.ExpiredTime
		// 解析过期时间到 ExpiredAt 字段
		if *inst.ExpiredTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05Z", *inst.ExpiredTime); err == nil {
				instance.ExpiredAt = &t
			}
		}
	}

	// 镜像 ID
	if inst.ImageId != nil {
		instance.Metadata["image_id"] = *inst.ImageId
	}

	// 标签
	if inst.Tags != nil {
		for _, tag := range inst.Tags {
			if tag.Key != nil && tag.Value != nil {
				instance.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	// 生成控制台跳转URL
	regionID, ok := tencentRegionIDMap[region]
	if !ok {
		// 如果找不到映射,使用默认值1(北京)
		regionID = 1
	}
	instance.ConsoleURL = fmt.Sprintf("https://console.cloud.tencent.com/cvm/instance/detail?rid=%d&id=%s",
		regionID, instance.ID)

	return instance
}
