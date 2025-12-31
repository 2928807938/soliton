package main

import (
	"fmt"
	"log"
	"os"
	"soliton/pkg/analyzer"
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
	fmt.Println("✨ 元数据构建完成！")
	fmt.Println("💡 下一步: 实现泛型框架开发")
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
