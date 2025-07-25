package util

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeInvalidParam, "测试错误")

	if err.Code != ErrCodeInvalidParam {
		t.Errorf("期望错误代码为 '%s'，实际为 '%s'", ErrCodeInvalidParam, err.Code)
	}

	if err.Message != "测试错误" {
		t.Errorf("期望错误消息为 '测试错误'，实际为 '%s'", err.Message)
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := WrapError(ErrCodeNetworkFailed, "网络请求失败", originalErr)

	if wrappedErr.Code != ErrCodeNetworkFailed {
		t.Errorf("期望错误代码为 '%s'，实际为 '%s'", ErrCodeNetworkFailed, wrappedErr.Code)
	}

	if wrappedErr.Cause != originalErr {
		t.Error("期望包装错误包含原始错误")
	}

	if wrappedErr.Unwrap() != originalErr {
		t.Error("期望Unwrap()返回原始错误")
	}
}

func TestIsErrorCode(t *testing.T) {
	appErr := NewError(ErrCodeConfigInvalid, "配置无效")
	normalErr := errors.New("普通错误")

	if !IsErrorCode(appErr, ErrCodeConfigInvalid) {
		t.Error("期望IsErrorCode返回true")
	}

	if IsErrorCode(normalErr, ErrCodeConfigInvalid) {
		t.Error("期望IsErrorCode对普通错误返回false")
	}

	if IsErrorCode(appErr, ErrCodeNetworkFailed) {
		t.Error("期望IsErrorCode对不匹配的错误代码返回false")
	}
}

func TestGetUserFriendlyMessage(t *testing.T) {
	testCases := []struct {
		err      error
		expected string
	}{
		{
			NewError(ErrCodeConfigNotFound, "配置文件未找到"),
			"配置文件未找到，请检查配置文件路径",
		},
		{
			NewError(ErrCodeAPIKeyMissing, "API密钥缺失"),
			"API密钥未配置，请在配置文件或环境变量中设置",
		},
		{
			errors.New("普通错误"),
			"发生未知错误",
		},
	}

	for _, tc := range testCases {
		result := GetUserFriendlyMessage(tc.err)
		if result != tc.expected {
			t.Errorf("期望友好消息为 '%s'，实际为 '%s'", tc.expected, result)
		}
	}
}
