package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-ops/internal/config"
	"ai-ops/internal/util"
)

func TestNewWeatherTool(t *testing.T) {
	tool := NewWeatherTool()

	if tool == nil {
		t.Error("天气工具不应该为空")
		return
	}

	if tool.Name() != "weather" {
		t.Errorf("期望工具名称为'weather'，实际为: %s", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("工具描述不应该为空")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("工具参数不应该为空")
		return
	}

	// 验证参数结构
	if params["type"] != "object" {
		t.Errorf("期望参数类型为'object'，实际为: %v", params["type"])
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("参数properties应该是map类型")
		return
	}

	location, ok := properties["location"].(map[string]interface{})
	if !ok {
		t.Error("应该包含location参数")
		return
	}

	if location["type"] != "string" {
		t.Errorf("location参数类型应该为string，实际为: %v", location["type"])
	}
}

func TestWeatherTool_Execute_InvalidParams(t *testing.T) {
	tool := NewWeatherTool()
	ctx := context.Background()

	// 测试缺少location参数
	_, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("缺少location参数应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}

	// 测试空location参数
	_, err = tool.Execute(ctx, map[string]interface{}{
		"location": "",
	})
	if err == nil {
		t.Error("空location参数应该返回错误")
	}

	// 测试错误类型的location参数
	_, err = tool.Execute(ctx, map[string]interface{}{
		"location": 123,
	})
	if err == nil {
		t.Error("错误类型的location参数应该返回错误")
	}
}

func TestWeatherTool_Execute_ConfigNotFound(t *testing.T) {
	tool := NewWeatherTool()
	ctx := context.Background()

	// 保存原配置
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// 设置配置为空
	config.Config = nil

	_, err := tool.Execute(ctx, map[string]interface{}{
		"location": "北京",
	})

	if err == nil {
		t.Error("配置未初始化应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeConfigNotFound) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeConfigNotFound, util.GetErrorCode(err))
	}
}

func TestWeatherTool_Execute_ConfigInvalid(t *testing.T) {
	tool := NewWeatherTool()
	ctx := context.Background()

	// 保存原配置
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// 设置无效配置
	config.Config = &config.AppConfig{
		Weather: config.WeatherConfig{
			ApiHost: "", // 空的API主机
			ApiKey:  "", // 空的API密钥
		},
	}

	_, err := tool.Execute(ctx, map[string]interface{}{
		"location": "北京",
	})

	if err == nil {
		t.Error("无效配置应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeConfigInvalid) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeConfigInvalid, util.GetErrorCode(err))
	}
}

func TestWeatherTool_isLocationIDOrLatLon(t *testing.T) {
	tool := NewWeatherTool()

	testCases := []struct {
		location string
		expected bool
	}{
		// LocationID测试
		{"101010100", true}, // 8位数字
		{"1010101", true},   // 7位数字
		{"123456", true},    // 6位数字
		{"12345", false},    // 5位数字（不够6位）
		{"abc123", false},   // 包含字母

		// 经纬度测试
		{"116.41,39.92", true},         // 标准经纬度
		{"-116.41,39.92", true},        // 负经度
		{"116.41,-39.92", true},        // 负纬度
		{"-116.41,-39.92", true},       // 负经纬度
		{"116.123456,39.123456", true}, // 6位小数

		// 无效格式
		{"116.41", false},          // 缺少纬度
		{"116.41,", false},         // 缺少纬度值
		{",39.92", false},          // 缺少经度
		{"116.41,39.92,10", false}, // 多余的值
		{"abc,def", false},         // 非数字
		{"北京", false},              // 城市名称
		{"", false},                // 空字符串
	}

	for _, tc := range testCases {
		result := tool.isLocationIDOrLatLon(tc.location)
		if result != tc.expected {
			t.Errorf("位置'%s'的检测结果期望为%v，实际为%v", tc.location, tc.expected, result)
		}
	}
}

func TestWeatherTool_queryLocationID_Success(t *testing.T) {
	tool := NewWeatherTool()

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.URL.Path != "/geo/v2/city/lookup" {
			t.Errorf("期望请求路径为'/geo/v2/city/lookup'，实际为: %s", r.URL.Path)
		}

		if r.URL.Query().Get("location") != "北京" {
			t.Errorf("期望查询参数location为'北京'，实际为: %s", r.URL.Query().Get("location"))
		}

		if r.Header.Get("X-QW-Api-Key") != "test-api-key" {
			t.Errorf("期望API密钥为'test-api-key'，实际为: %s", r.Header.Get("X-QW-Api-Key"))
		}

		// 返回模拟响应
		response := QWeatherCityLookupResp{
			Code: "200",
			Location: []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Adm1    string `json:"adm1"`
				Adm2    string `json:"adm2"`
				Country string `json:"country"`
			}{
				{
					ID:      "101010100",
					Name:    "北京",
					Adm1:    "北京",
					Adm2:    "北京",
					Country: "中国",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}

	locationID, err := tool.queryLocationID(ctx, server.URL, "test-api-key", "北京", client)
	if err != nil {
		t.Errorf("查询LocationID不应该出错，错误: %v", err)
	}

	if locationID != "101010100" {
		t.Errorf("期望LocationID为'101010100'，实际为: %s", locationID)
	}
}

func TestWeatherTool_queryLocationID_NotFound(t *testing.T) {
	tool := NewWeatherTool()

	// 创建模拟服务器，返回未找到结果
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := QWeatherCityLookupResp{
			Code: "404",
			Location: []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Adm1    string `json:"adm1"`
				Adm2    string `json:"adm2"`
				Country string `json:"country"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}

	_, err := tool.queryLocationID(ctx, server.URL, "test-api-key", "不存在的城市", client)
	if err == nil {
		t.Error("查询不存在的城市应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeNotFound) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeNotFound, util.GetErrorCode(err))
	}
}

func TestWeatherTool_queryQWeatherNow_Success(t *testing.T) {
	tool := NewWeatherTool()

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.URL.Path != "/v7/weather/now" {
			t.Errorf("期望请求路径为'/v7/weather/now'，实际为: %s", r.URL.Path)
		}

		if r.URL.Query().Get("location") != "101010100" {
			t.Errorf("期望查询参数location为'101010100'，实际为: %s", r.URL.Query().Get("location"))
		}

		// 返回模拟天气响应
		response := QWeatherNowResp{
			Code: "200",
			Now: struct {
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
			}{
				ObsTime:   "2024-01-01T12:00+08:00",
				Temp:      "20",
				FeelsLike: "22",
				Text:      "晴",
				WindDir:   "北",
				WindScale: "3",
				Humidity:  "45",
				Precip:    "0.0",
				Vis:       "30",
				Cloud:     "10",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}

	weather, err := tool.queryQWeatherNow(ctx, server.URL, "test-api-key", "101010100", client)
	if err != nil {
		t.Errorf("查询天气不应该出错，错误: %v", err)
	}

	if weather == nil {
		t.Error("天气结果不应该为空")
		return
	}

	if weather.Code != "200" {
		t.Errorf("期望响应代码为'200'，实际为: %s", weather.Code)
	}

	if weather.Now.Temp != "20" {
		t.Errorf("期望温度为'20'，实际为: %s", weather.Now.Temp)
	}

	if weather.Now.Text != "晴" {
		t.Errorf("期望天气状况为'晴'，实际为: %s", weather.Now.Text)
	}
}

func TestWeatherTool_queryQWeatherNow_Failed(t *testing.T) {
	tool := NewWeatherTool()

	// 创建模拟服务器，返回失败响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := QWeatherNowResp{
			Code: "400",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}

	_, err := tool.queryQWeatherNow(ctx, server.URL, "test-api-key", "invalid-location", client)
	if err == nil {
		t.Error("无效位置查询应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeNotFound) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeNotFound, util.GetErrorCode(err))
	}
}
