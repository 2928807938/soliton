# Soliton 代码生成器

> 基于DDD的Go代码生成器 - 通过注解驱动自动生成基础设施代码

## ✅ 已完成功能

### 第一阶段：标识体系开发

#### 1. 标记解析器
成功实现了完整的注解解析功能，支持以下标记：

#### 聚合根级别标记
- ✅ `+soliton:aggregate` - 声明为聚合根
- ✅ `+soliton:baseEntity(BaseEntity)` - 继承基础实体
- ✅ `+soliton:manyToMany` - 中间实体本身是聚合根
- ✅ `+soliton:ref(OtherAggregate)` - 多对多关联（纯关联表）

#### 字段级别标记
- ✅ `+soliton:unique` - 唯一索引
- ✅ `+soliton:ref` - 外部引用
- ✅ `+soliton:required` - 必填字段
- ✅ `+soliton:enum(value1,value2,...)` - 枚举校验
- ✅ `+soliton:entity` - 关联实体（一对一/一对多）
- ✅ `+soliton:valueObject` - 值对象
- ✅ `+soliton:valueObject(strategy=json)` - 值对象（JSON策略）
- ✅ `+soliton:index` - 普通索引

### 2. BaseEntity 字段识别
自动识别以下基础实体字段：
- ✅ `DeletedAt` - 软删除标记
- ✅ `Version` - 乐观锁版本号
- ✅ `CreatedAt` - 创建时间
- ✅ `UpdatedAt` - 更新时间
- ✅ `CreatedBy` - 创建人
- ✅ `UpdatedBy` - 更新人

### 3. ID 字段自动识别规则
按照优先级自动识别ID字段：
1. ✅ 优先级1：`db:"id"` 标签
2. ✅ 优先级2：名为 `ID` 的字段
3. ✅ 优先级3：名为 `XxxID` 的字段（如 `OrderID`）
4. ✅ 优先级4：第一个 `int64` 字段

#### 4. 元数据模型
完整的元数据结构体系：
- ✅ `AggregateMetadata` - 聚合根元数据
- ✅ `FieldMetadata` - 字段元数据
- ✅ `AggregateAnnotations` - 聚合根注解
- ✅ `FieldAnnotations` - 字段注解
- ✅ `BaseEntityMetadata` - 基础实体元数据
- ✅ `RelationMetadata` - 关系元数据

### 第二阶段：关系分析与元数据构建

#### 1. 全局元数据注册表 (`AggregateMetadataRegistry`)
- ✅ 聚合根注册与管理
- ✅ 关系存储与查询
- ✅ 多对多关联表管理
- ✅ 聚合根存在性检查

#### 2. 关系类型分析器 (`RelationAnalyzer`)
支持自动识别以下关系类型：
- ✅ **一对一关系**：单个对象 + `+soliton:entity` 标记
- ✅ **一对多关系**：切片类型 + `+soliton:entity` 标记
- ✅ **多对多关系**：双向 `+soliton:ref` 注解
- ✅ **外部引用**：`+soliton:ref` + 基础类型（如 int64）

#### 3. 多对多关联表自动生成
- ✅ 自动检测双向引用关系
- ✅ 生成关联表元数据（表名、列名、外键）
- ✅ 智能命名（字母序排列，如 `role_user`）
- ✅ 区分纯关联表和业务聚合根（`+soliton:manyToMany`）

#### 4. 关系验证
- ✅ 目标聚合根存在性验证
- ✅ 关系一致性检查
- ✅ 错误报告机制

### 第三阶段：泛型框架开发

#### 1. Entity 接口 (`framework/entity.go`)
- ✅ 定义实体约束接口
- ✅ 用作泛型约束，确保类型安全
- ✅ 提供 GetID、SetID、IsNew 方法

#### 2. Repository[T] 泛型接口 (`framework/repository.go`)
- ✅ 定义泛型仓储接口
- ✅ 完整的 CRUD 操作
- ✅ 软删除支持（Remove、Restore）
- ✅ 分页查询支持
- ✅ 类型安全的返回值

#### 3. Service[T] 泛型接口 (`framework/service.go`)
- ✅ 定义泛型领域服务接口
- ✅ 基础业务方法
- ✅ 自动校验支持（标记驱动）

#### 4. BaseRepository[T, D] 实现 (`framework/base_repository.go`)
- ✅ 双泛型参数（领域对象 + 数据对象）
- ✅ GORM 集成
- ✅ 自动软删除处理
- ✅ 乐观锁支持
- ✅ 对象转换支持

#### 5. BaseService[T] 实现 (`framework/base_service.go`)
- ✅ 泛型服务基类
- ✅ 委托仓储层操作
- ✅ 标准错误定义

#### 6. Entity 接口实现生成器 (`generator/entity_generator.go`)
- ✅ 自动生成 Entity 接口实现
- ✅ 智能 ID 字段识别
- ✅ 类型转换处理
- ✅ 生成到聚合根同目录

### 第四阶段：转换器生成

#### 1. DO（数据对象）生成器 (`generator/do_generator.go`)
- ✅ 生成用于数据库持久化的数据对象
- ✅ 自动添加 GORM 标签
- ✅ 主键、索引、唯一约束自动配置
- ✅ 跳过关联实体字段（只存储外键ID）
- ✅ 值对象支持（展开/JSON序列化）

#### 2. 转换器生成器 (`generator/convertor_generator.go`)
- ✅ 生成 ToDomain 方法（数据对象 → 领域对象）
- ✅ 生成 ToData 方法（领域对象 → 数据对象）
- ✅ 简单类型直接映射
- ✅ 关联实体字段自动跳过
- ✅ 值对象转换注释提示

### 第五阶段：仓储与服务生成

#### 1. 仓储接口生成器 (`generator/repository_interface_generator.go`)
- ✅ 继承 Repository[T] 泛型接口
- ✅ 根据字段注解生成扩展方法
  - `unique` → GetByXxx() 返回单个对象
  - `index/ref` → GetByXxx() 返回列表

#### 2. 仓储实现生成器 (`generator/repository_impl_generator.go`)
- ✅ 嵌入 BaseRepository[T, D]
- ✅ 实现所有扩展方法
- ✅ 自动使用转换器进行对象转换
- ✅ GORM 查询实现

#### 3. 领域服务接口生成器 (`generator/service_interface_generator.go`)
- ✅ 继承 Service[T] 泛型接口
- ✅ 提供业务方法扩展接口

#### 4. 领域服务实现生成器 (`generator/service_impl_generator.go`)
- ✅ 嵌入 BaseService[T]
- ✅ 标记驱动的自动校验逻辑：
  - `required` → 非空校验
  - `unique` → 唯一性校验（Add时）
  - `unique` → 唯一性校验排除自己（Update时）
  - `enum` → 枚举值校验
- ✅ 完整的 Add/Update 方法实现

## 🚀 快速开始

### 编译

```bash
go build -o soliton.exe cmd/soliton/main.go
```

### 运行

```bash
./soliton.exe <领域模型目录>

# 示例
./soliton.exe ./domain/model
```

### 示例输出

```
🚀 Soliton 代码生成器 v5.0
===================================================
📂 正在解析目录: ./domain/model

✅ 成功解析 5 个聚合根

1. 📦 Order
   包名: model
   🔑 ID 字段: ID (int64)
   🛡️  特性: 软删除, 乐观锁, 审计
   📊 字段统计: 10 个字段, 1 个唯一索引, 1 个外键, 1 个必填, 1 个关联实体

2. 📦 User
   包名: model
   🔑 ID 字段: ID (int64)
   🛡️  特性: 软删除, 审计
   📊 字段统计: 7 个字段, 2 个唯一索引, 1 个必填
   🔗 多对多关联: [Role]

===================================================

🔍 开始关系分析...

✅ 关系分析完成！

📊 关系统计:
   - 总关系数: 4
   - 一对一: 0
   - 一对多: 1
   - 多对多: 1
   - 外部引用: 2
   - 关联表: 1

🔗 关系详情:
1. Order → User (外部引用)
   字段: UserID
2. Order → OrderItem (一对多)
   字段: Items
3. User → Role (多对多)
4. OrderItem → Product (外部引用)
   字段: ProductID

📋 多对多关联表:
1. role_user (Role ↔ User)
   列: role_id, user_id

===================================================

🔨 开始代码生成...

📝 生成 Entity 接口实现:
1. order_entity.go ✅
2. user_entity.go ✅
3. product_entity.go ✅

📝 生成数据对象（DO）:
1. OrderDO.go ✅
2. UserDO.go ✅
3. ProductDO.go ✅

📝 生成转换器:
1. OrderConvertor.go ✅
2. UserConvertor.go ✅
3. ProductConvertor.go ✅

📝 生成仓储接口:
1. OrderRepository.go ✅
2. UserRepository.go ✅
3. ProductRepository.go ✅

📝 生成仓储实现:
1. OrderRepositoryImpl.go ✅
2. UserRepositoryImpl.go ✅
3. ProductRepositoryImpl.go ✅

📝 生成领域服务接口:
1. OrderService.go ✅
2. UserService.go ✅
3. ProductService.go ✅

📝 生成领域服务实现:
1. OrderServiceImpl.go ✅
2. UserServiceImpl.go ✅
3. ProductServiceImpl.go ✅

===================================================
✨ 代码生成完成！

📊 生成统计:
   - Entity 实现: 3 个
   - 数据对象（DO）: 3 个
   - 转换器: 3 个
   - 仓储接口: 3 个
   - 仓储实现: 3 个
   - 服务接口: 3 个
   - 服务实现: 3 个

📂 生成目录:
   - Entity: ./domain/model
   - DO: ./infrastructure/persistence/do
   - 转换器: ./infrastructure/persistence/convertor
   - 仓储接口: ./domain/repository
   - 仓储实现: ./infrastructure/persistence
   - 服务接口: ./domain/service
   - 服务实现: ./domain/service

💡 完成！所有DDD基础设施代码已生成
```

## 📝 使用示例

### 定义聚合根

```go
package model

import "time"

// +soliton:aggregate
// +soliton:baseEntity(BaseEntity)
type Order struct {
    ID        int64       `db:"id"`
    OrderNo   string      `db:"order_no" +soliton:unique`
    UserID    int64       `db:"user_id" +soliton:ref`
    Amount    float64     `db:"amount" +soliton:required`
    Status    string      `db:"status" +soliton:enum("PENDING,PAID,CANCELLED")`
    Items     []*OrderItem `db:"-" +soliton:entity`
    CreatedAt time.Time   `db:"created_at"`
    UpdatedAt time.Time   `db:"updated_at"`
    Version   int         `db:"version"`
    DeletedAt *time.Time  `db:"deleted_at"`
}
```


## 📦 项目结构

```
soliton/
├─ cmd/
│  └─ soliton/          # 命令行工具入口
│     └─ main.go
├─ pkg/
│  ├─ parser/           # 标记解析器
│  │  ├─ annotation_parser.go  # 注解解析
│  │  └─ ast_parser.go         # AST 解析
│  ├─ metadata/         # 元数据模型
│  │  └─ metadata.go           # 元数据结构 + 注册表
│  ├─ analyzer/         # 关系分析器
│  │  └─ relation_analyzer.go  # 关系分析与验证
│  ├─ generator/        # 代码生成器
│  │  ├─ entity_generator.go            # Entity接口实现生成
│  │  ├─ do_generator.go                # 数据对象(DO)生成
│  │  ├─ convertor_generator.go         # 转换器生成
│  │  ├─ repository_interface_generator.go  # 仓储接口生成
│  │  ├─ repository_impl_generator.go   # 仓储实现生成
│  │  ├─ service_interface_generator.go # 服务接口生成
│  │  └─ service_impl_generator.go      # 服务实现生成
│  └─ framework/        # 泛型框架
│      ├─ entity.go            # Entity接口定义
│      ├─ repository.go        # Repository[T]接口
│      ├─ service.go           # Service[T]接口
│      ├─ base_repository.go   # BaseRepository[T,D]实现
│      └─ base_service.go      # BaseService[T]实现
├─ go.mod
└─ README.md
```

## 🎯 核心特性

### 1. 类型安全的注解解析
- 使用正则表达式精确解析注解
- 支持带参数和不带参数的注解
- 完整的错误处理

### 2. 智能字段识别
- 自动识别 ID 字段（多种策略）
- 自动识别 BaseEntity 字段
- 支持指针类型和切片类型

### 3. 完整的元数据模型
- 聚合根级别元数据
- 字段级别元数据
- 关系元数据
- 多对多关联表元数据

## 🎯 泛型框架核心优势

### 1. 类型安全
```go
// 编译时类型检查，无需类型断言
var repo OrderRepository
order, err := repo.FindByID(ctx, 123)  // 返回 *Order，不是 interface{}
order.Pay()  // 直接调用业务方法
```

### 2. 代码复用
```go
// 框架层：所有实体共用
type BaseRepository[T Entity, D any] struct { ... }

// 生成层：每个聚合根一行代码
type OrderRepositoryImpl struct {
    BaseRepository[Order, OrderDO]  // 复用所有 CRUD 逻辑
}
```

### 3. 易于扩展
```go
// 新增聚合根，只需继承
type ProductRepository interface {
    Repository[Product]  // 自动拥有所有 CRUD

    // 添加扩展方法
    GetByCategory(ctx, catID) ([]*Product, error)
}
```

## 📋 转换规则说明

### 字段转换规则

根据设计文档，转换器按以下规则处理字段：

| 字段类型 | 转换策略 | 说明 |
|---------|---------|------|
| **简单类型** | 直接赋值 | int64、string、bool、float64、time.Time |
| **值对象** | 展开或序列化 | 内嵌：展开为多字段；JSON：序列化为字符串 |
| **关联实体** | 只转换 ID | 不递归转换对象，保持聚合边界 |
| **时间类型** | 自动处理 | time.Time → DATETIME |

### 生成的代码示例

#### 数据对象（DO）
```go
// Code generated by soliton. DO NOT EDIT.
package do

import "time"

// OrderDO Order 数据对象
type OrderDO struct {
	ID        int64      `gorm:"column:id;primaryKey;autoIncrement"`
	OrderNo   string     `gorm:"column:order_no;uniqueIndex:idx_order_no"`
	UserID    int64      `gorm:"column:user_id;index:idx_user_id"`
	Amount    float64    `gorm:"column:amount;not null"`
	Status    string     `gorm:"column:status"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	Version   int        `gorm:"column:version"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index:idx_deleted_at"`
}

// TableName 指定表名
func (OrderDO) TableName() string {
	return "orders"
}
```

#### 转换器
```go
// Code generated by soliton. DO NOT EDIT.
package convertor

import (
	"domain/model"
	"infrastructure/persistence/do"
)

// ToDomain 数据对象转领域对象
func ToDomain(dataObj *do.OrderDO) *model.Order {
	if dataObj == nil {
		return nil
	}

	return &model.Order{
		ID:        dataObj.ID,
		OrderNo:   dataObj.OrderNo,
		UserID:    dataObj.UserID,
		Amount:    dataObj.Amount,
		Status:    dataObj.Status,
		// Items: 关联实体，不转换
		CreatedAt: dataObj.CreatedAt,
		UpdatedAt: dataObj.UpdatedAt,
		Version:   dataObj.Version,
		DeletedAt: dataObj.DeletedAt,
	}
}

// ToData 领域对象转数据对象
func ToData(domain *model.Order) *do.OrderDO {
	if domain == nil {
		return nil
	}

	return &do.OrderDO{
		ID:        domain.ID,
		OrderNo:   domain.OrderNo,
		UserID:    domain.UserID,
		Amount:    domain.Amount,
		Status:    domain.Status,
		// Items: 关联实体，不转换
		CreatedAt: domain.CreatedAt,
		UpdatedAt: domain.UpdatedAt,
		Version:   domain.Version,
		DeletedAt: domain.DeletedAt,
	}
}
```

## 📊 项目统计

- **总文件数**: 17个Go文件
- **总代码行数**: 3013行
- **完整的DDD代码生成系统**

## 🔜 下一步计划

按照开发计划文档，接下来可以实现：

1. **SQL 脚本生成器**
   - 根据 DO 生成建表 SQL
   - 索引、约束自动生成

2. **优化与扩展**
   - 单元测试生成
   - API 文档生成

## 📖 设计文档

详细设计思路请参考：[Soliton代码生成器-核心设计思路.md](./Soliton代码生成器-核心设计思路.md)

## 🧑‍💻 开发者

基于 DDD 最佳实践和 Go 1.18+ 泛型特性开发

## 📄 许可证

待定
