package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"soliton/pkg/analyzer"
	"soliton/pkg/generator"
	"soliton/pkg/metadata"
	"soliton/pkg/parser"
)

func main() {
	fmt.Println("🚀 Soliton 代码生成器 v5.0")
	fmt.Println("=" + repeat("=", 50))

	// 检查参数
	if len(os.Args) < 2 {
		fmt.Println("使用方法: soliton <领域模型目录>")
		fmt.Println("示例: soliton ./domain/model")
		os.Exit(1)
	}

	modelDir := os.Args[1]

	// 创建解析器
	astParser := parser.NewASTParser()

	// 解析目录
	fmt.Printf("📂 正在解析目录: %s\n\n", modelDir)
	aggregates, err := astParser.ParseDirectory(modelDir)
	if err != nil {
		log.Fatalf("❌ 解析失败: %v", err)
	}

	fmt.Printf("✅ 成功解析 %d 个聚合根\n\n", len(aggregates))

	// 打印每个聚合根的摘要信息
	for i, agg := range aggregates {
		fmt.Printf("%d. 📦 %s\n", i+1, agg.Name)
		fmt.Printf("   包名: %s\n", agg.PackageName)

		// 打印 ID 字段
		if agg.IDField != nil {
			fmt.Printf("   🔑 ID 字段: %s (%s)\n", agg.IDField.Name, agg.IDField.Type)
		}

		// 打印 BaseEntity 特性
		baseFeatures := []string{}
		if agg.BaseEntity.HasDeletedAt {
			baseFeatures = append(baseFeatures, "软删除")
		}
		if agg.BaseEntity.HasVersion {
			baseFeatures = append(baseFeatures, "乐观锁")
		}
		if agg.BaseEntity.HasCreatedAt || agg.BaseEntity.HasUpdatedAt {
			baseFeatures = append(baseFeatures, "审计")
		}

		if len(baseFeatures) > 0 {
			fmt.Printf("   🛡️  特性: %s\n", joinStrings(baseFeatures, ", "))
		}

		// 统计字段注解
		uniqueCount := 0
		refCount := 0
		requiredCount := 0
		entityCount := 0

		for _, field := range agg.Fields {
			if field.Annotations.IsUnique {
				uniqueCount++
			}
			if field.Annotations.IsRef {
				refCount++
			}
			if field.Annotations.IsRequired {
				requiredCount++
			}
			if field.Annotations.IsEntity {
				entityCount++
			}
		}

		fmt.Printf("   📊 字段统计: %d 个字段", len(agg.Fields))
		if uniqueCount > 0 {
			fmt.Printf(", %d 个唯一索引", uniqueCount)
		}
		if refCount > 0 {
			fmt.Printf(", %d 个外键", refCount)
		}
		if requiredCount > 0 {
			fmt.Printf(", %d 个必填", requiredCount)
		}
		if entityCount > 0 {
			fmt.Printf(", %d 个关联实体", entityCount)
		}
		fmt.Println()

		// 打印关联关系
		if len(agg.Annotations.Refs) > 0 {
			fmt.Printf("   🔗 多对多关联: %v\n", agg.Annotations.Refs)
		}

		fmt.Println()
	}

	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	// ==================== 阶段二：关系分析 ====================
	fmt.Println("🔍 开始关系分析...")
	fmt.Println()

	// 构建全局元数据注册表
	registry := metadata.NewAggregateMetadataRegistry()
	for _, agg := range aggregates {
		registry.Register(agg)
	}

	// 创建关系分析器
	relationAnalyzer := analyzer.NewRelationAnalyzer(registry)

	// 分析关系
	if err := relationAnalyzer.AnalyzeRelations(); err != nil {
		log.Fatalf("❌ 关系分析失败: %v", err)
	}

	// 生成多对多关联表
	if err := relationAnalyzer.GenerateManyToManyTables(); err != nil {
		log.Fatalf("❌ 生成多对多关联表失败: %v", err)
	}

	// 验证关系
	validationErrors := relationAnalyzer.ValidateRelations()
	if len(validationErrors) > 0 {
		fmt.Printf("⚠️  发现 %d 个关系验证错误:\n", len(validationErrors))
		for _, err := range validationErrors {
			fmt.Printf("  - %v\n", err)
		}
		fmt.Println()
	}

	// 收集枚举
	registry.CollectEnums()

	fmt.Println("✅ 关系分析完成！")
	fmt.Println()

	// 打印关系统计
	relations := registry.GetRelations()
	manyToManyTables := registry.GetManyToManyTables()

	fmt.Printf("📊 关系统计:\n")
	fmt.Printf("   - 总关系数: %d\n", len(relations))

	// 统计各类关系
	oneToOneCount := 0
	oneToManyCount := 0
	manyToManyCount := 0
	refCount := 0

	for _, rel := range relations {
		switch rel.Type {
		case metadata.RelationTypeOneToOne:
			oneToOneCount++
		case metadata.RelationTypeOneToMany:
			oneToManyCount++
		case metadata.RelationTypeManyToMany:
			manyToManyCount++
		case metadata.RelationTypeRef:
			refCount++
		}
	}

	fmt.Printf("   - 一对一: %d\n", oneToOneCount)
	fmt.Printf("   - 一对多: %d\n", oneToManyCount)
	fmt.Printf("   - 多对多: %d\n", manyToManyCount)
	fmt.Printf("   - 外部引用: %d\n", refCount)
	fmt.Printf("   - 关联表: %d\n", len(manyToManyTables))
	fmt.Println()

	// 打印详细关系信息
	if len(relations) > 0 {
		fmt.Println("🔗 关系详情:")
		for i, rel := range relations {
			fmt.Printf("%d. %s → %s (%s)\n",
				i+1,
				rel.SourceAggregate,
				rel.TargetAggregate,
				relationTypeName(rel.Type))
			if rel.Field != nil {
				fmt.Printf("   字段: %s\n", rel.Field.Name)
			}
		}
		fmt.Println()
	}

	// 打印多对多关联表
	if len(manyToManyTables) > 0 {
		fmt.Println("📋 多对多关联表:")
		for i, table := range manyToManyTables {
			fmt.Printf("%d. %s (%s ↔ %s)\n",
				i+1,
				table.TableName,
				table.LeftAggregate,
				table.RightAggregate)
			fmt.Printf("   列: %s, %s\n", table.LeftColumn, table.RightColumn)
		}
		fmt.Println()
	}

	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	// 确定输出目录（项目根目录）
	outputDir := filepath.Dir(modelDir)

	// ==================== 阶段三：SQL 脚本生成 ====================
	fmt.Println("💾 开始生成 SQL 建表脚本...")
	fmt.Println()

	sqlGenerator := generator.NewSQLGenerator(registry)
	if err := sqlGenerator.Generate(outputDir); err != nil {
		log.Fatalf("❌ SQL 脚本生成失败: %v", err)
	}

	fmt.Println("✅ SQL 脚本生成完成：sql/schema.sql")
	fmt.Println()

	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	// ==================== 阶段四：代码生成 ====================
	fmt.Println("🔨 开始代码生成...")
	fmt.Println()

	// 创建生成器
	entityGenerator := generator.NewEntityGenerator()
	enumGenerator := generator.NewEnumGenerator()
	doGenerator := generator.NewDOGenerator()
	queryFieldGenerator := generator.NewQueryFieldGenerator()
	convertorGenerator := generator.NewConvertorGenerator()
	repoInterfaceGenerator := generator.NewRepositoryInterfaceGenerator()
	repoImplGenerator := generator.NewRepositoryImplGenerator()
	serviceImplGenerator := generator.NewServiceImplGenerator()

	// 生成统计
	entityCount := 0
	enumCount := 0
	doCount := 0
	queryFieldCount := 0
	convertorCount := 0
	repoInterfaceCount := 0
	repoImplCount := 0
	serviceImplCount := 0

	// 0. 生成 Entity 接口实现（追加到原领域模型文件）
	fmt.Println("📝 生成 Entity 接口实现:")
	for i, agg := range aggregates {
		fmt.Printf("%d. %s.go", i+1, toLowerFirst(agg.Name))

		if err := entityGenerator.Generate(agg); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		entityCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 1. 生成枚举类型
	enums := registry.GetEnums()
	if len(enums) > 0 {
		fmt.Println("📝 生成枚举类型:")
		if err := enumGenerator.Generate(registry, outputDir); err != nil {
			fmt.Printf("   ⚠️  失败: %v\n", err)
		} else {
			for i, enum := range enums {
				fmt.Printf("%d. %s.go ✅\n", i+1, toLowerFirst(enum.Name))
				enumCount++
			}
		}
		fmt.Println()
	}

	// 2. 生成数据对象（DO）
	fmt.Println("📝 生成数据对象（DO）:")
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sDO.go", i+1, agg.Name)

		if err := doGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		doCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 3. 生成查询字段（Query Fields）
	fmt.Println("📝 生成查询字段（类型安全查询）:")
	// 先生成通用字段类型定义
	fmt.Printf("0. field_types.go")
	if err := queryFieldGenerator.GenerateFieldTypes(outputDir); err != nil {
		fmt.Printf(" ⚠️  失败: %v\n", err)
	} else {
		fmt.Printf(" ✅\n")
	}
	// 为每个聚合根生成查询字段
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sFields.go", i+1, agg.Name)

		if err := queryFieldGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		queryFieldCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 4. 生成转换器
	fmt.Println("📝 生成转换器:")
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sConvertor.go", i+1, agg.Name)

		if err := convertorGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		convertorCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 5. 生成仓储接口
	fmt.Println("📝 生成仓储接口:")
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sRepository.go", i+1, agg.Name)

		if err := repoInterfaceGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		repoInterfaceCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 6. 生成仓储实现
	fmt.Println("📝 生成仓储实现:")
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sRepositoryImpl.go", i+1, agg.Name)

		if err := repoImplGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		repoImplCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	// 7. 生成领域服务实现
	fmt.Println("📝 生成领域服务实现:")
	for i, agg := range registry.GetAll() {
		fmt.Printf("%d. %sServiceImpl.go", i+1, agg.Name)

		if err := serviceImplGenerator.Generate(agg, outputDir); err != nil {
			fmt.Printf(" ⚠️  失败: %v\n", err)
			continue
		}

		serviceImplCount++
		fmt.Printf(" ✅\n")
	}
	fmt.Println()

	fmt.Println("=" + repeat("=", 50))
	fmt.Println("✨ 代码生成完成！")
	fmt.Println()
	fmt.Println("📊 生成统计:")
	fmt.Printf("   - SQL 建表脚本: 1 个\n")
	fmt.Printf("   - Entity 接口实现: %d 个\n", entityCount)
	fmt.Printf("   - 枚举类型: %d 个\n", enumCount)
	fmt.Printf("   - 数据对象（DO）: %d 个\n", doCount)
	fmt.Printf("   - 查询字段: %d 个\n", queryFieldCount)
	fmt.Printf("   - 转换器: %d 个\n", convertorCount)
	fmt.Printf("   - 仓储接口: %d 个\n", repoInterfaceCount)
	fmt.Printf("   - 仓储实现: %d 个\n", repoImplCount)
	fmt.Printf("   - 服务实现: %d 个\n", serviceImplCount)
	fmt.Println()
	fmt.Println("📂 生成目录:")
	fmt.Printf("   - SQL 脚本: %s\n", filepath.Join(outputDir, "sql"))
	fmt.Printf("   - Entity 接口实现: %s（已追加到原领域模型文件）\n", modelDir)
	fmt.Printf("   - 枚举类型: %s\n", filepath.Join(outputDir, "enum"))
	fmt.Printf("   - DO: %s\n", filepath.Join(filepath.Dir(outputDir), "infrastructure/do"))
	fmt.Printf("   - 查询字段: %s\n", filepath.Join(filepath.Dir(outputDir), "infrastructure/query"))
	fmt.Printf("   - 转换器: %s\n", filepath.Join(filepath.Dir(outputDir), "infrastructure/convertor"))
	fmt.Printf("   - 仓储接口: %s\n", filepath.Join(outputDir, "repository"))
	fmt.Printf("   - 仓储实现: %s\n", filepath.Join(filepath.Dir(outputDir), "infrastructure/repository"))
	fmt.Printf("   - 服务实现: %s\n", filepath.Join(outputDir, "service/impl"))
	fmt.Println()
	fmt.Println("💡 完成！所有DDD基础设施代码已生成")
}

func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]+32) + s[1:]
}

func relationTypeName(t metadata.RelationType) string {
	switch t {
	case metadata.RelationTypeOneToOne:
		return "一对一"
	case metadata.RelationTypeOneToMany:
		return "一对多"
	case metadata.RelationTypeManyToMany:
		return "多对多"
	case metadata.RelationTypeRef:
		return "外部引用"
	default:
		return "未知"
	}
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, str := range strs {
		if i > 0 {
			result += sep
		}
		result += str
	}
	return result
}
