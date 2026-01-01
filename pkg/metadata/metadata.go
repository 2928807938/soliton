package metadata

import "go/ast"

// AggregateMetadata 聚合根元数据
type AggregateMetadata struct {
	Name        string                // 聚合根名称，如 "Order"
	PackageName string                // 包名
	ImportPath  string                // 完整的 import 路径，如 "mymodule/domain/model"
	ModuleName  string                // Go 模块名，如 "mymodule"
	ModuleRoot  string                // 模块根目录绝对路径
	FilePath    string                // 文件路径
	Struct      *ast.StructType       // AST 结构体类型
	Fields      []*FieldMetadata      // 字段元数据列表
	Annotations *AggregateAnnotations // 聚合根级别注解
	IDField     *FieldMetadata        // ID 字段（自动识别）
	BaseEntity  *BaseEntityMetadata   // 基础实体元数据
}

// FieldMetadata 字段元数据
type FieldMetadata struct {
	Name        string            // 字段名称，如 "OrderNo"
	Type        string            // 字段类型，如 "string", "int64"
	DBTag       string            // db 标签值，如 "order_no"
	IsPointer   bool              // 是否指针类型
	IsSlice     bool              // 是否切片类型
	Annotations *FieldAnnotations // 字段级别注解
	RawType     ast.Expr          // 原始类型表达式
}

// AggregateAnnotations 聚合根级别注解
type AggregateAnnotations struct {
	IsAggregate  bool     // +soliton:aggregate
	BaseEntity   string   // +soliton:baseEntity(BaseEntity)
	IsManyToMany bool     // +soliton:manyToMany
	Refs         []string // +soliton:ref(OtherAggregate) 可能有多个
}

// FieldAnnotations 字段级别注解
type FieldAnnotations struct {
	IsUnique      bool     // +soliton:unique
	IsRef         bool     // +soliton:ref
	IsRequired    bool     // +soliton:required
	IsEntity      bool     // +soliton:entity
	IsValueObject bool     // +soliton:valueObject
	IsIndex       bool     // +soliton:index
	EnumValues    []string // +soliton:enum(value1,value2,...)
	Strategy      string   // +soliton:valueObject(strategy=json)
}

// BaseEntityMetadata 基础实体元数据（通过字段识别）
type BaseEntityMetadata struct {
	HasDeletedAt bool // 是否有 DeletedAt 字段（软删除）
	HasVersion   bool // 是否有 Version 字段（乐观锁）
	HasCreatedAt bool // 是否有 CreatedAt 字段（创建时间）
	HasUpdatedAt bool // 是否有 UpdatedAt 字段（更新时间）
	HasCreatedBy bool // 是否有 CreatedBy 字段（创建人）
	HasUpdatedBy bool // 是否有 UpdatedBy 字段（更新人）

	DeletedAtField *FieldMetadata // DeletedAt 字段元数据
	VersionField   *FieldMetadata // Version 字段元数据
	CreatedAtField *FieldMetadata // CreatedAt 字段元数据
	UpdatedAtField *FieldMetadata // UpdatedAt 字段元数据
	CreatedByField *FieldMetadata // CreatedBy 字段元数据
	UpdatedByField *FieldMetadata // UpdatedBy 字段元数据
}

// RelationType 关系类型枚举
// 通过字段类型和注解自动判断：
//   - 一对一：单个对象类型 + +soliton:entity（如：Address *Address）
//   - 一对多：切片类型 + +soliton:entity（如：Items []*OrderItem）
//   - 多对多：双向 +soliton:ref 注解（聚合根级别）
//   - 外部引用：基础类型 + +soliton:ref（如：UserID int64）
type RelationType int

const (
	RelationTypeOneToOne   RelationType = iota // 一对一：单个对象 + entity注解
	RelationTypeOneToMany                      // 一对多：切片 + entity注解
	RelationTypeManyToMany                     // 多对多：双向ref注解
	RelationTypeRef                            // 外部引用：基础类型 + ref注解
)

// RelationMetadata 关系元数据
type RelationMetadata struct {
	SourceAggregate string         // 源聚合根
	TargetAggregate string         // 目标聚合根
	Type            RelationType   // 关系类型
	Field           *FieldMetadata // 关联字段
	IsOwner         bool           // 是否为关系的拥有方（用于多对多）
}

// ManyToManyTableMetadata 多对多关联表元数据
type ManyToManyTableMetadata struct {
	TableName      string // 关联表名，如 "user_role"
	LeftAggregate  string // 左侧聚合根，如 "User"
	RightAggregate string // 右侧聚合根，如 "Role"
	LeftColumn     string // 左侧外键列名，如 "user_id"
	RightColumn    string // 右侧外键列名，如 "role_id"
	LeftIDField    string // 左侧ID字段名
	RightIDField   string // 右侧ID字段名
	GenerationType string // 生成类型："relation_only"（纯关联）或 "aggregate"（作为聚合根）
}

// EnumMetadata 枚举元数据
type EnumMetadata struct {
	Name          string   // 枚举名称，如 "UserStatus"
	FieldName     string   // 原字段名，如 "Status"
	AggregateName string   // 所属聚合根，如 "User"
	Values        []string // 枚举值列表，如 ["ACTIVE", "INACTIVE", "BANNED"]
	GoType        string   // Go 类型，通常是 string
}

// AggregateMetadataRegistry 全局聚合根元数据注册表
type AggregateMetadataRegistry struct {
	aggregates       map[string]*AggregateMetadata // 聚合根名 -> 元数据
	relations        []*RelationMetadata           // 所有关系
	manyToManyTables []*ManyToManyTableMetadata    // 多对多关联表
	enums            []*EnumMetadata               // 所有枚举
}

// NewAggregateMetadataRegistry 创建注册表
func NewAggregateMetadataRegistry() *AggregateMetadataRegistry {
	return &AggregateMetadataRegistry{
		aggregates:       make(map[string]*AggregateMetadata),
		relations:        make([]*RelationMetadata, 0),
		manyToManyTables: make([]*ManyToManyTableMetadata, 0),
		enums:            make([]*EnumMetadata, 0),
	}
}

// Register 注册聚合根
func (r *AggregateMetadataRegistry) Register(agg *AggregateMetadata) {
	r.aggregates[agg.Name] = agg
}

// Get 获取聚合根元数据
func (r *AggregateMetadataRegistry) Get(name string) *AggregateMetadata {
	return r.aggregates[name]
}

// GetAll 获取所有聚合根
func (r *AggregateMetadataRegistry) GetAll() []*AggregateMetadata {
	result := make([]*AggregateMetadata, 0, len(r.aggregates))
	for _, agg := range r.aggregates {
		result = append(result, agg)
	}
	return result
}

// AddRelation 添加关系
func (r *AggregateMetadataRegistry) AddRelation(rel *RelationMetadata) {
	r.relations = append(r.relations, rel)
}

// GetRelations 获取所有关系
func (r *AggregateMetadataRegistry) GetRelations() []*RelationMetadata {
	return r.relations
}

// GetRelationsByAggregate 获取指定聚合根的所有关系
func (r *AggregateMetadataRegistry) GetRelationsByAggregate(aggregateName string) []*RelationMetadata {
	result := make([]*RelationMetadata, 0)
	for _, rel := range r.relations {
		if rel.SourceAggregate == aggregateName {
			result = append(result, rel)
		}
	}
	return result
}

// AddManyToManyTable 添加多对多关联表
func (r *AggregateMetadataRegistry) AddManyToManyTable(table *ManyToManyTableMetadata) {
	r.manyToManyTables = append(r.manyToManyTables, table)
}

// GetManyToManyTables 获取所有多对多关联表
func (r *AggregateMetadataRegistry) GetManyToManyTables() []*ManyToManyTableMetadata {
	return r.manyToManyTables
}

// Exists 检查聚合根是否存在
func (r *AggregateMetadataRegistry) Exists(name string) bool {
	_, ok := r.aggregates[name]
	return ok
}

// AddEnum 添加枚举
func (r *AggregateMetadataRegistry) AddEnum(enum *EnumMetadata) {
	r.enums = append(r.enums, enum)
}

// GetEnums 获取所有枚举
func (r *AggregateMetadataRegistry) GetEnums() []*EnumMetadata {
	return r.enums
}

// CollectEnums 从所有聚合根中收集枚举
func (r *AggregateMetadataRegistry) CollectEnums() {
	for _, agg := range r.aggregates {
		for _, field := range agg.Fields {
			if len(field.Annotations.EnumValues) > 0 {
				enumName := agg.Name + field.Name // 如 UserStatus
				r.enums = append(r.enums, &EnumMetadata{
					Name:          enumName,
					FieldName:     field.Name,
					AggregateName: agg.Name,
					Values:        field.Annotations.EnumValues,
					GoType:        field.Type,
				})
			}
		}
	}
}
