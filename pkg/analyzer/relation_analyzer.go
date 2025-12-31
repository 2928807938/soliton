package analyzer

import (
	"fmt"
	"soliton/pkg/metadata"
	"strings"
)

// RelationAnalyzer 关系分析器
type RelationAnalyzer struct {
	registry *metadata.AggregateMetadataRegistry
}

// NewRelationAnalyzer 创建关系分析器
func NewRelationAnalyzer(registry *metadata.AggregateMetadataRegistry) *RelationAnalyzer {
	return &RelationAnalyzer{
		registry: registry,
	}
}

// AnalyzeRelations 分析所有聚合根之间的关系
func (a *RelationAnalyzer) AnalyzeRelations() error {
	// 遍历所有聚合根
	for _, agg := range a.registry.GetAll() {
		// 分析字段关系
		if err := a.analyzeAggregateRelations(agg); err != nil {
			return fmt.Errorf("分析聚合根 %s 的关系失败: %w", agg.Name, err)
		}

		// 分析聚合根级别的多对多关系
		if err := a.analyzeManyToManyRelations(agg); err != nil {
			return fmt.Errorf("分析聚合根 %s 的多对多关系失败: %w", agg.Name, err)
		}
	}

	return nil
}

// analyzeAggregateRelations 分析聚合根的字段关系
func (a *RelationAnalyzer) analyzeAggregateRelations(agg *metadata.AggregateMetadata) error {
	for _, field := range agg.Fields {
		// 跳过基础类型字段
		if a.isBasicType(field.Type) {
			continue
		}

		// 识别关系类型
		relationType := a.identifyRelationType(field)

		if relationType != -1 {
			// 提取目标聚合根名称
			targetAggregate := a.extractTargetAggregate(field.Type)

			// 创建关系元数据
			relation := &metadata.RelationMetadata{
				SourceAggregate: agg.Name,
				TargetAggregate: targetAggregate,
				Type:            relationType,
				Field:           field,
			}

			a.registry.AddRelation(relation)
		}
	}

	return nil
}

// identifyRelationType 根据字段数据类型识别关系类型
//
// 判断规则（通过字段类型自动识别）：
//  1. 外部引用：字段类型为基础类型（int64等） + +soliton:ref 注解
//     示例：UserID int64 `db:"user_id" +soliton:ref`
//
//  2. 一对一：字段类型为单个对象（非切片） + +soliton:entity 注解
//     示例：Profile *UserProfile `db:"-" +soliton:entity`
//
//  3. 一对多：字段类型为切片 + +soliton:entity 注解
//     示例：Items []*OrderItem `db:"-" +soliton:entity`
//
// 4. 多对多：在聚合根级别通过双向 +soliton:ref 注解识别（由 analyzeManyToManyRelations 处理）
func (a *RelationAnalyzer) identifyRelationType(field *metadata.FieldMetadata) metadata.RelationType {
	// 规则1：外部引用 = 基础类型 + ref注解
	// 检查顺序：先检查注解，再检查类型
	if field.Annotations.IsRef && a.isBasicType(field.Type) {
		return metadata.RelationTypeRef
	}

	// 规则2和3：关联实体 = entity注解 + 根据是否切片判断一对一/一对多
	if field.Annotations.IsEntity {
		if field.IsSlice {
			// 规则3：切片类型 → 一对多
			return metadata.RelationTypeOneToMany
		} else {
			// 规则2：单个对象 → 一对一
			return metadata.RelationTypeOneToOne
		}
	}

	// 不是关系字段
	return -1
}

// analyzeManyToManyRelations 分析多对多关系（通过 +soliton:ref 注解）
// 规则：如果两个聚合根互相引用，则为多对多关系
func (a *RelationAnalyzer) analyzeManyToManyRelations(agg *metadata.AggregateMetadata) error {
	// 如果该聚合根标记为 +soliton:manyToMany，则作为聚合根处理（不生成关联表）
	if agg.Annotations.IsManyToMany {
		return nil
	}

	// 检查聚合根级别的 +soliton:ref 注解
	for _, refAggregateName := range agg.Annotations.Refs {
		// 检查目标聚合根是否存在
		targetAgg := a.registry.Get(refAggregateName)
		if targetAgg == nil {
			return fmt.Errorf("引用的聚合根 %s 不存在", refAggregateName)
		}

		// 检查是否双向引用（多对多）
		isBidirectional := false
		for _, targetRef := range targetAgg.Annotations.Refs {
			if targetRef == agg.Name {
				isBidirectional = true
				break
			}
		}

		if isBidirectional {
			// 为避免重复，只在字母序较小的一方创建关联表
			if agg.Name < refAggregateName {
				// 创建多对多关系
				relation := &metadata.RelationMetadata{
					SourceAggregate: agg.Name,
					TargetAggregate: refAggregateName,
					Type:            metadata.RelationTypeManyToMany,
					IsOwner:         true,
				}
				a.registry.AddRelation(relation)
			}
		}
	}

	return nil
}

// isBasicType 判断是否为基础类型
func (a *RelationAnalyzer) isBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"int":     true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"string":  true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
	}

	// 去除指针符号
	typeName = strings.TrimPrefix(typeName, "*")

	// 处理 time.Time
	if typeName == "time.Time" {
		return true
	}

	return basicTypes[typeName]
}

// extractTargetAggregate 提取目标聚合根名称
// 例如：*OrderItem -> OrderItem, []*OrderItem -> OrderItem
func (a *RelationAnalyzer) extractTargetAggregate(typeName string) string {
	// 去除指针和切片符号
	typeName = strings.TrimPrefix(typeName, "[]")
	typeName = strings.TrimPrefix(typeName, "*")

	// 去除包名前缀（如 model.Order -> Order）
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return typeName
}

// GenerateManyToManyTables 生成多对多关联表元数据
func (a *RelationAnalyzer) GenerateManyToManyTables() error {
	for _, relation := range a.registry.GetRelations() {
		if relation.Type == metadata.RelationTypeManyToMany {
			// 生成关联表元数据
			table := a.createManyToManyTable(relation)
			a.registry.AddManyToManyTable(table)
		}
	}
	return nil
}

// createManyToManyTable 创建多对多关联表元数据
func (a *RelationAnalyzer) createManyToManyTable(relation *metadata.RelationMetadata) *metadata.ManyToManyTableMetadata {
	leftAgg := a.registry.Get(relation.SourceAggregate)
	rightAgg := a.registry.Get(relation.TargetAggregate)

	// 生成表名：按字母序排列（如 role_user）
	var tableName string
	var leftName, rightName string
	var leftColumn, rightColumn string
	var leftIDField, rightIDField string

	if relation.SourceAggregate < relation.TargetAggregate {
		leftName = relation.SourceAggregate
		rightName = relation.TargetAggregate
	} else {
		leftName = relation.TargetAggregate
		rightName = relation.SourceAggregate
	}

	// 表名：左_右（全小写）
	tableName = toSnakeCase(leftName) + "_" + toSnakeCase(rightName)

	// 列名：聚合根名_id
	leftColumn = toSnakeCase(leftName) + "_id"
	rightColumn = toSnakeCase(rightName) + "_id"

	// ID 字段名
	if leftAgg != nil && leftAgg.IDField != nil {
		leftIDField = leftAgg.IDField.Name
	} else {
		leftIDField = "ID"
	}

	if rightAgg != nil && rightAgg.IDField != nil {
		rightIDField = rightAgg.IDField.Name
	} else {
		rightIDField = "ID"
	}

	return &metadata.ManyToManyTableMetadata{
		TableName:      tableName,
		LeftAggregate:  leftName,
		RightAggregate: rightName,
		LeftColumn:     leftColumn,
		RightColumn:    rightColumn,
		LeftIDField:    leftIDField,
		RightIDField:   rightIDField,
		GenerationType: "relation_only",
	}
}

// toSnakeCase 转换为蛇形命名
// Order -> order, OrderItem -> order_item
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// ValidateRelations 验证关系的有效性
func (a *RelationAnalyzer) ValidateRelations() []error {
	var errors []error

	// 检查所有关系的目标聚合根是否存在
	for _, relation := range a.registry.GetRelations() {
		// 外部引用不需要检查目标聚合根是否存在（可能是外部系统的）
		if relation.Type == metadata.RelationTypeRef {
			continue
		}

		// 检查目标聚合根是否已注册
		if !a.registry.Exists(relation.TargetAggregate) {
			errors = append(errors, fmt.Errorf(
				"聚合根 %s 的字段 %s 引用了不存在的聚合根 %s",
				relation.SourceAggregate,
				relation.Field.Name,
				relation.TargetAggregate,
			))
		}
	}

	return errors
}
