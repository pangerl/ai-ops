package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"ai-ops/internal/common/errors"
	"ai-ops/internal/config"
	"ai-ops/internal/util"
)

// WeatherTool 天气工具实现
type WeatherTool struct{}

func (w *WeatherTool) ID() string          { return "weather" }
func (w *WeatherTool) Name() string        { return "weather" }
func (w *WeatherTool) Type() string        { return "plugin" }
func (w *WeatherTool) Description() string { return "查询指定地点的实时天气信息" }
func (w *WeatherTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "城市名称、LocationID或经纬度坐标（格式：116.41,39.92）",
			},
		},
		"required": []string{"location"},
	}
}

func (w *WeatherTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	location, ok := args["location"].(string)
	if !ok || location == "" {
		return "", errors.NewError(errors.ErrCodeInvalidParam, "缺少或无效的 location 参数")
	}

	util.Infow("执行天气工具", map[string]any{"location": location})

	// 调用天气查询逻辑
	result, err := w.callWeatherAPI(ctx, location)
	if err != nil {
		util.LogErrorWithFields(err, "天气查询失败", map[string]any{"location": location})
		return "", err
	}

	return result, nil
}

// callWeatherAPI 调用天气API
func (w *WeatherTool) callWeatherAPI(ctx context.Context, location string) (string, error) {
	// 配置验证
	if config.Config == nil {
		return "", errors.NewError(errors.ErrCodeConfigNotFound, "系统配置未初始化")
	}

	apiHost := config.Config.Weather.ApiHost
	apiKey := config.Config.Weather.ApiKey

	if apiHost == "" || apiKey == "" {
		// 返回模拟结果而不是错误，便于演示
		return fmt.Sprintf(`{"location":"%s","message":"天气工具需要配置API密钥才能正常工作","status":"demo"}`, location), nil
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var locationID string
	var err error

	// 判断输入类型并获取LocationID
	if w.isLocationIDOrLatLon(location) {
		locationID = location
		util.Infow("使用LocationID或经纬度", map[string]any{"location_id": locationID})
	} else {
		locationID, err = w.queryLocationID(ctx, apiHost, apiKey, location, client)
		if err != nil {
			return "", err
		}
		util.Infow("城市ID查询成功", map[string]any{"city": location, "location_id": locationID})
	}

	// 查询天气
	weather, err := w.queryQWeatherNow(ctx, apiHost, apiKey, locationID, client)
	if err != nil {
		return "", err
	}

	// 直接序列化返回的天气信息
	jsonBytes, err := json.Marshal(weather)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeInternalErr, "结果序列化失败", err)
	}

	return string(jsonBytes), nil
}

// isLocationIDOrLatLon 校验 location 是否为 LocationID 或 经纬度
func (w *WeatherTool) isLocationIDOrLatLon(location string) bool {
	// LocationID: 全数字，通常为6位以上
	idRe := regexp.MustCompile(`^\d{6,}$`)
	// 经纬度: 116.41,39.92
	latlonRe := regexp.MustCompile(`^-?\d{1,3}\.\d{1,6},-?\d{1,3}\.\d{1,6}$`)
	return idRe.MatchString(location) || latlonRe.MatchString(location)
}

// queryLocationID 查询城市名称对应的 LocationID
func (w *WeatherTool) queryLocationID(ctx context.Context, apiHost, apiKey, city string, client *http.Client) (string, error) {
	urlStr := fmt.Sprintf("%s/geo/v2/city/lookup?location=%s", strings.TrimRight(apiHost, "/"), url.QueryEscape(city))

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "创建城市查询请求失败", err)
	}

	req.Header.Set("X-QW-Api-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "城市查询请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "读取城市查询响应失败", err)
	}

	var lookup QWeatherCityLookupResp
	if err := json.Unmarshal(body, &lookup); err != nil {
		return "", errors.WrapError(errors.ErrCodeAIResponseInvalid, "城市查询响应解析失败", err)
	}

	if lookup.Code != "200" || len(lookup.Location) == 0 {
		return "", errors.NewErrorWithDetails(errors.ErrCodeNotFound, "城市查询失败",
			fmt.Sprintf("code=%s, body=%s", lookup.Code, string(body)))
	}

	return lookup.Location[0].ID, nil
}

// queryQWeatherNow 查询实时天气
func (w *WeatherTool) queryQWeatherNow(ctx context.Context, apiHost, apiKey, location string, client *http.Client) (*QWeatherNow, error) {
	urlStr := fmt.Sprintf("%s/v7/weather/now?location=%s", strings.TrimRight(apiHost, "/"), url.QueryEscape(location))

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeNetworkFailed, "创建天气查询请求失败", err)
	}

	req.Header.Set("X-QW-Api-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeNetworkFailed, "天气查询请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeNetworkFailed, "读取天气查询响应失败", err)
	}

	var res QWeatherNowResp
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.WrapError(errors.ErrCodeAIResponseInvalid, "天气响应解析失败", err)
	}

	if res.Code != "200" {
		return nil, errors.NewErrorWithDetails(errors.ErrCodeNotFound, "天气查询失败",
			fmt.Sprintf("code=%s, body=%s", res.Code, string(body)))
	}

	return &res.Now, nil
}

// QWeatherNow 实时天气数据
type QWeatherNow struct {
	ObsTime   string `json:"obsTime"`
	Temp      string `json:"temp"`
	FeelsLike string `json:"feelsLike"`
	Text      string `json:"text"`
	WindDir   string `json:"windDir"`
	WindScale string `json:"windScale"`
	Humidity  string `json:"humidity"`
	Precip    string `json:"precip"`
	Vis       string `json:"vis"`
	Cloud     string `json:"cloud"`
}

// 和风天气 API 响应结构体
type QWeatherNowResp struct {
	Code  string      `json:"code"`
	Now   QWeatherNow `json:"now"`
	Refer struct {
		Sources []string `json:"sources"`
		License []string `json:"license"`
	} `json:"refer"`
}

type QWeatherCityLookupResp struct {
	Code     string `json:"code"`
	Location []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Adm1    string `json:"adm1"`
		Adm2    string `json:"adm2"`
		Country string `json:"country"`
	} `json:"location"`
}

// NewWeatherTool 创建天气工具实例
func NewWeatherTool() interface{} {
	return &WeatherTool{}
}
