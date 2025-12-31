# Soliton 代码生成器 - 核心设计思路

> **版本**: v5.0
> **目标语言**: Go 1.18+
> **设计理念**: 注解驱动的 DDD 基础设施代码生成
> **更新日期**: 2025-12-31

---

## 一、设计目标

### 核心价值
通过在领域模型上添加**注释标记**，自动生成 DDD 分层架构中的基础设施代码，让开发者：
- ✅ 专注于业务逻辑实现
- ✅ 减少 70%-80% 的重复代码编写
- ✅ 保证代码质量和架构一致性
- ✅ 降低 DDD 实践门槛

### 生成与不生成的界限

**自动生成**（基础设施代码）：
- 仓储接口和实现（Repository）
- 领域服务（Domain Service）
- 数据对象（Data Object）
- 对象转换器（Convertor）
- SQL 建表脚本

**不生成**（核心业务逻辑）：
- 复杂业务用例方法
- 应用服务层
- 领域事件处理器
- 复杂查询逻辑

---

## 二、整体架构设计

### 2.1 DDD 三层架构

```
┌─────────────────────────────────┐
│   应用服务层 (Application)      │  ← 手写（用例编排、权限、事务）
├─────────────────────────────────┤
│     领域层 (Domain)             │  ← 标记 + 生成接口
│  - 聚合根（手写 + 标记）         │
│  - 仓储接口（生成）              │
│  - 领域服务接口（生成）          │
├─────────────────────────────────┤
│  基础设施层 (Infrastructure)    │  ← 生成实现
│  - 仓储实现                     │
│  - 数据对象（DO）                │
│  - 对象转换器                    │
└─────────────────────────────────┘
```

### 2.2 依赖倒置原则

**核心原则**：领域层定义接口，不依赖任何具体实现

```
应用服务层
    ↓ 依赖接口
领域层接口
    ↑ 实现接口
基础设施层
```

---

## 三、标记体系设计

### 3.1 标记分类

#### 聚合根级别标记

```go
// +soliton:aggregate                    // 声明为聚合根
// +soliton:baseEntity(BaseEntity)       // 继承基础实体（审计、软删除、乐观锁）
// +soliton:manyToMany                   // 中间实体本身是聚合根
// +soliton:ref(OtherAggregate)          // 多对多关联（纯关联表）
```

#### 字段级别标记

```go
type Order struct {
    ID      int64      `db:"id"`
    OrderNo string     `db:"order_no" +soliton:unique`      // 唯一索引
    UserID  int64      `db:"user_id" +soliton:ref`          // 外部引用
    Amount  float64    `db:"amount" +soliton:required`      // 必填字段
    Status  string     `db:"status" +soliton:enum(...)`     // 枚举校验
    Item    *OrderItem `db:"-" +soliton:entity`             // 关联实体（一对一）
    Items   []*Item    `db:"-" +soliton:entity`             // 关联实体（一对多）
    Address Address    `db:"-" +soliton:valueObject`        // 值对象
}
```

### 3.2 标记影响矩阵

| 标记 | 生成内容 | 影响 |
|------|----------|------|
| `+soliton:aggregate` | 完整代码体系 | 触发代码生成 |
| `+soliton:baseEntity` | 软删除、乐观锁、审计方法 | 智能识别字段 |
| `+soliton:entity` | 关联关系处理 | 一对一/一对多 |
| `+soliton:ref` | 外键校验、关联查询 | 外部引用 |
| `+soliton:unique` | 唯一索引、唯一性校验 | SQL + Service |
| `+soliton:required` | 非空校验 | Service 层 |
| `+soliton:enum` | 枚举值校验 | Service 层 |

---

## 四、关系处理策略

### 4.1 关系识别规则

| 字段类型 | 标记 | 识别结果 | 存储策略 |
|---------|------|---------|---------|
| 单个对象 | `+soliton:entity` | 一对一 | 关联表存外键 |
| 切片 | `+soliton:entity` | 一对多 | 关联表存外键 |
| - | 双向 `+soliton:ref` | 多对多（纯关联） | 独立关联表 |
| - | `+soliton:manyToMany` | 多对多（有业务属性） | 中间实体作为聚合根 |
| - | `+soliton:ref` | 外部引用 | 只存 ID |

### 4.2 多对多关系的两种设计

#### 方案一：领域内多对多（中间实体有业务属性）

**场景**：选课记录（有成绩）、订单明细（有数量、价格）

```go
// +soliton:manyToMany
type CourseSelection struct {
    ID       int64   `db:"id"`
    CourseID int64   `db:"course_id" +soliton:ref`
    StudentID int64  `db:"student_id" +soliton:ref`
    Score    float64 `db:"score"`      // 业务属性
    Grade    string  `db:"grade"`      // 业务属性
}
```

**特点**：中间实体作为独立聚合根，生成完整代码

#### 方案二：领域外多对多（纯关联表）

**场景**：用户-角色、文章-标签

```go
// User.go
// +soliton:aggregate
// +soliton:ref(Role)
type User struct {
    ID int64 `db:"id"`
    // ...
}

// Role.go
// +soliton:aggregate
// +soliton:ref(User)
type Role struct {
    ID int64 `db:"id"`
    // ...
}
```

**特点**：生成关联表（role_user）和关联管理 Repository

### 4.3 外部引用（避免循环依赖）

**问题**：聚合根之间不能直接持有对方对象，否则循环依赖

**解决方案**：
```go
type Order struct {
    ID     int64  `db:"id"`
    UserID int64  `db:"user_id" +soliton:ref`  // 只存 ID
    // User  *User  // ❌ 不能这样
}
```

**优点**：
- 避免循环依赖
- 保持聚合边界清晰
- 符合 DDD 聚合设计原则

---

## 五、泛型设计方案

### 5.1 设计目标

**核心价值**：
- ✅ **类型安全**：编译时类型检查，避免类型断言
- ✅ **代码复用**：基础 CRUD 只写一次，所有实体共用
- ✅ **生成代码少**：具体实现只需继承/嵌入基类
- ✅ **易于扩展**：新增实体只需继承接口

### 5.2 两层泛型架构

#### 框架层（框架提供，不生成）

**一次性开发，所有项目复用**

```go
// 实体约束接口 - 作为泛型约束
type Entity interface {
    GetID() int64
    SetID(id int64)
    IsNew() bool
}

**Entity 接口的作用**：

1. **泛型约束**：确保所有传入泛型仓储、泛型服务的类型 T 都满足基本要求
2. **统一规范**：所有实体必须实现这三个基础方法
3. **类型安全**：让泛型代码能够调用这些方法，而不需要类型断言

**为什么需要 Entity 接口？**

```go
// ❌ 没有约束，泛型代码无法调用 entity 的方法
type Repository[T any] interface {
    Add(ctx context.Context, entity T) error
}
// 编译错误：any 类型没有 GetID()、IsNew() 等方法

// ✅ 有约束，泛型代码可以调用方法
type Repository[T Entity] interface {
    Add(ctx context.Context, entity T) error
}
// T 约束为 Entity，可以调用 entity.GetID()、entity.IsNew() 等
```

**ID 字段自动识别规则**：

| 优先级 | 识别规则 | 示例 |
|-------|---------|------|
| 1 | `db:"id"` 标签 | `ID int64 \`db:"id"\`` |
| 2 | 名为 ID 的字段 | `ID int64` |
| 3 | 名为 XxxID 的字段 | `OrderID int64` |
| 4 | 第一个 int64 字段 | (兜底方案) |

**生成器实现策略**：

```go
// pkg/generator/entity_generator.go

// EntityGenerator 为每个聚合根自动生成 Entity 接口实现
// 生成方法：GetID()、SetID()、IsNew()
// 生成策略：直接追加到聚合根原文件末尾（充血模型）
type EntityGenerator struct{}

// 生成时会查找分隔标记，替换旧的生成代码
// 分隔标记：// ========== 以下代码由 soliton 自动生成，请勿手动修改 ==========
```

**生成到原文件末尾**：

```go
// 用户手写：domain/model/Order.go
// +soliton:aggregate
package model

type Order struct {
    ID       int64  `db:"id"`
    OrderNo  string `db:"order_no"`
    Amount   float64 `db:"amount"`
}

// ========== 以下代码由 soliton 自动生成，请勿手动修改 ==========
// Code generated by soliton. DO NOT EDIT.

func (o *Order) GetID() int64 {
    return o.ID
}

func (o *Order) SetID(id int64) {
    o.ID = id
}

func (o *Order) IsNew() bool {
    return o.ID == 0
}
```

**优势**：
- 聚合根和 Entity 方法在同一文件，代码内聚
- 重新生成时自动替换旧代码
- 用户只需关注聚合根定义，Entity 方法全自动

```go
// 泛型仓储接口 - 泛型参数 T 约束为 Entity
type Repository[T Entity] interface {
    Add(ctx context.Context, entity T) error
    Update(ctx context.Context, entity T) error
    Delete(ctx context.Context, id int64) error
    FindByID(ctx context.Context, id int64) (T, error)           // 返回类型自动推导为 T
    FindAll(ctx context.Context) ([]T, error)                    // 返回类型自动推导为 []T
    FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error)
}

// 泛型服务接口 - 泛型参数 T 约束为 Entity
type Service[T Entity] interface {
    Add(ctx context.Context, entity T) error
    Update(ctx context.Context, entity T) error
    Delete(ctx context.Context, id int64) error
    GetByID(ctx context.Context, id int64) (T, error)            // 返回类型自动推导为 T
    GetAll(ctx context.Context) ([]T, error)                     // 返回类型自动推导为 []T
    GetPage(ctx context.Context, page, pageSize int) ([]T, int, error)
}

// 泛型仓储实现基类 - 双泛型参数：T(领域模型), D(数据对象)
type BaseRepository[T Entity, D any] struct {
    db    *gorm.DB
    toDO  func(T) D      // 领域对象 → 数据对象
    toDom func(D) T      // 数据对象 → 领域对象
}
```

**关键特性**：返回类型根据泛型参数 `T` 自动推导，无需类型断言

#### 生成层（自动生成）

**每个聚合根自动生成**

```go
// 具体仓储接口 - 继承泛型接口，指定具体类型
type OrderRepository interface {
    Repository[Order]  // T 被推导为 Order，所有返回类型自动变为 Order 相关类型

    // 扩展方法（根据字段标记自动生成）
    // 规则：
    //   - +soliton:unique → GetByXxx(单个对象)
    //   - +soliton:index  → GetByXxx(列表)
    //   - +soliton:ref    → GetByXxxID(列表)
    GetByOrderNo(ctx context.Context, orderNo string) (*Order, error)    // OrderNo 有 +soliton:unique
    GetByUserID(ctx context.Context, userID int64) ([]*Order, error)     // UserID 有 +soliton:ref
    GetByStatus(ctx context.Context, status string) ([]*Order, error)    // Status 有 +soliton:index
}

// 具体仓储实现 - 嵌入泛型基类
type OrderRepositoryImpl struct {
    BaseRepository[Order, OrderDO]  // T=Order, D=OrderDO
}

// 具体服务接口 - 继承泛型服务接口
type OrderService interface {
    Service[Order]  // T 被推导为 Order，所有返回类型自动变为 Order 相关类型
}
```

### 5.3 完整的代码示例

#### 框架层代码（不生成）

```go
// 框架层：实体约束接口
type Entity interface {
    GetID() int64
    SetID(id int64)
    IsNew() bool
}

// 框架层：泛型仓储接口
type Repository[T Entity] interface {
    Add(ctx context.Context, entity T) error
    Update(ctx context.Context, entity T) error
    Delete(ctx context.Context, id int64) error
    FindByID(ctx context.Context, id int64) (T, error)
    FindAll(ctx context.Context) ([]T, error)
    FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error)
}

// 框架层：泛型仓储实现基类
type BaseRepository[T Entity, D any] struct {
    db    *gorm.DB
    toDO  func(T) D      // 领域对象 → 数据对象
    toDom func(D) T      // 数据对象 → 领域对象
}

// 实现通用 CRUD 方法
func (r *BaseRepository[T, D]) Add(ctx context.Context, entity T) error {
    do := r.toDO(entity)
    return r.db.WithContext(ctx).Create(&do).Error
}

func (r *BaseRepository[T, D]) FindByID(ctx context.Context, id int64) (T, error) {
    var do D
    err := r.db.WithContext(ctx).First(&do, id).Error
    if err != nil {
        var zero T
        return zero, err
    }
    return r.toDom(do), nil
}
```

#### 生成层代码（自动生成）

```go
// 生成：具体仓储接口
type OrderRepository interface {
    Repository[Order]  // 继承泛型接口，自动拥有所有 CRUD 方法

    // 扩展方法（根据字段标记自动生成）
    // 规则：+soliton:unique → 单个对象，+soliton:index/ref → 列表
    GetByOrderNo(ctx context.Context, orderNo string) (*Order, error)    // unique 字段
    GetByUserID(ctx context.Context, userID int64) ([]*Order, error)     // ref 字段
}

// 生成：具体仓储实现
type OrderRepositoryImpl struct {
    BaseRepository[Order, OrderDO]  // 嵌入泛型基类，自动拥有 CRUD 实现
}

func NewOrderRepository(db *gorm.DB) *OrderRepositoryImpl {
    return &OrderRepositoryImpl{
        BaseRepository: BaseRepository[Order, OrderDO]{
            db:    db,
            toDO:  OrderConvertor.ToData,
            toDom: OrderConvertor.ToDomain,
        },
    }
}

// 只需实现扩展方法
func (r *OrderRepositoryImpl) GetByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
    var do OrderDO
    err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&do).Error
    if err != nil {
        return nil, err
    }
    return OrderConvertor.ToDomain(do), nil
}
```

### 5.4 对象转换策略

**转换器职责**：领域对象 ↔ 数据对象 双向转换

```
┌──────────────────┐    Convertor.ToData()    ┌──────────────────┐
│  领域对象         │  ────────────────────>  │  数据对象 (DO)    │
│  (Order)         │                          │  (OrderDO)       │
│  - 业务逻辑       │                          │  - 数据库映射     │
│  - 领域规则       │                          │  - ORM 标签       │
│  - 实体关系       │                          │  - 仅存 ID        │
└──────────────────┘    Convertor.ToDomain()  └──────────────────┘
```

#### 转换规则

| 字段类型 | 转换策略 | 说明 |
|---------|---------|------|
| **简单类型** | 直接赋值 | int64、string、bool、float64、time.Time |
| **值对象** | 展开或序列化 | 内嵌：展开为多字段；JSON：序列化为字符串 |
| **关联实体** | 只转换 ID | 不递归转换对象，保持聚合边界 |
| **时间类型** | 自动处理时区 | time.Time → DATETIME |

#### 转换器生成示例

```go
// 生成：对象转换器
package convertor

import (
    "domain/model"
    "infrastructure/persistence/do"
)

// ToDomain - 数据对象转领域对象
func ToDomain(do *do.OrderDO) *model.Order {
    if do == nil {
        return nil
    }
    return &model.Order{
        ID:       do.ID,
        OrderNo:  do.OrderNo,
        UserID:   do.UserID,
        Amount:   do.Amount,
        Status:   do.Status,
        CreatedAt: do.CreatedAt,
        UpdatedAt: do.UpdatedAt,
        Version:   do.Version,
        DeletedAt: do.DeletedAt,
    }
}

// ToData - 领域对象转数据对象
func ToData(order *model.Order) *do.OrderDO {
    if order == nil {
        return nil
    }
    return &do.OrderDO{
        ID:       order.ID,
        OrderNo:  order.OrderNo,
        UserID:   order.UserID,
        Amount:   order.Amount,
        Status:   order.Status,
        CreatedAt: order.CreatedAt,
        UpdatedAt: order.UpdatedAt,
        Version:   order.Version,
        DeletedAt: order.DeletedAt,
    }
}
```

### 5.5 类型映射规则

#### 领域模型 → 数据对象 → 数据库

| 领域模型类型 | 数据对象类型 | 数据库类型 | 可空性 |
|------------|------------|-----------|-------|
| int64 | int64 | BIGINT | NOT NULL |
| int/int32 | int32 | INT | NOT NULL |
| string | string | VARCHAR(256) | NOT NULL |
| bool | bool | TINYINT(1) | NOT NULL |
| float64 | float64 | DECIMAL(20,6) | NOT NULL |
| time.Time | time.Time | DATETIME | NOT NULL |
| *int64 | *int64 | BIGINT | NULL |
| *string | *string | VARCHAR(256) | NULL |
| *time.Time | *time.Time | DATETIME | NULL |

#### 值对象处理

**内嵌展开策略**（默认）：
```go
// 领域模型
type Order struct {
    Amount Money `db:"-" +soliton:valueObject`
}

type Money struct {
    Amount   float64
    Currency string
}

// 数据对象
type OrderDO struct {
    Amount      float64  `db:"amount"`       // 展开字段
    AmountCurrency string `db:"amount_currency"` // 带前缀展开
}
```

**JSON 序列化策略**：
```go
// 领域模型
type Order struct {
    Address Address `db:"-" +soliton:valueObject(strategy=json)`
}

// 数据对象
type OrderDO struct {
    Address string `db:"address"` // JSON 字符串存储
}
```

### 5.6 泛型设计的优势

#### 优势 1：类型安全

```go
// 编译时类型检查，无需类型断言
var repo OrderRepository
order, err := repo.FindByID(ctx, 123)  // 返回 *Order，不是 interface{}
order.Pay()  // 直接调用业务方法
```

#### 优势 2：代码复用

```go
// 框架层：所有实体共用
type BaseRepository[T Entity, D any] struct { ... }

// 生成层：每个聚合根一行代码
type OrderRepositoryImpl struct {
    BaseRepository[Order, OrderDO]  // 复用所有 CRUD 逻辑
}
```

#### 优势 3：易于扩展

```go
// 新增聚合根，只需继承
type ProductRepository interface {
    Repository[Product]  // 自动拥有所有 CRUD

    // 添加扩展方法
    GetByCategory(ctx, catID) ([]*Product, error)
}
```

#### 优势 4：生成代码少

| 不使用泛型 | 使用泛型 |
|-----------|---------|
| 每个仓储实现 500+ 行 | 每个仓储实现 50 行 |
| 重复代码多 | 只写扩展方法 |
| 维护成本高 | 维护成本低 |

---

## 六、事务与一致性保证

### 6.1 聚合一致性原则

**核心原则**：一个聚合根的所有变更必须在同一个事务中完成

```go
// Repository.Add() 的事务边界
func (r *OrderRepository) Add(ctx context.Context, order *Order) error {
    return r.db.Transaction(func(tx *DB) error {
        // 1. 保存主实体（Order）
        // 2. 回填生成的主键 ID
        // 3. 保存一对一关联实体
        // 4. 批量保存一对多关联实体
        // 全部成功或全部回滚
    })
}
```

### 6.2 跨聚合一致性

**问题**：多个聚合根之间如何保证一致性？

**解决方案**：
- **应用服务层控制事务**：手动控制多聚合根操作
- **最终一致性**：领域事件 + 异步处理
- **Saga 模式**：长事务拆分为多个本地事务 + 补偿机制

---

## 七、领域事件机制

### 7.1 事件收集模式

```go
type Order struct {
    events []DomainEvent  // 内存事件列表（不持久化）
}

// 业务方法收集事件
func (o *Order) Pay() error {
    o.Status = "PAID"
    o.events = append(o.events, OrderPaidEvent{...})
    return nil
}

// 获取并清空事件
func (o *Order) PopEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}
```

### 7.2 事件发布流程

```
1. 应用服务调用聚合根业务方法
2. 聚合根收集事件到内存列表
3. Repository 保存聚合根（事务提交）
4. 应用服务获取事件列表
5. 应用服务发布事件到事件总线（事务外）
6. 事件处理器异步处理
```

**发布时机**：持久化成功后再发布（避免保存失败导致事件不一致）

---

## 八、基础实体继承设计

### 8.1 用户自定义 BaseEntity

**设计理念**：用户根据项目需求自定义基础实体，生成器智能识别字段并生成相应功能

```go
// domain/model/BaseEntity.go（用户定义）
type BaseEntity struct {
    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
    Version   int        `db:"version"`
    DeletedAt *time.Time `db:"deleted_at"`
}
```

### 8.2 智能生成规则

| BaseEntity 字段 | 生成的方法/逻辑 |
|----------------|----------------|
| `DeletedAt` | Remove（软删除）、Restore、FindByIDWithDeleted、查询自动过滤已删除 |
| `Version` | Update 方法实现乐观锁（CAS） |
| `CreatedAt/UpdatedAt` | Add/Update 自动设置时间戳 |
| `CreatedBy/UpdatedBy` | 自动从 context 获取用户 ID |

### 8.3 聚合根继承示例

```go
// +soliton:aggregate
// +soliton:baseEntity(BaseEntity)
type Order struct {
    ID       int64  `db:"id"`
    OrderNo  string `db:"order_no" +soliton:unique`

    // 审计字段（显式声明，保持代码清晰）
    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
    Version   int        `db:"version"`
    DeletedAt *time.Time `db:"deleted_at"`
}
```

**生成的额外方法**：
```go
Remove(ctx, id)           // 软删除
Restore(ctx, id)          // 恢复
FindByIDWithDeleted(ctx)  // 包含已删除
```

---

## 九、值对象处理策略

### 9.1 两种存储策略

| 策略 | 适用场景 | 优点 | 缺点 |
|------|----------|------|------|
| **内嵌展开** | 字段少、需要索引查询 | 可建索引、性能好 | 字段多时表复杂 |
| **JSON 序列化** | 字段多、不需要索引 | 表简洁、支持嵌套 | 无法建索引 |

### 9.2 标记方式

```go
// 默认：内嵌展开
Address Address `db:"-" +soliton:valueObject`

// 显式指定 JSON 序列化
Address Address `db:"-" +soliton:valueObject(strategy=json)`
```

---

## 十、代码生成工作流程

### 10.1 两阶段生成策略

```
┌─────────────────────────────────────┐
│ 阶段一：元数据收集（串行）            │
│ 1. 扫描领域模型目录                  │
│ 2. 解析 Go 源文件（go/ast）          │
│ 3. 识别聚合根和字段标记              │
│ 4. 构建全局元数据模型（关系索引）    │
│ 5. 生成多对多关联表                  │
├─────────────────────────────────────┤
│ 阶段二：代码生成（并行）              │
│ 为每个聚合根启动 goroutine：         │
│ - 仓储接口                           │
│ - 仓储实现                           │
│ - 领域服务                           │
│ - 数据对象                           │
│ - 转换器                             │
│ - SQL 脚本                           │
└─────────────────────────────────────┘
```

### 10.2 性能优势

**并行生成效果**（10个聚合根）：
- 串行生成：5250ms
- 先解析后并行：750ms
- **性能提升：85.7%**

---

## 十一、SQL 生成规则

### 11.1 类型映射

| Go 类型 | 数据库类型 | 说明 |
|---------|-----------|------|
| int64 | BIGINT | 主键、外键 |
| int/int32 | INT | 普通整型 |
| string | VARCHAR(256) | 默认长度 |
| bool | TINYINT(1) | 布尔值 |
| float64 | DECIMAL(20,6) | 金额、精度 |
| time.Time | DATETIME | 时间 |
| *T | NULL | 可空 |

### 11.2 索引生成

| 标记 | 索引类型 | 生成内容 |
|------|---------|---------|
| `+soliton:index` | INDEX | 普通索引 |
| `+soliton:unique` | UNIQUE INDEX | 唯一索引 + Service 校验 |
| `+soliton:ref` | INDEX | 外键索引（自动） |
| `DeletedAt` | INDEX | 软删除索引（必须） |

### 11.3 命名映射

```
Go 结构体名    →  数据库表名
Order         →  orders
OrderItem     →  order_item

db 标签值      →  数据库字段名
db:"user_id"  →  user_id
db:"orderNo"  →  orderNo
```

---

## 十二、领域服务自动生成

### 12.1 领域服务 vs 应用服务

| 维度 | 领域服务（生成） | 应用服务（手写） |
|------|----------------|----------------|
| **职责** | 封装基础业务规则和校验 | 用例编排、权限、事务 |
| **位置** | domain/service | application |
| **内容** | CRUD + 基础校验 | 复杂业务流程、DTO 转换 |
| **依赖** | 依赖仓储接口 | 依赖领域服务 |

### 12.2 生成目录结构

```
domain/service/
├── OrderService.go              # 服务接口（生成）
├── UserService.go               # 服务接口（生成）
└── impl/                        # 服务实现目录
    ├── OrderServiceImpl.go      # 服务实现（生成）
    └── UserServiceImpl.go       # 服务实现（生成）
```

### 12.3 服务接口生成

**生成文件**：`domain/service/{AggregateName}Service.go`

```go
// Code generated by soliton. DO NOT EDIT.

package service

import (
    "domain/model"
    "soliton/pkg/framework"
)

// OrderService Order 领域服务接口
type OrderService interface {
    framework.Service[model.Order]  // 继承泛型接口

    // 在此处添加扩展业务方法
    // 例如：PlaceOrder(ctx context.Context, order *model.Order) error
}
```

**设计特点**：
- 继承 `framework.Service[T]` 泛型接口，自动拥有所有 CRUD 方法
- 预留扩展业务方法位置，开发者可手动添加

### 12.4 服务实现生成

**生成文件**：`domain/service/impl/{AggregateName}ServiceImpl.go`

```go
// Code generated by soliton. DO NOT EDIT.

package impl

import (
    "context"
    "errors"
    "fmt"
    "domain/model"
    "domain/repository"
    "domain/service"
    "soliton/pkg/framework"
)

// OrderServiceImpl Order 领域服务实现
type OrderServiceImpl struct {
    framework.BaseService[model.Order]  // 嵌入泛型基类
    repository repository.OrderRepository
}

// NewOrderService 创建 Order 领域服务实例
func NewOrderService(repo repository.OrderRepository) *OrderServiceImpl {
    return &OrderServiceImpl{
        BaseService: *framework.NewBaseService[model.Order](repo),
        repository:  repo,
    }
}

// Add 添加实体（含校验）
func (o *OrderServiceImpl) Add(ctx context.Context, entity *model.Order) error {
    // 必填字段校验
    if err := o.validateRequired(entity); err != nil {
        return err
    }

    // 唯一性校验
    if err := o.validateUnique(ctx, entity); err != nil {
        return err
    }

    // 枚举值校验
    if err := o.validateEnum(entity); err != nil {
        return err
    }

    // 调用仓储层保存
    return o.repository.Add(ctx, entity)
}

// Update 更新实体（含校验）
func (o *OrderServiceImpl) Update(ctx context.Context, entity *model.Order) error {
    // 必填字段校验
    if err := o.validateRequired(entity); err != nil {
        return err
    }

    // 唯一性校验（排除自己）
    if err := o.validateUniqueExcludeSelf(ctx, entity); err != nil {
        return err
    }

    // 枚举值校验
    if err := o.validateEnum(entity); err != nil {
        return err
    }

    // 调用仓储层更新
    return o.repository.Update(ctx, entity)
}

// validateRequired 必填字段校验
func (o *OrderServiceImpl) validateRequired(entity *model.Order) error {
    if entity.OrderNo == "" {
        return errors.New("OrderNo 不能为空")
    }
    if entity.TotalAmount == 0 {
        return errors.New("TotalAmount 不能为空")
    }
    return nil
}

// validateUnique 唯一性校验
func (o *OrderServiceImpl) validateUnique(ctx context.Context, entity *model.Order) error {
    // OrderNo 唯一性校验
    existing, err := o.repository.GetByOrderNo(ctx, entity.OrderNo)
    if err == nil && existing != nil {
        return fmt.Errorf("OrderNo 已存在: %v", entity.OrderNo)
    }
    return nil
}

// validateUniqueExcludeSelf 唯一性校验（排除自己）
func (o *OrderServiceImpl) validateUniqueExcludeSelf(ctx context.Context, entity *model.Order) error {
    // OrderNo 唯一性校验
    existing, err := o.repository.GetByOrderNo(ctx, entity.OrderNo)
    if err == nil && existing != nil && existing.GetID() != entity.GetID() {
        return fmt.Errorf("OrderNo 已存在: %v", entity.OrderNo)
    }
    return nil
}

// validateEnum 枚举值校验
func (o *OrderServiceImpl) validateEnum(entity *model.Order) error {
    // Status 枚举校验
    validStatus := map[string]bool{
        "PENDING":    true,
        "PAID":       true,
        "SHIPPED":    true,
        "COMPLETED":  true,
        "CANCELLED":  true,
    }
    if !validStatus[entity.Status] {
        return fmt.Errorf("Status 值无效: %s", entity.Status)
    }
    return nil
}

// 确保实现了接口
var _ service.OrderService = (*OrderServiceImpl)(nil)
```

**设计特点**：
- 嵌入 `framework.BaseService[T]`，自动继承基础 CRUD 实现
- 重写 `Add` 和 `Update` 方法，加入校验逻辑
- 根据字段标记自动生成 4 种校验方法
- 包含接口实现检查，编译时保证接口正确性

### 12.5 标记驱动的校验逻辑

| 标记 | 生成校验逻辑 | 位置 | 说明 |
|------|-------------|------|------|
| `+soliton:required` | 非空校验 | `validateRequired` | string 判空，数值判 0 |
| `+soliton:unique` | 唯一性校验 | `validateUnique` | 调用 Repository 的 `GetByXxx` |
| `+soliton:unique` | 唯一性校验（排除自己） | `validateUniqueExcludeSelf` | Update 时使用 |
| `+soliton:enum` | 枚举值校验 | `validateEnum` | map 有效性检查 |

### 12.6 生成器实现

```go
// pkg/generator/service_interface_generator.go
type ServiceInterfaceGenerator struct{}

func (g *ServiceInterfaceGenerator) Generate(agg *metadata.AggregateMetadata, outputDir string) error {
    // 生成文件：domain/service/{AggregateName}Service.go
    // 继承 framework.Service[model.{AggregateName}]
}

// pkg/generator/service_impl_generator.go
type ServiceImplGenerator struct{}

func (g *ServiceImplGenerator) Generate(agg *metadata.AggregateMetadata, outputDir string) error {
    // 生成文件：domain/service/impl/{AggregateName}ServiceImpl.go
    // 生成内容：
    //   1. 结构体定义（嵌入 BaseService）
    //   2. 构造函数
    //   3. Add 方法（含校验）
    //   4. Update 方法（含校验）
    //   5. validateRequired 方法
    //   6. validateUnique 方法
    //   7. validateUniqueExcludeSelf 方法
    //   8. validateEnum 方法
    //   9. 接口实现检查
}
```

---

## 十三、设计原则总结

### 13.1 DDD 核心原则

| 原则 | 设计体现 |
|------|---------|
| **依赖倒置** | 领域层定义接口，基础设施层实现 |
| **单一职责** | 应用服务协调、领域服务规则、Repository 持久化 |
| **聚合一致性** | Repository 内事务保证原子性操作 |
| **避免循环依赖** | 使用 `+soliton:ref` 只存 ID |
| **聚合边界** | 每个聚合根有独立的 Repository |

### 13.2 代码生成原则

| 原则 | 说明 |
|------|------|
| **约定优于配置** | 默认规则满足 80% 场景 |
| **生成基础代码** | 只生成 CRUD 和基础校验 |
| **可重新生成** | 生成的代码不手动修改，通过调整标记重新生成 |
| **类型安全** | 使用泛型，编译时类型检查 |
| **模板驱动** | 通过模板引擎生成，便于维护 |

### 13.3 架构分层原则

| 原则 | 说明 |
|------|------|
| **分层清晰** | 领域层、基础设施层、应用层职责明确 |
| **面向接口** | 依赖接口而非实现 |
| **框架提供基础** | 框架提供泛型基类 |
| **生成器提供脚手架** | 生成器生成具体实现 |
| **开发者专注业务** | 只需关注业务逻辑 |

---

## 十四、核心价值与适用场景

### 14.1 核心价值

✅ **提升开发效率**
- 减少 70%-80% 的基础设施代码编写
- 专注于业务逻辑开发
- 避免重复性工作

✅ **保证代码质量**
- 统一的代码风格
- 符合 DDD 原则
- 类型安全
- 事务一致性保证

✅ **降低学习成本**
- 标记简单易懂
- 生成代码可读性高
- 符合 Go 语言惯例

### 14.2 适用场景

**适合**：
- ✅ 中大型业务系统
- ✅ 业务逻辑复杂的系统
- ✅ 需要长期维护的系统
- ✅ 团队对 DDD 有一定认知

**不适合**：
- ❌ 简单 CRUD 系统（过度设计）
- ❌ 快速原型验证（学习成本）
- ❌ 性能要求极高的场景（ORM 开销）

---

## 十五、快速示例

### 15.1 定义聚合根

```go
// domain/model/Order.go
package model

// +soliton:aggregate
// +soliton:baseEntity(BaseEntity)
type Order struct {
    ID       int64      `db:"id"`
    OrderNo  string     `db:"order_no" +soliton:unique`        // → 生成 GetByOrderNo(*Order)
    UserID   int64      `db:"user_id" +soliton:ref`            // → 生成 GetByUserID([]*Order)
    Amount   float64    `db:"amount" +soliton:required`        // → 生成非空校验
    Status   string     `db:"status" +soliton:enum("PENDING,PAID,CANCELLED")` // → 生成枚举校验
    Items    []*OrderItem `db:"-" +soliton:entity`            // → 一对多关系处理

    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
    Version   int        `db:"version"`
    DeletedAt *time.Time `db:"deleted_at"`
}
```

### 15.2 生成的内容

**自动生成**：
- `domain/repository/OrderRepository.go` - 仓储接口
- `infrastructure/persistence/OrderRepositoryImpl.go` - 仓储实现
- `domain/service/OrderService.go` - 领域服务
- `infrastructure/persistence/do/OrderDO.go` - 数据对象
- `infrastructure/persistence/convertor/OrderConvertor.go` - 转换器
- `scripts/orders.sql` - 建表 SQL

**生成的方法**：
```go
// 标准CRUD
Add(ctx, order)
Update(ctx, order)
FindByID(ctx, id)
FindAll(ctx)
FindPage(ctx, page, pageSize)

// 软删除（因为 BaseEntity 有 DeletedAt）
Remove(ctx, id)
Restore(ctx, id)
FindByIDWithDeleted(ctx, id)

// 唯一查询（因为 +soliton:unique）
GetByOrderNo(ctx, orderNo)

// 外键查询（因为 +soliton:ref）
GetByUserID(ctx, userID)
```

### 15.3 手写业务逻辑

```go
// application/service/OrderAppService.go（手写）
func (s *OrderAppService) CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    // 1. 权限校验
    if !s.authService.CanCreateOrder(ctx) {
        return errors.New("无权限")
    }

    // 2. 创建聚合根
    order := &Order{
        OrderNo: orderNoGenerator.Generate(),
        UserID:  req.UserID,
        Amount:  req.Amount,
        Status:  "PENDING",
    }

    // 3. 调用领域服务（自动生成的校验）
    if err := s.orderService.Add(ctx, order); err != nil {
        return err
    }

    // 4. 获取并发布领域事件
    events := order.PopEvents()
    for _, event := range events {
        s.eventBus.Publish(event)
    }

    return nil
}
```

---

## 十六、实施路线图

### 第一阶段：基础能力
- 聚合根识别
- 仓储接口和实现生成
- 数据对象生成
- 转换器生成
- SQL 脚本生成

### 第二阶段：关系处理
- 一对一关系处理
- 一对多关系处理
- 领域内多对多处理
- 领域外多对多处理
- 外部引用处理

### 第三阶段：高级特性
- 领域服务生成
- 值对象处理
- 索引优化
- 事务管理
- 领域事件支持

### 第四阶段：优化与扩展
- 性能优化
- 单元测试生成
- API 文档生成
- IDE 插件

---

**总结**：这是一个通过**声明式注解 + 泛型框架 + 模板引擎**实现的 DDD 代码生成器，核心思想是让开发者专注业务逻辑，基础设施代码全部自动生成。通过两阶段（串行解析 + 并行生成）的工作流程，既保证了关系识别的准确性，又最大化了生成性能。













开发计划
2. 标识体系开发
   ├─ 标记解析器(+soliton:aggregate、+soliton:ref等)
   ├─ BaseEntity 字段识别(DeletedAt、Version等)
   └─ ID 字段自动识别规则

2. 关系分析与元数据构建
   ├─ 识别聚合根及其类型
   ├─ 分析关系类型(一对一、一对多、多对多)
   ├─ 构建全局元数据模型(AggregateMetadataRegistry)
   └─ 生成多对多关联表元数据

3. 泛型框架开发
   ├─ Entity 接口定义
   ├─ Repository[T] 泛型接口
   ├─ Service[T] 泛型接口
   ├─ BaseRepository[T, D] 实现
   └─ 为聚合根生成 Entity 接口实现

4. 转换器生成
   ├─ 简单类型直接映射
   ├─ 值对象展开/JSON序列化
   └─ 关联实体只转换ID

5. 代码模板与生成策略
   ├─ 仓储接口模板(继承泛型 + 扩展方法)
   ├─ 仓储实现模板(嵌入 BaseRepository)
   ├─ 领域服务模板(含校验逻辑)
   ├─ 数据对象模板(DO)
   └─ 两阶段生成(串行解析 + 并行生成)

6. SQL 生成
   ├─ 类型映射
   ├─ 索引生成(unique、index、ref)
   └─ 关联表生成

7. 事务与一致性保证
   ├─ 聚合内事务(Repository 层)
   ├─ 乐观锁实现(Version 字段)
   └─ 软删除支持(DeletedAt 字段)

8. 测试与验证
   ├─ 单元测试生成
   └─ 生成代码正确性验证
