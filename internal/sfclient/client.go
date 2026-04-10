package sfclient

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ProdURL     = "https://sfapi.sf-express.com/std/service"
	SandboxURL  = "https://sfapi-sbox.sf-express.com/std/service"
	ServiceCode = "EXP_RECE_SEARCH_ROUTES"
)

// Config 顺丰对接配置
type Config struct {
	PartnerID string // 顾客编码
	CheckWord string // 校验码
	APIUrl    string // 接口地址
	Timeout   time.Duration
}

// DefaultConfig 使用生产环境默认配置
func DefaultConfig(partnerID, checkWord string) *Config {
	return &Config{
		PartnerID: partnerID,
		CheckWord: checkWord,
		APIUrl:    ProdURL,
		Timeout:   15 * time.Second,
	}
}

// Client 顺丰接口客户端
type Client struct {
	cfg        *Config
	httpClient *http.Client
}

// New 创建客户端
func New(cfg *Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// -------------------------------------------------------------------
// 请求 / 响应结构
// -------------------------------------------------------------------

// RouteQueryRequest 路由查询请求参数
type RouteQueryRequest struct {
	TrackingType    int      `json:"trackingType"`              // 1=运单号 2=订单号
	TrackingNumber  []string `json:"trackingNumber"`            // 单号列表
	CheckPhoneNo    string   `json:"checkPhoneNo,omitempty"`    // 手机后四位（可选）
	MethodType      int      `json:"methodType,omitempty"`      // 路由查询类型，1=标准 (默认)
	ReferenceNumber string   `json:"referenceNumber,omitempty"` // 参考编号
	Language        string   `json:"language,omitempty"`        // 语言代码，默认 zh-CN
}

// ------- 外层响应 -------

type apiResponse struct {
	APIResultCode string `json:"apiResultCode"`
	APIResultData string `json:"apiResultData"`
	APIErrorMsg   string `json:"apiErrorMsg,omitempty"`
}

// ------- apiResultData 内层 -------

type innerResult struct {
	Success  bool            `json:"success"`
	ErrorMsg string          `json:"errorMsg,omitempty"`
	MsgData  json.RawMessage `json:"msgData,omitempty"`
}

// ------- msgData 层 -------

type routeRespsWrapper struct {
	RouteResps []RouteResp `json:"routeResps"`
}

// RouteResp 单个运单路由
type RouteResp struct {
	MailNo   string  `json:"mailNo"`
	Routes   []Route `json:"routes"`
	Accepted bool    `json:"accepted"`
}

// Route 单条路由记录
type Route struct {
	OpCode              string `json:"opCode"`
	AcceptTime          string `json:"acceptTime"`
	AcceptAddress       string `json:"acceptAddress"`
	Remark              string `json:"remark"`
	Longitude           string `json:"longitude,omitempty"`
	Latitude            string `json:"latitude,omitempty"`
	OpName              string `json:"opName,omitempty"`
	FirstStatusCode     string `json:"firstStatusCode"`
	SecondaryStatusName string `json:"secondaryStatusName"`
	SecondaryStatusCode string `json:"secondaryStatusCode"`
	FirstStatusName     string `json:"firstStatusName"`
}

// -------------------------------------------------------------------
// 业务方法
// -------------------------------------------------------------------

// SearchRoutes 查询路由轨迹
func (c *Client) SearchRoutes(req RouteQueryRequest) ([]RouteResp, error) {
	if len(req.TrackingNumber) == 0 {
		return nil, fmt.Errorf("trackingNumber 不能为空")
	}
	if req.TrackingType == 0 {
		req.TrackingType = 1
	}

	msgData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	respBody, err := c.post(ServiceCode, string(msgData))
	if err != nil {
		return nil, err
	}

	var outer apiResponse
	if err := json.Unmarshal(respBody, &outer); err != nil {
		return nil, fmt.Errorf("解析外层响应失败: %w", err)
	}
	if outer.APIResultCode != "A1000" {
		return nil, fmt.Errorf("API 错误 [%s]: %s", outer.APIResultCode, outer.APIErrorMsg)
	}

	var inner innerResult
	if err := json.Unmarshal([]byte(outer.APIResultData), &inner); err != nil {
		return nil, fmt.Errorf("解析 apiResultData 失败: %w", err)
	}
	fmt.Printf("------: %+v\n", inner)
	if !inner.Success || inner.ErrorMsg != "" {
		return nil, fmt.Errorf("业务错误: %s", inner.ErrorMsg)
	}

	var wrapper routeRespsWrapper
	if err := json.Unmarshal(inner.MsgData, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 msgData 失败: %w", err)
	}

	return wrapper.RouteResps, nil
}

// -------------------------------------------------------------------
// 内部工具
// -------------------------------------------------------------------

// post 发送签名 POST 请求，返回原始响应体
func (c *Client) post(serviceCode, msgData string) ([]byte, error) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	digest := c.msgDigest(msgData, timestamp)
	requestID := randomID()

	form := url.Values{}
	form.Set("partnerID", c.cfg.PartnerID)
	form.Set("requestID", requestID)
	form.Set("serviceCode", serviceCode)
	form.Set("timestamp", timestamp)
	form.Set("msgData", msgData)
	form.Set("msgDigest", digest)

	req, err := http.NewRequest(http.MethodPost, c.cfg.APIUrl,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// msgDigest 顺丰签名算法：Base64( MD5( msgData + timestamp + checkWord ) )
func (c *Client) msgDigest(msgData, timestamp string) string {
	raw := msgData + timestamp + c.cfg.CheckWord
	sum := md5.Sum([]byte(raw))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// randomID 生成 32 位请求 ID
func randomID() string {
	return fmt.Sprintf("%x%x", time.Now().UnixNano(), time.Now().UnixMicro())
}
