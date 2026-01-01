package parser

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"soliton/pkg/metadata"
	"strings"
)

// ASTParser AST 解析器
type ASTParser struct {
	annotationParser *AnnotationParser
	fset             *token.FileSet
}

// NewASTParser 创建 AST 解析器
func NewASTParser() *ASTParser {
	return &ASTParser{
		annotationParser: NewAnnotationParser(),
		fset:             token.NewFileSet(),
	}
}

// ParseFile 解析单个 Go 文件
// 返回：聚合根元数据列表
func (p *ASTParser) ParseFile(filePath string) ([]*metadata.AggregateMetadata, error) {
	// 解析文件
	file, err := parser.ParseFile(p.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	var aggregates []*metadata.AggregateMetadata

	// 遍历文件中的所有声明
	for _, decl := range file.Decls {
		// 只处理类型声明
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// 遍历类型声明中的所有类型
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// 只处理结构体类型
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// 提取注释
			comments := p.extractComments(genDecl.Doc)

			// 解析聚合根级别注解
			isAggregate, baseEntity, isManyToMany, refs := p.annotationParser.ParseAggregateAnnotations(comments)

			// 如果不是聚合根，跳过
			if !isAggregate {
				continue
			}

			// 创建聚合根元数据
			aggregate := &metadata.AggregateMetadata{
				Name:        typeSpec.Name.Name,
				PackageName: file.Name.Name,
				FilePath:    filePath,
				Struct:      structType,
				Annotations: &metadata.AggregateAnnotations{
					IsAggregate:  true,
					BaseEntity:   baseEntity,
					IsManyToMany: isManyToMany,
					Refs:         refs,
				},
			}

			// 解析字段
			aggregate.Fields = p.parseFields(structType)

			// 识别 BaseEntity 字段
			aggregate.BaseEntity = p.identifyBaseEntityFields(aggregate.Fields)

			// 识别 ID 字段
			aggregate.IDField = p.identifyIDField(aggregate.Fields)

			aggregates = append(aggregates, aggregate)
		}
	}

	return aggregates, nil
}

// ParseDirectory 解析目录（递归）
func (p *ASTParser) ParseDirectory(dirPath string) ([]*metadata.AggregateMetadata, error) {
	var allAggregates []*metadata.AggregateMetadata

	// 获取绝对路径
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 查找模块信息
	modRoot, modName := p.findGoMod(absDir)

	// 计算 import 路径
	importPath := p.calculateImportPath(absDir)

	// 解析目录中的所有包
	pkgs, err := parser.ParseDir(p.fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析目录失败: %w", err)
	}

	// 遍历每个包
	for _, pkg := range pkgs {
		// 遍历包中的每个文件
		// 注意：pkg.Files 的键已经是完整的文件路径，无需再拼接 dirPath
		for filePath, file := range pkg.Files {

			// 遍历文件中的所有声明
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					// 提取注释并解析
					comments := p.extractComments(genDecl.Doc)
					isAggregate, baseEntity, isManyToMany, refs := p.annotationParser.ParseAggregateAnnotations(comments)

					if !isAggregate {
						continue
					}

					aggregate := &metadata.AggregateMetadata{
						Name:        typeSpec.Name.Name,
						PackageName: file.Name.Name,
						ImportPath:  importPath,
						ModuleName:  modName,
						ModuleRoot:  modRoot,
						FilePath:    filePath,
						Struct:      structType,
						Annotations: &metadata.AggregateAnnotations{
							IsAggregate:  true,
							BaseEntity:   baseEntity,
							IsManyToMany: isManyToMany,
							Refs:         refs,
						},
					}

					aggregate.Fields = p.parseFields(structType)
					aggregate.BaseEntity = p.identifyBaseEntityFields(aggregate.Fields)
					aggregate.IDField = p.identifyIDField(aggregate.Fields)

					allAggregates = append(allAggregates, aggregate)
				}
			}
		}
	}

	return allAggregates, nil
}

// calculateImportPath 计算目录的完整 import 路径
func (p *ASTParser) calculateImportPath(absDir string) string {
	// 向上查找 go.mod 文件
	modRoot, modName := p.findGoMod(absDir)
	if modRoot == "" || modName == "" {
		// 找不到 go.mod，返回目录名作为包名
		return filepath.Base(absDir)
	}

	// 计算相对路径
	relPath, err := filepath.Rel(modRoot, absDir)
	if err != nil {
		return filepath.Base(absDir)
	}

	// 将路径分隔符统一为 /
	relPath = filepath.ToSlash(relPath)

	// 如果相对路径是 "."，则直接返回模块名
	if relPath == "." {
		return modName
	}

	// 拼接完整的 import 路径
	return modName + "/" + relPath
}

// findGoMod 向上查找 go.mod 文件，返回模块根目录和模块名
func (p *ASTParser) findGoMod(startDir string) (modRoot string, modName string) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// 找到 go.mod，读取模块名
			modName = p.readModuleName(goModPath)
			return dir, modName
		}

		// 向上一级目录
		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到达根目录
			return "", ""
		}
		dir = parent
	}
}

// readModuleName 从 go.mod 文件中读取模块名
func (p *ASTParser) readModuleName(goModPath string) string {
	file, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// parseFields 解析结构体字段
func (p *ASTParser) parseFields(structType *ast.StructType) []*metadata.FieldMetadata {
	var fields []*metadata.FieldMetadata

	for _, field := range structType.Fields.List {
		// 跳过匿名字段
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name

		// 提取标签
		var tag string
		if field.Tag != nil {
			tag = field.Tag.Value
			// 去除反引号
			tag = strings.Trim(tag, "`")
		}

		// 解析 db 标签
		dbTag := p.annotationParser.ParseDBTag(tag)

		// 解析字段注解
		isUnique, isRef, isRequired, isEntity, isValueObject, isIndex, enumValues, strategy :=
			p.annotationParser.ParseFieldAnnotations(tag)

		// 分析字段类型
		fieldType, isPointer, isSlice := p.analyzeFieldType(field.Type)

		fieldMeta := &metadata.FieldMetadata{
			Name:      fieldName,
			Type:      fieldType,
			DBTag:     dbTag,
			IsPointer: isPointer,
			IsSlice:   isSlice,
			RawType:   field.Type,
			Annotations: &metadata.FieldAnnotations{
				IsUnique:      isUnique,
				IsRef:         isRef,
				IsRequired:    isRequired,
				IsEntity:      isEntity,
				IsValueObject: isValueObject,
				IsIndex:       isIndex,
				EnumValues:    enumValues,
				Strategy:      strategy,
			},
		}

		fields = append(fields, fieldMeta)
	}

	return fields
}

// analyzeFieldType 分析字段类型
// 返回：类型名称、是否指针、是否切片
func (p *ASTParser) analyzeFieldType(expr ast.Expr) (typeName string, isPointer bool, isSlice bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		// 简单类型，如 int64, string
		return t.Name, false, false
	case *ast.StarExpr:
		// 指针类型，如 *time.Time
		innerType, _, _ := p.analyzeFieldType(t.X)
		return innerType, true, false
	case *ast.ArrayType:
		// 切片类型，如 []*OrderItem
		if t.Len == nil { // 切片
			innerType, isPtr, _ := p.analyzeFieldType(t.Elt)
			return innerType, isPtr, true
		}
	case *ast.SelectorExpr:
		// 限定类型，如 time.Time
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name, false, false
		}
	}
	return "unknown", false, false
}

// extractComments 提取注释文本
func (p *ASTParser) extractComments(commentGroup *ast.CommentGroup) []string {
	if commentGroup == nil {
		return nil
	}

	var comments []string
	for _, comment := range commentGroup.List {
		comments = append(comments, comment.Text)
	}
	return comments
}

// identifyBaseEntityFields 识别 BaseEntity 字段
func (p *ASTParser) identifyBaseEntityFields(fields []*metadata.FieldMetadata) *metadata.BaseEntityMetadata {
	baseEntity := &metadata.BaseEntityMetadata{}

	for _, field := range fields {
		switch field.Name {
		case "DeletedAt":
			baseEntity.HasDeletedAt = true
			baseEntity.DeletedAtField = field
		case "Version":
			baseEntity.HasVersion = true
			baseEntity.VersionField = field
		case "CreatedAt":
			baseEntity.HasCreatedAt = true
			baseEntity.CreatedAtField = field
		case "UpdatedAt":
			baseEntity.HasUpdatedAt = true
			baseEntity.UpdatedAtField = field
		case "CreatedBy":
			baseEntity.HasCreatedBy = true
			baseEntity.CreatedByField = field
		case "UpdatedBy":
			baseEntity.HasUpdatedBy = true
			baseEntity.UpdatedByField = field
		}
	}

	return baseEntity
}

// identifyIDField 识别 ID 字段（根据优先级）
func (p *ASTParser) identifyIDField(fields []*metadata.FieldMetadata) *metadata.FieldMetadata {
	var candidates []*struct {
		field    *metadata.FieldMetadata
		priority int
	}

	for _, field := range fields {
		priority, isID := IdentifyIDField(field.Name, field.DBTag, field.Type)
		if isID {
			candidates = append(candidates, &struct {
				field    *metadata.FieldMetadata
				priority int
			}{field: field, priority: priority})
		}
	}

	// 如果没有候选字段，返回 nil
	if len(candidates) == 0 {
		return nil
	}

	// 找到优先级最高的（优先级数字越小越高）
	bestCandidate := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.priority < bestCandidate.priority {
			bestCandidate = candidate
		}
	}

	return bestCandidate.field
}
