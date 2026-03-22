package validate

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

const (
	Unknown = "未知错误"
)

type CustomValidator struct {
	validate   *validator.Validate
	translator ut.Translator
	err        string
}

func NewCustomValidator() (*CustomValidator, error) {
	validate := validator.New()

	// 使用 json 标签名作为字段名，便于对外提示与请求参数一致
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := fld.Tag.Get("json")
		if tag == "-" || tag == "" {
			return fld.Name
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			return fld.Name
		}
		return name
	})

	// 注册中文翻译器
	chinese := zh.New()
	uni := ut.New(chinese, chinese)
	trans, isFound := uni.GetTranslator("zh")
	if !isFound {
		return nil, errors.New("validator未找到中文翻译器")
	}
	if err := zh_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, err
	}

	cv := &CustomValidator{
		validate:   validate,
		translator: trans,
	}

	// 注册自定义标签翻译：e164 -> "{0} 手机号格式不正确"
	// {0} 会被替换为字段名（json 标签名）
	_ = cv.RegisterCustomTranslation("e164", "{0} 手机号格式不正确")

	return cv, nil
}

func (cv *CustomValidator) Error() string {
	return cv.err
}

// Validate 校验结构体，如果失败返回中文错误
func (cv *CustomValidator) Validate(r *http.Request, data any) error {
	err := cv.validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		cv.err = Unknown
		return cv
	}

	// 将validator的错误翻译为中文
	var sb strings.Builder
	for _, ve := range validationErrors {

		// 先尝试使用官方中文翻译
		translated := ve.Translate(cv.translator)

		// 如果未能有效翻译，则使用自定义兜底文案（不枚举标签）
		if translated == ve.Error() || translated == "" {
			translated = fallbackGenericMessage()
		}

		// 前缀采用 tag 值，便于快速定位具体的校验规则
		sb.WriteString(translated)
		sb.WriteString("; ")
	}
	cv.err = strings.TrimSuffix(sb.String(), "; ")
	return cv
}

// fallbackGenericMessage 统一兜底文案（不做标签枚举），保持简洁
func fallbackGenericMessage() string {
	return "不满足校验要求"
}

// RegisterCustomTranslation 为指定标签注册自定义翻译模板，解耦注册方式。
// 模板占位符：
//   - {0} 字段名（使用 json 标签名）
//   - {1} 参数（如 min=3 中的 3）
//
// 示例：cv.RegisterCustomTranslation("e164", "{0} 手机号格式不正确")
func (cv *CustomValidator) RegisterCustomTranslation(tag, tmpl string) error {
	if tag == "" || tmpl == "" {
		return errors.New("tag 或 tmpl 不能为空")
	}
	return cv.validate.RegisterTranslation(tag, cv.translator,
		func(ut ut.Translator) error {
			return ut.Add(tag, tmpl, true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			msg, _ := ut.T(tag, fe.Field(), fe.Param())
			return msg
		},
	)
}

// RegisterCustomTranslations 批量注册自定义翻译。
func (cv *CustomValidator) RegisterCustomTranslations(items map[string]string) error {
	for tag, tmpl := range items {
		if err := cv.RegisterCustomTranslation(tag, tmpl); err != nil {
			return err
		}
	}
	return nil
}
