package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <file.go>", os.Args[0])
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	// Собираем диапазоны тел функций
	var bodyRanges []struct{ start, end token.Pos }

	ast.Inspect(f, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Body != nil {
			bodyRanges = append(bodyRanges, struct{ start, end token.Pos }{
				start: fn.Body.Lbrace + 1, // после '{'
				end:   fn.Body.Rbrace,     // до '}'
			})
			fn.Body = &ast.BlockStmt{} // обнуляем тело
		}
		return true
	})

	// Фильтруем комментарии: оставляем только те, что НЕ внутри тел функций
	var keptComments []*ast.CommentGroup
	for _, cg := range f.Comments {
		keep := true
		for _, r := range bodyRanges {
			// Если комментарий начинается внутри тела функции — удаляем
			if cg.Pos() >= r.start && cg.End() <= r.end {
				keep = false
				break
			}
		}
		if keep {
			keptComments = append(keptComments, cg)
		}
	}
	f.Comments = keptComments

	// Форматируем и выводим
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		log.Fatalf("format error: %v", err)
	}
	os.Stdout.Write(buf.Bytes())
}
