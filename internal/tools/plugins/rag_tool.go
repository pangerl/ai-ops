package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-ops/internal/common/errors"
	"ai-ops/internal/config"
	"ai-ops/internal/util"
)

// RAGTool retrieves data from a knowledge base.
type RAGTool struct{}

func (t *RAGTool) Name() string {
	return "rag_retrieval"
}

func (t *RAGTool) Description() string {
	return "从知识库检索数据"
}

func (t *RAGTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "要查询的问题",
			},
		},
		"required": []string{"query"},
	}
}

func (t *RAGTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", errors.NewError(errors.ErrCodeInvalidParam, "缺少或无效的 query 参数")
	}

	if config.Config == nil {
		return "", errors.NewError(errors.ErrCodeConfigNotFound, "系统配置未初始化")
	}

	retrievalK := config.Config.RAG.RetrievalK
	topK := config.Config.RAG.TopK
	useReranker := true // 默认开启

	util.Infow("执行RAG检索工具", map[string]any{
		"query":        query,
		"retrieval_k":  retrievalK,
		"top_k":        topK,
		"use_reranker": useReranker,
	})

	result, err := t.callRAGAPI(ctx, query, retrievalK, topK, useReranker)
	if err != nil {
		util.LogErrorWithFields(err, "RAG检索失败", map[string]any{"query": query})
		return "", err
	}

	return result, nil
}

func (t *RAGTool) callRAGAPI(ctx context.Context, query string, retrievalK, topK int, useReranker bool) (string, error) {
	if config.Config == nil {
		return "", errors.NewError(errors.ErrCodeConfigNotFound, "系统配置未初始化")
	}

	apiHost := config.Config.RAG.ApiHost
	if apiHost == "" {
		return `{"message":"RAG工具需要配置API主机才能正常工作","status":"demo"}`, nil
	}

	requestBody, err := json.Marshal(map[string]any{
		"query":        query,
		"retrieval_k":  retrievalK,
		"top_k":        topK,
		"use_reranker": useReranker,
	})
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeInternalErr, "创建请求体失败", err)
	}

	url := fmt.Sprintf("%s/api/v1/retrieve", strings.TrimRight(apiHost, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "创建RAG请求失败", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "RAG请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeNetworkFailed, "读取RAG响应失败", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewErrorWithDetails(errors.ErrCodeAPIRequestFailed, "RAG API返回错误",
			fmt.Sprintf("status_code=%d, body=%s", resp.StatusCode, string(body)))
	}

	var responseData map[string]any
	if err := json.Unmarshal(body, &responseData); err != nil {
		return "", errors.WrapError(errors.ErrCodeInternalErr, "解析RAG响应JSON失败", err)
	}

	results, ok := responseData["results"]
	if !ok {
		return "", errors.NewErrorWithDetails(errors.ErrCodeAPIRequestFailed, "RAG API响应缺少 'results' 字段", string(body))
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeInternalErr, "序列化RAG results失败", err)
	}

	return string(resultsJSON), nil
}

// NewRAGTool creates a new RAGTool instance.
func NewRAGTool() interface{} {
	return &RAGTool{}
}
