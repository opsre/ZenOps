package tencent

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// ListCOSBuckets 查询 COS Bucket 列表
func (c *Client) ListCOSBuckets(ctx context.Context, pageSize, pageNum int, filters map[string]string) ([]*model.OSSBucket, error) {
	cosClient := c.GetCOSClient()

	logx.Debug("Querying Tencent COS buckets")

	// COS SDK 的 GetService 方法列出所有 buckets
	result, _, err := cosClient.Service.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	if result == nil || result.Buckets == nil {
		logx.Warn("COS GetService response is nil")
		return []*model.OSSBucket{}, nil
	}

	logx.Debug("COS API response - Buckets count: %d", len(result.Buckets))

	// 转换所有 buckets
	allBuckets := make([]*model.OSSBucket, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		cosBucket := convertCOSBucket(bucket)
		allBuckets = append(allBuckets, cosBucket)
	}

	// 应用过滤条件
	var filteredBuckets []*model.OSSBucket
	if prefix, ok := filters["prefix"]; ok && prefix != "" {
		for _, bucket := range allBuckets {
			if len(bucket.Name) >= len(prefix) && bucket.Name[:len(prefix)] == prefix {
				filteredBuckets = append(filteredBuckets, bucket)
			}
		}
	} else {
		filteredBuckets = allBuckets
	}

	// 手动实现分页 (因为 COS GetService 不支持分页参数)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	start := (pageNum - 1) * pageSize
	end := start + pageSize

	if start >= len(filteredBuckets) {
		return []*model.OSSBucket{}, nil
	}

	if end > len(filteredBuckets) {
		end = len(filteredBuckets)
	}

	buckets := filteredBuckets[start:end]

	logx.Info("Successfully queried Tencent COS buckets, count %d", len(buckets))

	return buckets, nil
}

// GetCOSBucket 获取 COS Bucket 详情
func (c *Client) GetCOSBucket(ctx context.Context, bucketName string) (*model.OSSBucket, error) {
	logx.Debug("Querying Tencent COS bucket info, bucket_name %s", bucketName)

	// 获取 bucket 专用客户端
	bucketClient := c.GetCOSBucketClient(bucketName)

	// 获取 bucket ACL
	aclResult, _, err := bucketClient.Bucket.GetACL(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket ACL: %w", err)
	}

	// 获取 bucket location
	locationResult, _, err := bucketClient.Bucket.GetLocation(ctx)
	if err != nil {
		logx.Warn("Failed to get bucket location: %v", err)
	}

	bucket := convertCOSBucketFromDetail(bucketName, aclResult, locationResult)

	logx.Info("Successfully queried Tencent COS bucket info, bucket_name %s", bucketName)

	return bucket, nil
}

// convertCOSBucket 将腾讯云 COS Bucket 转换为统一的 OSS Bucket 模型
func convertCOSBucket(bucket cos.Bucket) *model.OSSBucket {
	cosBucket := &model.OSSBucket{
		Name:      bucket.Name,
		Provider:  "tencent",
		Region:    bucket.Region,
		CreatedAt: bucket.CreationDate,
		Metadata:  make(map[string]any),
	}

	// 保存额外的元数据
	if bucket.BucketType != "" {
		cosBucket.Metadata["bucket_type"] = bucket.BucketType
	}
	if bucket.Type != "" {
		cosBucket.Metadata["type"] = bucket.Type
	}

	// 生成控制台跳转URL
	cosBucket.ConsoleURL = fmt.Sprintf("https://console.cloud.tencent.com/cos/bucket?bucket=%s&region=%s",
		cosBucket.Name, cosBucket.Region)

	return cosBucket
}

// convertCOSBucketFromDetail 从详细信息转换为统一的 OSS Bucket 模型
func convertCOSBucketFromDetail(bucketName string, aclResult *cos.BucketGetACLResult, locationResult *cos.BucketGetLocationResult) *model.OSSBucket {
	cosBucket := &model.OSSBucket{
		Name:     bucketName,
		Provider: "tencent",
		Metadata: make(map[string]any),
	}

	// 从 location result 获取区域信息
	if locationResult != nil && locationResult.Location != "" {
		cosBucket.Region = locationResult.Location
	}

	// 从 ACL result 获取权限信息和所有者信息
	if aclResult != nil {
		if aclResult.Owner != nil {
			cosBucket.Metadata["owner_id"] = aclResult.Owner.ID
			cosBucket.Metadata["owner_display_name"] = aclResult.Owner.DisplayName
		}
		// ACL grants
		if len(aclResult.AccessControlList) > 0 {
			var grants []string
			for _, grant := range aclResult.AccessControlList {
				grants = append(grants, grant.Permission)
			}
			if len(grants) > 0 {
				cosBucket.ACL = grants[0] // 使用第一个权限作为主要 ACL
			}
		}
	}

	// 生成控制台跳转URL
	cosBucket.ConsoleURL = fmt.Sprintf("https://console.cloud.tencent.com/cos/bucket?bucket=%s&region=%s",
		cosBucket.Name, cosBucket.Region)

	return cosBucket
}
