package schema

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Schema 表示从 Go 类型反射生成的 JSON Schema 描述。
//
// 在真实业务场景（意图识别、任务规划、路由决策、工具调用参数）中，
// 我们通常希望模型返回稳定的结构化 JSON。手写 JSON Schema 冗长且容易与 Go 结构体不同步，
// 因此通过反射从 Go 结构体自动生成 Schema 是更可靠的做法。
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Description          string             `json:"description,omitempty"`
	Format               string             `json:"format,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Required             []string           `json:"required,omitempty"`
}

var timeType = reflect.TypeOf(time.Time{})

// Generate 从任意 Go 值（通常为结构体实例或指针）生成对应的 JSON Schema。
func Generate(value any) (*Schema, error) {
	if value == nil {
		return nil, fmt.Errorf("不能为 nil 生成 JSON Schema")
	}
	return generateType(reflect.TypeOf(value), make(map[reflect.Type]bool))
}

// generateType 根据反射类型生成对应的 Schema，visiting 用于检测循环类型。
func generateType(t reflect.Type, visiting map[reflect.Type]bool) (*Schema, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t == timeType {
		return &Schema{Type: "string", Format: "date-time"}, nil
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	case reflect.Slice, reflect.Array:
		items, err := generateType(t.Elem(), visiting)
		if err != nil {
			return nil, err
		}
		return &Schema{Type: "array", Items: items}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("JSON object 的 map key 必须是 string，得到 %s", t.Key())
		}
		valueSchema, err := generateType(t.Elem(), visiting)
		if err != nil {
			return nil, err
		}
		return &Schema{Type: "object", AdditionalProperties: valueSchema}, nil
	case reflect.Struct:
		if visiting[t] {
			return nil, fmt.Errorf("检测到循环类型 %s", t)
		}
		visiting[t] = true
		defer delete(visiting, t)
		return generateStruct(t, visiting)
	default:
		return nil, fmt.Errorf("不支持为 %s 生成 JSON Schema", t)
	}
}

// generateStruct 从结构体类型生成 object 类型的 Schema。
func generateStruct(t reflect.Type, visiting map[reflect.Type]bool) (*Schema, error) {
	result := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name, omitEmpty, skip := parseJSONTag(field)
		if skip {
			continue
		}

		fieldSchema, err := generateType(field.Type, visiting)
		if err != nil {
			return nil, fmt.Errorf("字段 %s: %w", field.Name, err)
		}
		fieldSchema.Description = field.Tag.Get("desc")

		if field.Anonymous && field.Tag.Get("json") == "" && fieldSchema.Type == "object" {
			for propertyName, propertySchema := range fieldSchema.Properties {
				result.Properties[propertyName] = propertySchema
			}
			result.Required = append(result.Required, fieldSchema.Required...)
			continue
		}

		result.Properties[name] = fieldSchema
		if !omitEmpty {
			result.Required = append(result.Required, name)
		}
	}
	return result, nil
}

// parseJSONTag 解析结构体字段的 json tag，返回序列化名称、是否 omitempty 以及是否忽略。
func parseJSONTag(field reflect.StructField) (name string, omitEmpty bool, skip bool) {
	name = field.Name
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return name, false, false
	}

	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		name = parts[0]
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty, false
}
