package jenkins

import (
	"context"
	"fmt"

	"github.com/bndr/gojenkins"
)

// Client Jenkins 客户端
type Client struct {
	URL      string
	Username string
	Token    string
	jenkins  *gojenkins.Jenkins
}

// NewClient 创建 Jenkins 客户端
func NewClient(url, username, token string) *Client {
	return &Client{
		URL:      url,
		Username: username,
		Token:    token,
	}
}

// Connect 连接到 Jenkins
func (c *Client) Connect(ctx context.Context) error {
	if c.jenkins != nil {
		return nil
	}

	jenkins := gojenkins.CreateJenkins(nil, c.URL, c.Username, c.Token)
	_, err := jenkins.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Jenkins: %w", err)
	}

	c.jenkins = jenkins
	return nil
}

// GetJenkins 获取 Jenkins 客户端实例
func (c *Client) GetJenkins() *gojenkins.Jenkins {
	return c.jenkins
}
