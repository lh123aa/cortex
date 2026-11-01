package symbols

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// extractGoAST 用 Go 官方解析器（标准库 go/parser）提取 Go 符号。
//
// 为什么 Go 要特殊处理：
//   - 规则法（正则/逐行）在多行签名、泛型、字符串里的"假代码"、被注释掉的函数上会漏判或误判。
//   - Go 自带完整解析器，零外部依赖、零 CGO，准确率是 100%。
//
// 只有 Go 语言值得这么做：其他语言没有标准库级别的解析器，
// 引入第三方（如 tree-sitter）需要 CGO，会破坏「纯 Go 单二进制」的承诺。
//
// 返回 error 时调用方应回退到规则法（例如文件本身有语法错误，
// 解析器只能给出部分 AST，此时规则法反而更宽容）。
func extractGoAST(path, content string) ([]Symbol, error) {
	src := []byte(content)
	fset := token.NewFileSet()

	// ParseComments 让我们能识别文档注释；
	// SkipObjectResolution 跳过已废弃的标识符解析，快一些。
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	out := make([]Symbol, 0, 64)
	add := func(name string, kind Kind, pos token.Pos, sig string) {
		name = strings.TrimSpace(name)
		if name == "" || name == "_" {
			return
		}
		line := fset.Position(pos).Line
		out = append(out, Symbol{
			Name:      name,
			Kind:      kind,
			Line:      line,
			Signature: truncateLine(collapseSpace(sig), 200),
			Language:  "go",
			Path:      path,
		})
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			kind := KindFunction
			if d.Recv != nil && len(d.Recv.List) > 0 {
				kind = KindMethod
			}
			// 签名取 Pos → 函数体左花括号，天然覆盖多行声明，再压缩空白。
			end := d.Type.End()
			if d.Body != nil {
				end = d.Body.Lbrace
			}
			sig := sliceSource(src, fset, d.Pos(), end)
			if sig == "" {
				sig = "func " + d.Name.Name + "(…)"
			}
			add(d.Name.Name, kind, d.Name.Pos(), sig)

		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				// var / const / import 暂不当作符号：
				// 数量大、噪声高，对「找函数/类型定义」帮助有限。
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				kind := KindType
				switch ts.Type.(type) {
				case *ast.StructType:
					kind = KindStruct
				case *ast.InterfaceType:
					kind = KindInterface
				}
				// 类型签名取声明所在行；struct/interface 不把整个 body 塞进签名。
				sig := lineAt(src, fset, ts.Pos())
				add(ts.Name.Name, kind, ts.Name.Pos(), sig)
			}
		}
	}

	return dedupe(out), nil
}

// sliceSource 截取源码 [start, end) 区间。
func sliceSource(src []byte, fset *token.FileSet, start, end token.Pos) string {
	s := fset.Position(start).Offset
	e := fset.Position(end).Offset
	if s < 0 || e < 0 || s >= e || e > len(src) {
		return ""
	}
	return string(src[s:e])
}

// lineAt 返回 pos 所在行的内容。
func lineAt(src []byte, fset *token.FileSet, pos token.Pos) string {
	p := fset.Position(pos)
	s := p.Offset
	if s < 0 || s > len(src) {
		return ""
	}
	// 回退到行首
	for s > 0 && src[s-1] != '\n' {
		s--
	}
	e := p.Offset
	for e < len(src) && src[e] != '\n' {
		e++
	}
	return strings.TrimSpace(string(src[s:e]))
}

// collapseSpace 把连续空白（含换行）压成单个空格。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
