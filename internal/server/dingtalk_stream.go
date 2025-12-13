package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dingtalkcard_1_0 "github.com/alibabacloud-go/dingtalk/card_1_0"
	dingtalkoauth2_1_0 "github.com/alibabacloud-go/dingtalk/oauth2_1_0"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
)

// DingTalkStreamClient 钉钉流式客户端(使用官方SDK)
type DingTalkStreamClient struct {
	appKey      string
	appSecret   string
	templateID  string // AI 卡片模板 ID
	oauthClient *dingtalkoauth2_1_0.Client
	cardClient  *dingtalkcard_1_0.Client
	tokenCache  struct {
		accessToken string
		expireAt    time.Time
	}
	tokenMutex sync.RWMutex
}

// NewDingTalkStreamClient 创建钉钉流式客户端
func NewDingTalkStreamClient(appKey, appSecret, templateID string) (*DingTalkStreamClient, error) {
	config := &openapi.Config{}
	config.Protocol = tea.String("https")
	config.RegionId = tea.String("central")

	oauthClient, err := dingtalkoauth2_1_0.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth client: %w", err)
	}

	cardClient, err := dingtalkcard_1_0.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create card client: %w", err)
	}

	return &DingTalkStreamClient{
		appKey:      appKey,
		appSecret:   appSecret,
		templateID:  templateID,
		oauthClient: oauthClient,
		cardClient:  cardClient,
	}, nil
}

// GetAccessToken 获取访问令牌(带缓存)
func (c *DingTalkStreamClient) GetAccessToken() (string, error) {
	c.tokenMutex.RLock()
	if c.tokenCache.accessToken != "" && time.Now().Before(c.tokenCache.expireAt) {
		token := c.tokenCache.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Double check
	if c.tokenCache.accessToken != "" && time.Now().Before(c.tokenCache.expireAt) {
		return c.tokenCache.accessToken, nil
	}

	request := &dingtalkoauth2_1_0.GetAccessTokenRequest{
		AppKey:    tea.String(c.appKey),
		AppSecret: tea.String(c.appSecret),
	}

	response, tryErr := func() (_resp *dingtalkoauth2_1_0.GetAccessTokenResponse, _e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		_resp, _err := c.oauthClient.GetAccessToken(request)
		if _err != nil {
			return nil, _err
		}
		return _resp, nil
	}()

	if tryErr != nil {
		return "", tryErr
	}

	accessToken := *response.Body.AccessToken
	logx.Info("Got DingTalk access token, expire_at %d", int(*response.Body.ExpireIn))

	c.tokenCache.accessToken = accessToken
	c.tokenCache.expireAt = time.Now().Add(time.Duration(*response.Body.ExpireIn-300) * time.Second)

	return c.tokenCache.accessToken, nil
}

// CreateAndDeliverCard 创建并投递 AI 卡片(实现CardClient接口)
func (c *DingTalkStreamClient) CreateAndDeliverCard(ctx context.Context, trackID, conversationID, conversationType, senderStaffID string) error {
	// 构造消息对象
	msg := &DingTalkMessage{
		ConversationID:   conversationID,
		ConversationType: conversationType,
		SenderStaffID:    senderStaffID,
	}

	return c.createAndDeliverCardInternal(ctx, trackID, msg)
}

// createAndDeliverCardInternal 内部创建卡片方法
func (c *DingTalkStreamClient) createAndDeliverCardInternal(ctx context.Context, trackID string, msg *DingTalkMessage) error {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	headers := &dingtalkcard_1_0.CreateAndDeliverHeaders{}
	headers.XAcsDingtalkAccessToken = tea.String(accessToken)

	cardDataCardParamMap := map[string]*string{
		"content": tea.String(""), // 初始内容为空
	}

	cardData := &dingtalkcard_1_0.CreateAndDeliverRequestCardData{
		CardParamMap: cardDataCardParamMap,
	}

	request := &dingtalkcard_1_0.CreateAndDeliverRequest{
		CardTemplateId: tea.String(c.templateID),
		OutTrackId:     tea.String(trackID),
		CardData:       cardData,
		CallbackType:   tea.String("STREAM"), // 使用 STREAM 模式
		ImGroupOpenSpaceModel: &dingtalkcard_1_0.CreateAndDeliverRequestImGroupOpenSpaceModel{
			SupportForward: tea.Bool(true),
		},
		ImRobotOpenSpaceModel: &dingtalkcard_1_0.CreateAndDeliverRequestImRobotOpenSpaceModel{
			SupportForward: tea.Bool(true),
		},
		UserIdType: tea.Int32(1),
	}

	// 根据会话类型设置 OpenSpaceId
	switch msg.ConversationType {
	case "2": // 群聊
		openSpaceId := fmt.Sprintf("dtv1.card//IM_GROUP.%s", msg.ConversationID)
		request.SetOpenSpaceId(openSpaceId)
		request.SetImGroupOpenDeliverModel(
			&dingtalkcard_1_0.CreateAndDeliverRequestImGroupOpenDeliverModel{
				RobotCode: tea.String(c.appKey),
			})
	case "1": // 单聊
		openSpaceId := fmt.Sprintf("dtv1.card//IM_ROBOT.%s", msg.SenderStaffID)
		request.SetOpenSpaceId(openSpaceId)
		request.SetImRobotOpenDeliverModel(&dingtalkcard_1_0.CreateAndDeliverRequestImRobotOpenDeliverModel{
			SpaceType: tea.String("IM_ROBOT"),
		})
	default:
		return fmt.Errorf("invalid conversation type: %s", msg.ConversationType)
	}

	_, err = c.cardClient.CreateAndDeliverWithOptions(request, headers, &util.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("failed to create and deliver card: %w", err)
	}

	logx.Info("Created and delivered AI card, track_id %s, conversation_type %s", trackID, msg.ConversationType)

	return nil
}

// StreamingUpdate 流式更新卡片内容
func (c *DingTalkStreamClient) StreamingUpdate(trackID, content string, isFinalize bool) error {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	headers := &dingtalkcard_1_0.StreamingUpdateHeaders{
		XAcsDingtalkAccessToken: tea.String(accessToken),
	}

	request := &dingtalkcard_1_0.StreamingUpdateRequest{
		OutTrackId: tea.String(trackID),
		Guid:       tea.String(uuid.New().String()),
		Key:        tea.String("content"), // 更新 content 字段
		Content:    tea.String(content),
		IsFull:     tea.Bool(true),       // 全量更新
		IsFinalize: tea.Bool(isFinalize), // 是否最终版本
		IsError:    tea.Bool(false),
	}

	_, err = c.cardClient.StreamingUpdateWithOptions(request, headers, &util.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	logx.Debug("Streaming update card, track_id %s, content_len %d, finalize %t", trackID, len(content), isFinalize)

	return nil
}

// StreamResponse 流式响应(定时更新)
func (c *DingTalkStreamClient) StreamResponse(ctx context.Context, trackID string, contentCh <-chan string, question string) {
	fullContent := fmt.Sprintf("**%s**\n\n", question)
	initialContent := fullContent
	updateTicker := time.NewTicker(1500 * time.Millisecond) // 1.5秒更新一次
	defer updateTicker.Stop()

	for {
		select {
		case content, ok := <-contentCh:
			if !ok {
				// 通道关闭,发送最终更新
				if err := c.StreamingUpdate(trackID, fullContent, true); err != nil {
					logx.Error("Final streaming update failed: %v", err)
					c.StreamingUpdate(trackID, fullContent+"\n\n⚠️ 部分内容可能未完整显示", true)
				}
				return
			}
			fullContent += content

		case <-updateTicker.C:
			// 定时更新(只有内容变化时才更新)
			if fullContent != initialContent {
				if err := c.StreamingUpdate(trackID, fullContent, false); err != nil {
					logx.Error("Periodic streaming update failed: %v", err)
					// 继续尝试,不中断
				}
				initialContent = fullContent
			}

		case <-ctx.Done():
			// 上下文取消
			c.StreamingUpdate(trackID, fullContent+"\n\n⚠️ 查询已取消", true)
			return
		}
	}
}

// StreamError 发送错误信息
func (c *DingTalkStreamClient) StreamError(trackID string, err error, question string) error {
	content := fmt.Sprintf("**%s**\n\n❌ **查询失败**\n\n错误: %s\n\n💡 请检查参数后重试", question, err.Error())
	return c.StreamingUpdate(trackID, content, true)
}

// StreamInitial 发送初始提示信息
func (c *DingTalkStreamClient) StreamInitial(trackID, question string) error {
	content := fmt.Sprintf("**%s**\n\n⏳ 正在查询,请稍候...", question)
	return c.StreamingUpdate(trackID, content, false)
}
