// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file was copied from github.com/jessevdk/go-operators,
// which itself was forked from go/ast walk.go.

package genlib

import (
	"fmt"
	"go/ast"
)

type ReplaceFunc func(ast.Node) ast.Node

func replaceIdentList(r ReplaceFunc, list []*ast.Ident) {
	for i, x := range list {
		list[i] = replace(r, x).(*ast.Ident)
	}
}

func replaceExprList(r ReplaceFunc, list []ast.Expr) {
	for i, x := range list {
		list[i] = replace(r, x).(ast.Expr)
	}
}

func replaceStmtList(r ReplaceFunc, list []ast.Stmt) {
	for i, x := range list {
		list[i] = replace(r, x).(ast.Stmt)
	}
}

func replaceDeclList(r ReplaceFunc, list []ast.Decl) {
	for i, x := range list {
		list[i] = replace(r, x).(ast.Decl)
	}
}

func replace(r ReplaceFunc, node ast.Node) ast.Node {
	node = r(node)

	if node == nil {
		return nil
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := node.(type) {
	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:
		for i, c := range n.List {
			n.List[i] = replace(r, c).(*ast.Comment)
		}

	case *ast.Field:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		replaceIdentList(r, n.Names)

		n.Type = replace(r, n.Type).(ast.Expr)

		if n.Tag != nil {
			n.Tag = replace(r, n.Tag).(*ast.BasicLit)
		}

		if n.Comment != nil {
			n.Comment = replace(r, n.Comment).(*ast.CommentGroup)
		}

	case *ast.FieldList:
		for i, f := range n.List {
			n.List[i] = replace(r, f).(*ast.Field)
		}

	// Expressions
	case *ast.BadExpr, *ast.Ident, *ast.BasicLit:
		// nothing to do

	case *ast.Ellipsis:
		if n.Elt != nil {
			n.Elt = replace(r, n.Elt).(ast.Expr)
		}

	case *ast.FuncLit:
		n.Type = replace(r, n.Type).(*ast.FuncType)
		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	case *ast.CompositeLit:
		if n.Type != nil {
			n.Type = replace(r, n.Type).(ast.Expr)
		}

		replaceExprList(r, n.Elts)

	case *ast.ParenExpr:
		n.X = replace(r, n.X).(ast.Expr)

	case *ast.SelectorExpr:
		n.X = replace(r, n.X).(ast.Expr)
		n.Sel = replace(r, n.Sel).(*ast.Ident)

	case *ast.IndexExpr:
		n.X = replace(r, n.X).(ast.Expr)
		n.Index = replace(r, n.Index).(ast.Expr)

	case *ast.SliceExpr:
		n.X = replace(r, n.X).(ast.Expr)

		if n.Low != nil {
			n.Low = replace(r, n.Low).(ast.Expr)
		}

		if n.High != nil {
			n.High = replace(r, n.High).(ast.Expr)
		}

	case *ast.TypeAssertExpr:
		n.X = replace(r, n.X).(ast.Expr)

		if n.Type != nil {
			n.Type = replace(r, n.Type).(ast.Expr)
		}

	case *ast.CallExpr:
		n.Fun = replace(r, n.Fun).(ast.Expr)
		replaceExprList(r, n.Args)

	case *ast.StarExpr:
		n.X = replace(r, n.X).(ast.Expr)

	case *ast.UnaryExpr:
		n.X = replace(r, n.X).(ast.Expr)

	case *ast.BinaryExpr:
		n.X = replace(r, n.X).(ast.Expr)
		n.Y = replace(r, n.Y).(ast.Expr)

	case *ast.KeyValueExpr:
		n.Key = replace(r, n.Key).(ast.Expr)
		n.Value = replace(r, n.Value).(ast.Expr)

	// Types
	case *ast.ArrayType:
		if n.Len != nil {
			n.Len = replace(r, n.Len).(ast.Expr)
		}

		n.Elt = replace(r, n.Elt).(ast.Expr)

	case *ast.StructType:
		n.Fields = replace(r, n.Fields).(*ast.FieldList)

	case *ast.FuncType:
		n.Params = replace(r, n.Params).(*ast.FieldList)

		if n.Results != nil {
			n.Results = replace(r, n.Results).(*ast.FieldList)
		}

	case *ast.InterfaceType:
		n.Methods = replace(r, n.Methods).(*ast.FieldList)

	case *ast.MapType:
		n.Key = replace(r, n.Key).(ast.Expr)
		n.Value = replace(r, n.Value).(ast.Expr)

	case *ast.ChanType:
		n.Value = replace(r, n.Value).(ast.Expr)

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		n.Decl = replace(r, n.Decl).(ast.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		n.Label = replace(r, n.Label).(*ast.Ident)
		n.Stmt = replace(r, n.Stmt).(ast.Stmt)

	case *ast.ExprStmt:
		n.X = replace(r, n.X).(ast.Expr)

	case *ast.SendStmt:
		n.Chan = replace(r, n.Chan).(ast.Expr)
		n.Value = replace(r, n.Value).(ast.Expr)

	case *ast.IncDecStmt:
		n.X = replace(r, n.X).(ast.Expr)

	case *ast.AssignStmt:
		replaceExprList(r, n.Lhs)
		replaceExprList(r, n.Rhs)

	case *ast.GoStmt:
		n.Call = replace(r, n.Call).(*ast.CallExpr)

	case *ast.DeferStmt:
		n.Call = replace(r, n.Call).(*ast.CallExpr)

	case *ast.ReturnStmt:
		replaceExprList(r, n.Results)

	case *ast.BranchStmt:
		if n.Label != nil {
			n.Label = replace(r, n.Label).(*ast.Ident)
		}

	case *ast.BlockStmt:
		replaceStmtList(r, n.List)

	case *ast.IfStmt:
		if n.Init != nil {
			n.Init = replace(r, n.Init).(ast.Stmt)
		}

		n.Cond = replace(r, n.Cond).(ast.Expr)
		n.Body = replace(r, n.Body).(*ast.BlockStmt)

		if n.Else != nil {
			n.Else = replace(r, n.Else).(ast.Stmt)
		}

	case *ast.CaseClause:
		replaceExprList(r, n.List)
		replaceStmtList(r, n.Body)

	case *ast.SwitchStmt:
		if n.Init != nil {
			n.Init = replace(r, n.Init).(ast.Stmt)
		}

		if n.Tag != nil {
			n.Tag = replace(r, n.Tag).(ast.Expr)
		}

		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			n.Init = replace(r, n.Init).(ast.Stmt)
		}

		n.Assign = replace(r, n.Assign).(ast.Stmt)
		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	case *ast.CommClause:
		if n.Comm != nil {
			n.Comm = replace(r, n.Comm).(ast.Stmt)
		}

		replaceStmtList(r, n.Body)

	case *ast.SelectStmt:
		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	case *ast.ForStmt:
		if n.Init != nil {
			n.Init = replace(r, n.Init).(ast.Stmt)
		}

		if n.Cond != nil {
			n.Cond = replace(r, n.Cond).(ast.Expr)
		}

		if n.Post != nil {
			n.Post = replace(r, n.Post).(ast.Stmt)
		}

		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	case *ast.RangeStmt:
		n.Key = replace(r, n.Key).(ast.Expr)

		if n.Value != nil {
			n.Value = replace(r, n.Value).(ast.Expr)
		}

		n.X = replace(r, n.X).(ast.Expr)
		n.Body = replace(r, n.Body).(*ast.BlockStmt)

	// Declarations
	case *ast.ImportSpec:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		if n.Name != nil {
			n.Name = replace(r, n.Name).(*ast.Ident)
		}

		n.Path = replace(r, n.Path).(*ast.BasicLit)

		if n.Comment != nil {
			n.Comment = replace(r, n.Comment).(*ast.CommentGroup)
		}

	case *ast.ValueSpec:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		replaceIdentList(r, n.Names)

		if n.Type != nil {
			n.Type = replace(r, n.Type).(ast.Expr)
		}

		replaceExprList(r, n.Values)

		if n.Comment != nil {
			n.Comment = replace(r, n.Comment).(*ast.CommentGroup)
		}

	case *ast.TypeSpec:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		n.Name = replace(r, n.Name).(*ast.Ident)
		n.Type = replace(r, n.Type).(ast.Expr)

		if n.Comment != nil {
			n.Comment = replace(r, n.Comment).(*ast.CommentGroup)
		}

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		for i, s := range n.Specs {
			n.Specs[i] = replace(r, s).(ast.Spec)
		}

	case *ast.FuncDecl:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		if n.Recv != nil {
			n.Recv = replace(r, n.Recv).(*ast.FieldList)
		}

		n.Name = replace(r, n.Name).(*ast.Ident)
		n.Type = replace(r, n.Type).(*ast.FuncType)

		if n.Body != nil {
			n.Body = replace(r, n.Body).(*ast.BlockStmt)
		}

	// Files and packages
	case *ast.File:
		if n.Doc != nil {
			n.Doc = replace(r, n.Doc).(*ast.CommentGroup)
		}

		n.Name = replace(r, n.Name).(*ast.Ident)

		replaceDeclList(r, n.Decls)

		for i, g := range n.Comments {
			n.Comments[i] = replace(r, g).(*ast.CommentGroup)
		}

		// don't walk n.Comments - they have been
		// visited already through the individual
		// nodes

	case *ast.Package:
		for i, f := range n.Files {
			n.Files[i] = replace(r, f).(*ast.File)
		}

	default:
		fmt.Printf("replace: unexpected node type %T", n)
		panic("replace")
	}

	return node
}
