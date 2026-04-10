package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"sfquery/internal/sfclient"
)

// -------------------------------------------------------------------
// JSON 响应结构
// -------------------------------------------------------------------

type response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// RouteItem 前端展示用路由条目
type RouteItem struct {
	OpCode  string `json:"opCode"`
	Time    string `json:"time"`
	Address string `json:"address"`
	Remark  string `json:"remark"`
	IsFirst bool   `json:"isFirst"`
}

// RouteResult 前端展示用查询结果
type RouteResult struct {
	MailNo     string      `json:"mailNo"`
	Status     string      `json:"status"`
	StatusCode string      `json:"statusCode"`
	Routes     []RouteItem `json:"routes"`
	Total      int         `json:"total"`
}

// -------------------------------------------------------------------
// Handler 封装
// -------------------------------------------------------------------

// Handler 持有 SF 客户端的处理器集合
type Handler struct {
	sf *sfclient.Client
}

// New 创建 Handler
func New(sf *sfclient.Client) *Handler {
	return &Handler{sf: sf}
}

// RegisterRoutes 注册所有 HTTP 路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/route/query", h.QueryRoute)
	mux.HandleFunc("/health", h.Health)
}

// -------------------------------------------------------------------
// /api/route/query
// -------------------------------------------------------------------

type queryRequest struct {
	TrackingType   int    `json:"trackingType"`
	TrackingNumber string `json:"trackingNumber"`
	CheckPhoneNo   string `json:"checkPhoneNo,omitempty"`
}

// QueryRoute POST /api/route/query
func (h *Handler) QueryRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, http.StatusMethodNotAllowed, "仅支持 POST 方法")
		return
	}

	// ---------- 解析入参 ----------
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "请求体 JSON 解析失败: "+err.Error())
		return
	}

	trackNo := strings.TrimSpace(req.TrackingNumber)
	if trackNo == "" {
		jsonErr(w, http.StatusBadRequest, "trackingNumber 不能为空")
		return
	}
	if req.TrackingType == 0 {
		req.TrackingType = 1
	}

	sfReq := sfclient.RouteQueryRequest{
		TrackingType:   req.TrackingType,
		TrackingNumber: []string{trackNo},
		CheckPhoneNo:   strings.TrimSpace(req.CheckPhoneNo),
	}

	// ---------- 调用顺丰接口 ----------
	log.Printf("[SF] query trackingNumber=%s type=%d", trackNo, req.TrackingType)
	routeResps, err := h.sf.SearchRoutes(sfReq)
	if err != nil {
		log.Printf("[SF] error: %v", err)
		jsonErr(w, http.StatusBadGateway, err.Error())
		return
	}

	// ---------- 组装前端响应 ----------
	//results := make([]RouteResult, 0, len(routeResps))
	//for _, rr := range routeResps {
	//	routes := buildRouteItems(rr.Routes)
	//	statusCode, statusLabel := resolveStatus(routes)
	//	results = append(results, RouteResult{
	//		MailNo:     rr.MailNo,
	//		Status:     statusLabel,
	//		StatusCode: statusCode,
	//		Routes:     routes,
	//		Total:      len(routes),
	//	})
	//}

	jsonOK(w, routeResps)
}

// -------------------------------------------------------------------
// /health
// -------------------------------------------------------------------

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// -------------------------------------------------------------------
// 工具函数
// -------------------------------------------------------------------

// buildRouteItems 将路由列表转换为前端条目，并按时间倒序排列
func buildRouteItems(routes []sfclient.Route) []RouteItem {
	items := make([]RouteItem, 0, len(routes))
	for _, r := range routes {
		items = append(items, RouteItem{
			OpCode:  r.OpCode,
			Time:    r.AcceptTime,
			Address: r.AcceptAddress,
			Remark:  r.Remark,
		})
	}

	// 按时间倒序（最新在前）
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time > items[j].Time
	})

	if len(items) > 0 {
		items[0].IsFirst = true
	}
	return items
}

// resolveStatus 根据最新路由的 OpCode 推断状态
func resolveStatus(routes []RouteItem) (code, label string) {
	if len(routes) == 0 {
		return "unknown", "暂无信息"
	}
	opCode := routes[0].OpCode
	switch opCode {
	case "80", "8000", "8001":
		return "delivered", "已签收"
	case "30", "3000":
		return "delivering", "派件中"
	case "50", "5000":
		return "exception", "异常件"
	case "70", "7000":
		return "returning", "退件中"
	case "10", "1000":
		return "collected", "已揽收"
	default:
		return "transit", "运输中"
	}
}

func jsonOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Code: 0, Message: "ok", Data: data})
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, response{Code: status, Message: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
