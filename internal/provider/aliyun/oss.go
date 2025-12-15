package aliyun

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/eryajf/zenops/internal/model"
)

// ListOSSBuckets 查询 OSS Bucket 列表
func (c *Client) ListOSSBuckets(ctx context.Context, pageSize, pageNum int, filters map[string]string) ([]*model.OSSBucket, error) {
	ossClient, err := c.GetOSSClient()
	if err != nil {
		return nil, err
	}

	// 准备 ListBuckets 选项
	options := []oss.Option{}

	// OSS SDK 使用 Marker 进行分页
	markerValue := ""
	if pageNum > 1 && pageSize > 0 {
		// 需要先获取前面页面的最后一个 bucket 名称作为 marker
		for i := 1; i < pageNum; i++ {
			tempOptions := []oss.Option{
				oss.MaxKeys(pageSize),
			}
			if markerValue != "" {
				tempOptions = append(tempOptions, oss.Marker(markerValue))
			}

			response, err := ossClient.ListBuckets(tempOptions...)
			if err != nil {
				return nil, fmt.Errorf("failed to list buckets: %w", err)
			}

			if len(response.Buckets) == 0 {
				break
			}

			// 更新 marker 为最后一个 bucket 的名称
			markerValue = response.Buckets[len(response.Buckets)-1].Name

			// 如果没有更多数据了,退出
			if !response.IsTruncated {
				break
			}
		}
	}

	// 设置页面大小
	if pageSize <= 0 {
		pageSize = 10
	}
	options = append(options, oss.MaxKeys(pageSize))

	// 设置 marker
	if markerValue != "" {
		options = append(options, oss.Marker(markerValue))
	}

	// 应用过滤条件
	if prefix, ok := filters["prefix"]; ok {
		options = append(options, oss.Prefix(prefix))
	}

	logx.Debug("Querying Aliyun OSS buckets, page_size %d, page_num %d, marker %s",
		pageSize, pageNum, markerValue)

	response, err := ossClient.ListBuckets(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	logx.Debug("OSS API response - Buckets count: %d", len(response.Buckets))

	buckets := make([]*model.OSSBucket, 0, len(response.Buckets))
	for _, bucket := range response.Buckets {
		ossBucket := convertOSSBucket(bucket)
		buckets = append(buckets, ossBucket)
	}

	logx.Info("Successfully queried Aliyun OSS buckets, count %d", len(buckets))

	return buckets, nil
}

// GetOSSBucket 获取 OSS Bucket 详情
func (c *Client) GetOSSBucket(ctx context.Context, bucketName string) (*model.OSSBucket, error) {
	ossClient, err := c.GetOSSClient()
	if err != nil {
		return nil, err
	}

	logx.Debug("Querying Aliyun OSS bucket info, bucket_name %s", bucketName)

	// 获取 bucket 信息
	result, err := ossClient.GetBucketInfo(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := convertOSSBucketFromInfo(result.BucketInfo)

	logx.Info("Successfully queried Aliyun OSS bucket info, bucket_name %s", bucketName)

	return bucket, nil
}

// convertOSSBucket 将阿里云 OSS Bucket 转换为统一的 OSS Bucket 模型
func convertOSSBucket(bucket oss.BucketProperties) *model.OSSBucket {
	ossBucket := &model.OSSBucket{
		Name:         bucket.Name,
		Provider:     "aliyun",
		Region:       bucket.Location,
		StorageClass: bucket.StorageClass,
		CreatedAt:    bucket.CreationDate.Format("2006-01-02 15:04:05"),
		Metadata:     make(map[string]any),
	}

	// 保存额外的元数据
	if bucket.Location != "" {
		ossBucket.Metadata["location"] = bucket.Location
	}
	if bucket.Region != "" {
		ossBucket.Metadata["region"] = bucket.Region
	}

	// 生成控制台跳转URL
	ossBucket.ConsoleURL = fmt.Sprintf("https://oss.console.aliyun.com/bucket/%s/object?path=&region=%s",
		ossBucket.Name, ossBucket.Region)

	return ossBucket
}

// convertOSSBucketFromInfo 从 GetBucketInfo 响应转换为统一的 OSS Bucket 模型
func convertOSSBucketFromInfo(bucketInfo oss.BucketInfo) *model.OSSBucket {
	ossBucket := &model.OSSBucket{
		Name:         bucketInfo.Name,
		Provider:     "aliyun",
		Region:       bucketInfo.Location,
		StorageClass: bucketInfo.StorageClass,
		CreatedAt:    bucketInfo.CreationDate.Format("2006-01-02 15:04:05"),
		ACL:          bucketInfo.ACL,
		Metadata:     make(map[string]any),
	}

	// 保存额外的元数据
	if bucketInfo.ExtranetEndpoint != "" {
		ossBucket.Metadata["extranet_endpoint"] = bucketInfo.ExtranetEndpoint
	}
	if bucketInfo.IntranetEndpoint != "" {
		ossBucket.Metadata["intranet_endpoint"] = bucketInfo.IntranetEndpoint
	}
	if bucketInfo.Location != "" {
		ossBucket.Metadata["location"] = bucketInfo.Location
	}
	if bucketInfo.Owner.ID != "" {
		ossBucket.Metadata["owner_id"] = bucketInfo.Owner.ID
		ossBucket.Metadata["owner_display_name"] = bucketInfo.Owner.DisplayName
	}
	if bucketInfo.RedundancyType != "" {
		ossBucket.Metadata["data_redundancy_type"] = bucketInfo.RedundancyType
	}
	if bucketInfo.Versioning != "" {
		ossBucket.Metadata["versioning"] = bucketInfo.Versioning
	}
	if bucketInfo.TransferAcceleration != "" {
		ossBucket.Metadata["transfer_acceleration"] = bucketInfo.TransferAcceleration
	}

	// 生成控制台跳转URL
	ossBucket.ConsoleURL = fmt.Sprintf("https://oss.console.aliyun.com/bucket/%s/object?path=&region=%s",
		ossBucket.Name, ossBucket.Region)

	return ossBucket
}
