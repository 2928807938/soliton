package framework

import "time"

// Entity 实体接口 - 作为泛型约束
//
// 所有聚合根必须实现此接口，以便能够在泛型 Repository 和 Service 中使用。
//
// 用途：
//  1. 泛型约束：确保所有传入泛型仓储、泛型服务的类型 T 都满足基本要求
//  2. 统一规范：所有实体必须实现这三个基础方法
//  3. 类型安全：让泛型代码能够调用这些方法，而不需要类型断言
//
// 示例：
//
//	// 泛型仓储可以调用 entity 的方法
//	type Repository[T Entity] interface {
//	    Add(ctx context.Context, entity T) error
//	}
//
//	func (r *BaseRepository[T, D]) Add(ctx context.Context, entity T) error {
//	    if entity.IsNew() {  // 可以安全调用 Entity 接口的方法
//	        entity.SetID(generatedID)
//	    }
//	    // ...
//	}
type Entity interface {
	// GetID 获取实体ID
	GetID() int64

	// SetID 设置实体ID
	SetID(id int64)

	// IsNew 判断是否为新实体（ID为0表示新实体）
	IsNew() bool
}

// BaseEntity 基础实体
//
// 包含所有聚合根的通用字段和方法，聚合根通过嵌入此结构体自动实现 Entity 接口。
//
// 使用方式：
//
//	type Order struct {
//	    framework.BaseEntity  // 嵌入基础实体
//	    OrderNo string
//	    Amount  float64
//	    // ... 其他业务字段
//	}
//
// 嵌入后自动获得：
//   - ID 字段和 GetID()/SetID()/IsNew() 方法
//   - CreatedAt/UpdatedAt 审计字段
//   - Version 乐观锁字段
//   - DeletedAt 软删除字段
type BaseEntity struct {
	ID        int64      `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	Version   int        `db:"version" json:"version"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// GetID 获取实体ID
func (e *BaseEntity) GetID() int64 {
	return e.ID
}

// SetID 设置实体ID
func (e *BaseEntity) SetID(id int64) {
	e.ID = id
}

// IsNew 判断是否为新实体
func (e *BaseEntity) IsNew() bool {
	return e.ID == 0
}

// IsDeleted 判断是否已软删除
func (e *BaseEntity) IsDeleted() bool {
	return e.DeletedAt != nil
}

// MarkDeleted 标记为已删除
func (e *BaseEntity) MarkDeleted() {
	now := time.Now()
	e.DeletedAt = &now
}

// Restore 恢复已删除的实体
func (e *BaseEntity) Restore() {
	e.DeletedAt = nil
}

// IncrementVersion 增加版本号（用于乐观锁）
func (e *BaseEntity) IncrementVersion() {
	e.Version++
}

// SetAuditInfo 设置审计信息
func (e *BaseEntity) SetAuditInfo(isNew bool) {
	now := time.Now()
	if isNew {
		e.CreatedAt = now
		e.UpdatedAt = now
		e.Version = 1
	} else {
		e.UpdatedAt = now
	}
}
