package framework

import (
	"context"
	"errors"
	"reflect"

	"gorm.io/gorm"
)

// 仓储错误定义
var (
	// ErrRecordNotFound 记录不存在
	ErrRecordNotFound = errors.New("记录不存在")

	// ErrVersionConflict 版本冲突（乐观锁）
	ErrVersionConflict = errors.New("版本冲突：记录已被其他事务修改")

	// ErrNoRowsAffected 没有行被影响
	ErrNoRowsAffected = errors.New("操作失败：没有行被影响")
)

// BaseRepository 泛型仓储实现基类
//
// 双泛型参数：
//   - T: 领域模型类型（聚合根），必须实现 Entity 接口
//   - D: 数据对象类型（DO），用于数据库持久化
//
// 职责：
//  1. 实现 Repository[T] 接口的所有方法
//  2. 提供对象转换功能（领域对象 ↔ 数据对象）
//  3. 处理软删除、乐观锁等通用逻辑
//
// 具体仓储通过嵌入此基类，自动获得所有 CRUD 实现：
//
//	type OrderRepositoryImpl struct {
//	    BaseRepository[Order, OrderDO]  // 嵌入基类
//	}
//
//	// 只需实现扩展方法
//	func (r *OrderRepositoryImpl) GetByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
//	    // 自定义查询逻辑
//	}
type BaseRepository[T Entity, D any] struct {
	db       *gorm.DB   // GORM 数据库实例
	toDO     func(T) *D // 领域对象 → 数据对象转换函数（返回指针）
	toDomain func(*D) T // 数据对象 → 领域对象转换函数（接收指针）
}

// NewBaseRepository 创建基础仓储实例
func NewBaseRepository[T Entity, D any](
	db *gorm.DB,
	toDO func(T) *D,
	toDomain func(*D) T,
) *BaseRepository[T, D] {
	return &BaseRepository[T, D]{
		db:       db,
		toDO:     toDO,
		toDomain: toDomain,
	}
}

// DB 获取数据库实例（用于扩展方法）
func (r *BaseRepository[T, D]) DB() *gorm.DB {
	return r.db
}

// Add 添加实体
func (r *BaseRepository[T, D]) Add(ctx context.Context, entity T) error {
	do := r.toDO(entity)
	result := r.db.WithContext(ctx).Create(do)
	if result.Error != nil {
		return result.Error
	}

	// 回填生成的 ID
	// 通过反射获取 DO 的 ID 字段值并设置到 entity
	id := r.extractIDFromDO(do)
	if id > 0 {
		entity.SetID(id)
	}

	return nil
}

// extractIDFromDO 从数据对象中提取 ID
// 支持多种常见的 ID 字段命名：ID, Id, id
func (r *BaseRepository[T, D]) extractIDFromDO(do *D) int64 {
	val := reflect.ValueOf(do)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return 0
	}

	// 尝试常见的 ID 字段名
	idFieldNames := []string{"ID", "Id", "id"}
	for _, name := range idFieldNames {
		field := val.FieldByName(name)
		if field.IsValid() && field.CanInt() {
			return field.Int()
		}
	}

	return 0
}

// Update 更新实体（支持乐观锁）
//
// 如果 DO 有 Version 字段，GORM 会自动实现乐观锁：
//   - 更新时 WHERE 条件会包含当前版本号
//   - 更新成功后 Version 会自动 +1
//   - 如果版本号不匹配（被其他事务修改），RowsAffected = 0
//
// 乐观锁工作原理：
//
//	UPDATE table SET field=?, version=version+1 WHERE id=? AND version=?
func (r *BaseRepository[T, D]) Update(ctx context.Context, entity T) error {
	do := r.toDO(entity)

	// 使用 Updates 方法更新（只更新非零值字段）
	// GORM 会自动处理 Version 字段的乐观锁逻辑
	result := r.db.WithContext(ctx).Updates(do)
	if result.Error != nil {
		return result.Error
	}

	// 如果没有行被影响，可能是记录不存在或版本冲突
	if result.RowsAffected == 0 {
		// 尝试判断是记录不存在还是版本冲突
		var check D
		if err := r.db.WithContext(ctx).First(&check, entity.GetID()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound
			}
			return err
		}
		// 记录存在但未更新，说明是版本冲突
		return ErrVersionConflict
	}

	return nil
}

// Delete 硬删除实体
func (r *BaseRepository[T, D]) Delete(ctx context.Context, id int64) error {
	var do D
	result := r.db.WithContext(ctx).Delete(&do, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("删除失败：记录不存在")
	}

	return nil
}

// Remove 软删除实体
// 注意：只有当 DO 有 DeletedAt 字段时，GORM 才会执行软删除
func (r *BaseRepository[T, D]) Remove(ctx context.Context, id int64) error {
	var do D
	result := r.db.WithContext(ctx).Delete(&do, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("软删除失败：记录不存在")
	}

	return nil
}

// FindByID 根据 ID 查询实体
func (r *BaseRepository[T, D]) FindByID(ctx context.Context, id int64) (T, error) {
	var do D
	result := r.db.WithContext(ctx).First(&do, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			var zero T
			return zero, ErrRecordNotFound
		}
		var zero T
		return zero, result.Error
	}

	return r.toDomain(&do), nil
}

// FindByIDWithDeleted 根据 ID 查询实体（包含已删除）
func (r *BaseRepository[T, D]) FindByIDWithDeleted(ctx context.Context, id int64) (T, error) {
	var do D
	result := r.db.WithContext(ctx).Unscoped().First(&do, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			var zero T
			return zero, ErrRecordNotFound
		}
		var zero T
		return zero, result.Error
	}

	return r.toDomain(&do), nil
}

// FindAll 查询所有实体
func (r *BaseRepository[T, D]) FindAll(ctx context.Context) ([]T, error) {
	var dos []D
	result := r.db.WithContext(ctx).Find(&dos)

	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为领域对象列表
	entities := make([]T, len(dos))
	for i := range dos {
		entities[i] = r.toDomain(&dos[i])
	}

	return entities, nil
}

// FindPage 分页查询
func (r *BaseRepository[T, D]) FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error) {
	var dos []D
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	var doModel D
	if err := r.db.WithContext(ctx).Model(&doModel).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	result := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Find(&dos)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	// 转换为领域对象列表
	entities := make([]T, len(dos))
	for i := range dos {
		entities[i] = r.toDomain(&dos[i])
	}

	return entities, total, nil
}

// Exists 检查实体是否存在
func (r *BaseRepository[T, D]) Exists(ctx context.Context, id int64) (bool, error) {
	var count int64
	var do D
	result := r.db.WithContext(ctx).Model(&do).Where("id = ?", id).Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// Transaction 执行事务
//
// 用法示例：
//
//	err := repo.Transaction(ctx, func(txRepo *BaseRepository[Order, OrderDO]) error {
//	    // 在事务中执行多个操作
//	    if err := txRepo.Add(ctx, order1); err != nil {
//	        return err  // 自动回滚
//	    }
//	    if err := txRepo.Add(ctx, order2); err != nil {
//	        return err  // 自动回滚
//	    }
//	    return nil  // 自动提交
//	})
func (r *BaseRepository[T, D]) Transaction(ctx context.Context, fn func(*BaseRepository[T, D]) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建使用事务 DB 的新仓储实例
		txRepo := &BaseRepository[T, D]{
			db:       tx,
			toDO:     r.toDO,
			toDomain: r.toDomain,
		}
		return fn(txRepo)
	})
}

// WithTx 在现有事务中创建仓储实例
//
// 用于手动管理事务的场景：
//
//	tx := db.Begin()
//	defer tx.Rollback()
//
//	txRepo := repo.WithTx(tx)
//	if err := txRepo.Add(ctx, order); err != nil {
//	    return err
//	}
//
//	tx.Commit()
func (r *BaseRepository[T, D]) WithTx(tx *gorm.DB) *BaseRepository[T, D] {
	return &BaseRepository[T, D]{
		db:       tx,
		toDO:     r.toDO,
		toDomain: r.toDomain,
	}
}

// ==================== 批量操作 ====================

// AddBatch 批量添加实体
// batchSize 为每批次插入的数量，0 或负数表示一次性插入所有
// 会自动回填生成的 ID 到每个 entity
func (r *BaseRepository[T, D]) AddBatch(ctx context.Context, entities []T, batchSize int) error {
	if len(entities) == 0 {
		return nil
	}

	// 转换为数据对象
	dos := make([]*D, len(entities))
	for i, entity := range entities {
		dos[i] = r.toDO(entity)
	}

	// 确定批次大小
	if batchSize <= 0 {
		batchSize = len(dos)
	}

	// 分批插入
	result := r.db.WithContext(ctx).CreateInBatches(dos, batchSize)
	if result.Error != nil {
		return result.Error
	}

	// 回填 ID
	for i, do := range dos {
		id := r.extractIDFromDO(do)
		if id > 0 {
			entities[i].SetID(id)
		}
	}

	return nil
}

// UpdateBatch 批量更新实体
// 注意：批量更新使用事务保证原子性，但不支持乐观锁检测
func (r *BaseRepository[T, D]) UpdateBatch(ctx context.Context, entities []T) error {
	if len(entities) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, entity := range entities {
			do := r.toDO(entity)
			if err := tx.Updates(do).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBatch 批量硬删除实体
func (r *BaseRepository[T, D]) DeleteBatch(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	var do D
	result := r.db.WithContext(ctx).Delete(&do, ids)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// RemoveBatch 批量软删除实体
// 注意：只有当 DO 有 DeletedAt 字段时，GORM 才会执行软删除
func (r *BaseRepository[T, D]) RemoveBatch(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	var do D
	result := r.db.WithContext(ctx).Delete(&do, ids)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// FindByIDs 批量根据 ID 查询实体
func (r *BaseRepository[T, D]) FindByIDs(ctx context.Context, ids []int64) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}

	var dos []D
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&dos)

	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为领域对象列表
	entities := make([]T, len(dos))
	for i := range dos {
		entities[i] = r.toDomain(&dos[i])
	}

	return entities, nil
}
