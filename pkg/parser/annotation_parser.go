package parser

import (
	"regexp"
	"strings"
)

// AnnotationParser 注解解析器
type AnnotationParser struct {
	aggregatePattern   *regexp.Regexp
	baseEntityPattern  *regexp.Regexp
	manyToManyPattern  *regexp.Regexp
	refPattern         *regexp.Regexp
	uniquePattern      *regexp.Regexp
	requiredPattern    *regexp.Regexp
	entityPattern      *regexp.Regexp
	valueObjectPattern *regexp.Regexp
	indexPattern       *regexp.Regexp
	enumPattern        *regexp.Regexp
}

// NewAnnotationParser 创建注解解析器
func NewAnnotationParser() *AnnotationParser {
	return &AnnotationParser{
		aggregatePattern:   regexp.MustCompile(`\+soliton:aggregate`),
		baseEntityPattern:  regexp.MustCompile(`\+soliton:baseEntity\((\w+)\)`),
		manyToManyPattern:  regexp.MustCompile(`\+soliton:manyToMany`),
		refPattern:         regexp.MustCompile(`\+soliton:ref(?:\((\w+)\))?`),
		uniquePattern:      regexp.MustCompile(`\+soliton:unique`),
		requiredPattern:    regexp.MustCompile(`\+soliton:required`),
		entityPattern:      regexp.MustCompile(`\+soliton:entity`),
		valueObjectPattern: regexp.MustCompile(`\+soliton:valueObject(?:\(strategy=(\w+)\))?`),
		indexPattern:       regexp.MustCompile(`\+soliton:index`),
		enumPattern:        regexp.MustCompile(`\+soliton:enum\((.*?)\)`),
	}
}

// ParseAggregateAnnotations 解析聚合根级别注解
// 输入：注释文本列表（可能包含多行注释）
// 返回：是否为聚合根、基础实体名称、是否为多对多、引用列表
func (p *AnnotationParser) ParseAggregateAnnotations(comments []string) (isAggregate bool, baseEntity string, isManyToMany bool, refs []string) {
	for _, comment := range comments {
		// 去除注释前缀 //
		text := strings.TrimSpace(strings.TrimPrefix(comment, "//"))

		// 检查是否为聚合根
		if p.aggregatePattern.MatchString(text) {
			isAggregate = true
		}

		// 检查基础实体
		if matches := p.baseEntityPattern.FindStringSubmatch(text); len(matches) > 1 {
			baseEntity = matches[1]
		}

		// 检查是否为多对多
		if p.manyToManyPattern.MatchString(text) {
			isManyToMany = true
		}

		// 检查引用
		if matches := p.refPattern.FindStringSubmatch(text); len(matches) > 0 {
			if len(matches) > 1 && matches[1] != "" {
				refs = append(refs, matches[1])
			}
		}
	}

	return
}

// ParseFieldAnnotations 解析字段级别注解
// 输入：字段标签（如 `db:"id" +soliton:unique`）
// 返回：是否唯一、是否引用、是否必填、是否实体、是否值对象、是否索引、枚举值、策略
func (p *AnnotationParser) ParseFieldAnnotations(tag string) (
	isUnique bool,
	isRef bool,
	isRequired bool,
	isEntity bool,
	isValueObject bool,
	isIndex bool,
	enumValues []string,
	strategy string,
) {
	// 检查唯一
	if p.uniquePattern.MatchString(tag) {
		isUnique = true
	}

	// 检查引用
	if p.refPattern.MatchString(tag) {
		isRef = true
	}

	// 检查必填
	if p.requiredPattern.MatchString(tag) {
		isRequired = true
	}

	// 检查实体
	if p.entityPattern.MatchString(tag) {
		isEntity = true
	}

	// 检查值对象
	if matches := p.valueObjectPattern.FindStringSubmatch(tag); len(matches) > 0 {
		isValueObject = true
		if len(matches) > 1 && matches[1] != "" {
			strategy = matches[1]
		}
	}

	// 检查索引
	if p.indexPattern.MatchString(tag) {
		isIndex = true
	}

	// 检查枚举
	if matches := p.enumPattern.FindStringSubmatch(tag); len(matches) > 1 {
		enumStr := matches[1]
		// 去除引号并分割
		enumStr = strings.Trim(enumStr, `"`)
		for _, value := range strings.Split(enumStr, ",") {
			value = strings.TrimSpace(value)
			if value != "" {
				enumValues = append(enumValues, value)
			}
		}
	}

	return
}

// ParseDBTag 解析 db 标签
// 输入：完整标签字符串，如 `db:"order_no" +soliton:unique`
// 返回：db 标签值
func (p *AnnotationParser) ParseDBTag(tag string) string {
	// 使用正则提取 db:"xxx" 中的值
	dbPattern := regexp.MustCompile(`db:"([^"]+)"`)
	matches := dbPattern.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// IsBaseEntityField 判断字段名是否为 BaseEntity 的标准字段
// 根据设计文档，BaseEntity 可能包含：DeletedAt、Version、CreatedAt、UpdatedAt、CreatedBy、UpdatedBy
func (p *AnnotationParser) IsBaseEntityField(fieldName string) (isBaseField bool, fieldType string) {
	baseFields := map[string]string{
		"DeletedAt": "deleted_at",
		"Version":   "version",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
		"CreatedBy": "created_by",
		"UpdatedBy": "updated_by",
	}

	if fieldType, ok := baseFields[fieldName]; ok {
		return true, fieldType
	}
	return false, ""
}

// IdentifyIDField 识别 ID 字段（根据优先级）
// 优先级：
// 1. db:"id" 标签
// 2. 名为 ID 的字段
// 3. 名为 XxxID 的字段（如 OrderID）
// 4. 第一个 int64 字段
func IdentifyIDField(fieldName, dbTag, fieldType string) (priority int, isIDField bool) {
	// 优先级 1: db:"id" 标签
	if dbTag == "id" {
		return 1, true
	}

	// 优先级 2: 名为 ID 的字段
	if fieldName == "ID" {
		return 2, true
	}

	// 优先级 3: 名为 XxxID 的字段
	if strings.HasSuffix(fieldName, "ID") && len(fieldName) > 2 {
		return 3, true
	}

	// 优先级 4: int64 类型字段
	if fieldType == "int64" {
		return 4, true
	}

	return 0, false
}
