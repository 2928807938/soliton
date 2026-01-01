package framework

import "context"

// Repository 泛型仓储接口
//
// 泛型参数 T 约束为 Entity，确保类型安全。
// 所有具体的聚合根仓储接口都应该继承此接口。
//
// 优势：
//  1. 类型安全：返回类型根据泛型参数 T 自动推导，无需类型断言
//  2. 代码复用：基础 CRUD 只定义一次，所有仓储共用
//  3. 易于扩展：具体仓储可以添加自定义方法
//
// 示例：
//
//	// 具体仓储接口继承泛型接口
//	type OrderRepository interface {
//	    Repository[Order]  // 继承，T 被推导为 Order
//
//	    // 扩展方法
//	    GetByOrderNo(ctx context.Context, orderNo string) (*Order, error)
//	}
//
//	// 使用时类型自动推导
//	var repo OrderRepository
//	order, err := repo.FindByID(ctx, 123)  // 返回 *Order，不是 interface{}
type Repository[T Entity] interface {
	// Add 添加实体
	// 会自动回填生成的 ID 到 entity
	Add(ctx context.Context, entity T) error

	// AddBatch 批量添加实体
	// 会自动回填生成的 ID 到每个 entity
	// batchSize 为每批次插入的数量，0 表示一次性插入所有
	AddBatch(ctx context.Context, entities []T, batchSize int) error

	// Update 更新实体
	// 如果实体包含 Version 字段，会使用乐观锁
	Update(ctx context.Context, entity T) error

	// UpdateBatch 批量更新实体
	// 注意：批量更新不支持乐观锁检测
	UpdateBatch(ctx context.Context, entities []T) error

	// Delete 删除实体（硬删除）
	// 如果实体有 DeletedAt 字段，应使用 Remove 方法（软删除）
	Delete(ctx context.Context, id int64) error

	// DeleteBatch 批量硬删除实体
	DeleteBatch(ctx context.Context, ids []int64) error

	// Remove 软删除实体（仅当实体有 DeletedAt 字段时生成）
	// 设置 DeletedAt 为当前时间，不实际删除记录
	Remove(ctx context.Context, id int64) error

	// RemoveBatch 批量软删除实体
	RemoveBatch(ctx context.Context, ids []int64) error

	// FindByID 根据 ID 查询实体
	// 自动过滤已软删除的记录（如果有 DeletedAt 字段）
	FindByID(ctx context.Context, id int64) (T, error)

	// FindByIDs 批量根据 ID 查询实体
	FindByIDs(ctx context.Context, ids []int64) ([]T, error)

	// FindByIDWithDeleted 根据 ID 查询实体（包含已删除）
	// 仅当实体有 DeletedAt 字段时生成
	FindByIDWithDeleted(ctx context.Context, id int64) (T, error)

	// FindAll 查询所有实体
	// 自动过滤已软删除的记录
	FindAll(ctx context.Context) ([]T, error)

	// FindPage 分页查询
	// 返回：实体列表、总数、错误
	FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error)

	// Exists 检查实体是否存在
	Exists(ctx context.Context, id int64) (bool, error)
}
