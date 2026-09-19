package httpx

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"streetlight/internal/apperr"
)

var validationMessages = map[string]string{
	"required": "不能为空",
	"max":      "超出最大长度或取值范围",
	"min":      "小于最小长度或取值范围",
	"len":      "长度不符合要求",
	"oneof":    "取值不在允许范围内",
	"email":    "邮箱格式不正确",
	"url":      "URL 格式不正确",
	"gte":      "应大于等于给定值",
	"lte":      "应小于等于给定值",
	"gt":       "应大于给定值",
	"lt":       "应小于给定值",
	"numeric":  "必须是数字",
}

// BindJSON 解析并校验 JSON 请求体, 失败时返回可直接响应的业务错误。
func BindJSON(c *gin.Context, target any) error {
	if err := c.ShouldBindJSON(target); err != nil {
		return apperr.BadRequest("%s", describeBindError(err))
	}
	return nil
}

// BindQuery 解析并校验 query string 参数。
func BindQuery(c *gin.Context, target any) error {
	if err := c.ShouldBindQuery(target); err != nil {
		return apperr.BadRequest("%s", describeBindError(err))
	}
	return nil
}

// ParseID 解析路径中的无符号整型 ID。
func ParseID(c *gin.Context, name string) (uint, error) {
	raw := strings.TrimSpace(c.Param(name))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, apperr.BadRequest("路径参数 %s 不是合法的 ID: %q", name, raw)
	}
	return uint(value), nil
}

// describeBindError 将校验错误转换为面向使用者的中文提示。
func describeBindError(err error) string {
	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return "请求参数解析失败"
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		parts := make([]string, 0, len(validationErrs))
		for _, fieldErr := range validationErrs {
			message, ok := validationMessages[fieldErr.Tag()]
			if !ok {
				message = "校验失败(" + fieldErr.Tag() + ")"
			}
			parts = append(parts, toSnakeCase(fieldErr.Field())+" "+message)
		}
		return "请求参数校验失败: " + strings.Join(parts, "; ")
	}

	return "请求数据格式不正确"
}

// toSnakeCase 将结构体字段名转换为前端使用的下划线命名, 便于定位出错字段。
func toSnakeCase(value string) string {
	var builder strings.Builder
	for index, r := range value {
		if unicode.IsUpper(r) {
			if index > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
