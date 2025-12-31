package framework

import "context"

// Service 泛型领域服务接口
//
// 领域服务负责封装基础业务规则和校验，位于领域层。
// 与应用服务的区别：
//   - 领域服务：封装基础校验（唯一性、必填、枚举等），依赖仓储接口
//   - 应用服务：用例编排、权限控制、事务管理，依赖领域服务
//
// 泛型参数 T 约束为 Entity，确保类型安全。
//
// 示例：
//
//	// 具体服务接口继承泛型接口
//	type OrderService interface {
//	    Service[Order]  // 继承，T 被推导为 Order
//
//	    // 扩展业务方法
//	    PlaceOrder(ctx context.Context, order *Order) error
//	}
type Service[T Entity] interface {
	// Add 添加实体
	// 执行基础校验：
	//  - 必填字段校验（+soliton:required）
	//  - 唯一性校验（+soliton:unique）
	//  - 外键存在性校验（+soliton:ref）
	//  - 枚举值校验（+soliton:enum）
	Add(ctx context.Context, entity T) error

	// Update 更新实体
	// 执行基础校验（排除自己的唯一性校验）
	Update(ctx context.Context, entity T) error

	// Delete 删除实体
	// 如果有 DeletedAt 字段，使用软删除
	Delete(ctx context.Context, id int64) error

	// GetByID 根据 ID 获取实体
	GetByID(ctx context.Context, id int64) (T, error)

	// GetAll 获取所有实体
	GetAll(ctx context.Context) ([]T, error)

	// GetPage 分页获取实体
	GetPage(ctx context.Context, page, pageSize int) ([]T, int64, error)

	// Exists 检查实体是否存在
	Exists(ctx context.Context, id int64) (bool, error)
}
