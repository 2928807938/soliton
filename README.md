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
✨ 元数据构建完成！
💡 下一步: 实现泛型框架开发
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
│  ├─ generator/        # 代码生成器（待开发）
│  └─ framework/        # 泛型框架（待开发）
├─ templates/          # 代码模板（待开发）
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

## 🔜 下一步计划

按照开发计划文档，接下来需要实现：

1. **泛型框架开发**
   - Entity 接口定义
   - Repository[T] 泛型接口
   - Service[T] 泛型接口
   - BaseRepository[T, D] 实现
   - 为聚合根生成 Entity 接口实现

2. **代码生成**
   - 仓储接口和实现
   - 领域服务
   - 数据对象（DO）
   - 转换器（Convertor）
   - SQL 脚本

## 📖 设计文档

详细设计思路请参考：[Soliton代码生成器-核心设计思路.md](./Soliton代码生成器-核心设计思路.md)

## 🧑‍💻 开发者

基于 DDD 最佳实践和 Go 1.18+ 泛型特性开发

## 📄 许可证

待定
