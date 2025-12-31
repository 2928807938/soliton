package framework

import (
	"context"
)

// BaseService 泛型领域服务实现基类
//
// 泛型参数 T 约束为 Entity。
//
// 职责：
//  1. 实现 Service[T] 接口的所有方法
//  2. 封装基础业务规则和校验逻辑
//  3. 委托仓储层进行数据持久化
//
// 具体服务通过嵌入此基类，自动获得所有基础实现：
//
//	type OrderServiceImpl struct {
//	    BaseService[Order]
//	    // 可以添加其他依赖
//	}
//
//	// 只需实现扩展业务方法
//	func (s *OrderServiceImpl) PlaceOrder(ctx context.Context, order *Order) error {
//	    // 业务逻辑
//	}
type BaseService[T Entity] struct {
	repository Repository[T] // 仓储依赖
}

// NewBaseService 创建基础服务实例
func NewBaseService[T Entity](repository Repository[T]) *BaseService[T] {
	return &BaseService[T]{
		repository: repository,
	}
}

// Add 添加实体
// 执行基础校验后调用仓储层
// 具体的校验逻辑由生成器根据字段注解生成
func (s *BaseService[T]) Add(ctx context.Context, entity T) error {
	// 基础校验在生成的具体服务中实现
	// 这里直接调用仓储
	return s.repository.Add(ctx, entity)
}

// Update 更新实体
func (s *BaseService[T]) Update(ctx context.Context, entity T) error {
	// 基础校验在生成的具体服务中实现
	return s.repository.Update(ctx, entity)
}

// Delete 删除实体
func (s *BaseService[T]) Delete(ctx context.Context, id int64) error {
	// 检查实体是否存在
	exists, err := s.repository.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrEntityNotFound
	}

	return s.repository.Remove(ctx, id)
}

// GetByID 根据 ID 获取实体
func (s *BaseService[T]) GetByID(ctx context.Context, id int64) (T, error) {
	return s.repository.FindByID(ctx, id)
}

// GetAll 获取所有实体
func (s *BaseService[T]) GetAll(ctx context.Context) ([]T, error) {
	return s.repository.FindAll(ctx)
}

// GetPage 分页获取实体
func (s *BaseService[T]) GetPage(ctx context.Context, page, pageSize int) ([]T, int64, error) {
	return s.repository.FindPage(ctx, page, pageSize)
}

// Exists 检查实体是否存在
func (s *BaseService[T]) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repository.Exists(ctx, id)
}

// 常用错误定义
var (
	ErrEntityNotFound      = NewServiceError("实体不存在")
	ErrEntityAlreadyExists = NewServiceError("实体已存在")
	ErrValidationFailed    = NewServiceError("校验失败")
)

// ServiceError 服务层错误
type ServiceError struct {
	Message string
}

func NewServiceError(message string) *ServiceError {
	return &ServiceError{Message: message}
}

func (e *ServiceError) Error() string {
	return e.Message
}
