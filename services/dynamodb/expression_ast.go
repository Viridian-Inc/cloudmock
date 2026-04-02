package dynamodb

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// ConditionExpr is a compiled condition expression that can be evaluated
// against a DynamoDB item with zero string parsing in the hot path.
type ConditionExpr interface {
	Evaluate(item Item) bool
}

// operand resolves to an AttributeValue from either an item field path or a
// literal value captured at compile time.
type operand interface {
	Resolve(item Item) (AttributeValue, bool)
}

// ---------------------------------------------------------------------------
// Operand implementations
// ---------------------------------------------------------------------------

type pathOperand struct{ path string }

func (o pathOperand) Resolve(item Item) (AttributeValue, bool) {
	v, ok := item[o.path]
	return v, ok
}

type literalOperand struct{ value AttributeValue }

func (o literalOperand) Resolve(_ Item) (AttributeValue, bool) {
	return o.value, o.value != nil
}

// ---------------------------------------------------------------------------
// Condition node implementations
// ---------------------------------------------------------------------------

type andExpr struct{ left, right ConditionExpr }

func (e *andExpr) Evaluate(item Item) bool {
	return e.left.Evaluate(item) && e.right.Evaluate(item)
}

type orExpr struct{ left, right ConditionExpr }

func (e *orExpr) Evaluate(item Item) bool {
	return e.left.Evaluate(item) || e.right.Evaluate(item)
}

type notExpr struct{ inner ConditionExpr }

func (e *notExpr) Evaluate(item Item) bool {
	return !e.inner.Evaluate(item)
}

type compareExpr struct {
	lhs operand
	op  string // =, <>, <, <=, >, >=
	rhs operand
}

func (e *compareExpr) Evaluate(item Item) bool {
	lv, lok := e.lhs.Resolve(item)
	rv, rok := e.rhs.Resolve(item)
	if !lok || !rok {
		return false
	}
	cmp, ok := fastCompare(lv, rv)
	if !ok {
		return false
	}
	switch e.op {
	case "=":
		return cmp == 0
	case "<>":
		return cmp != 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	}
	return false
}

type betweenExpr struct{ val, lo, hi operand }

func (e *betweenExpr) Evaluate(item Item) bool {
	v, vok := e.val.Resolve(item)
	loV, lok := e.lo.Resolve(item)
	hiV, hok := e.hi.Resolve(item)
	if !vok || !lok || !hok {
		return false
	}
	cmpLo, ok1 := fastCompare(v, loV)
	cmpHi, ok2 := fastCompare(v, hiV)
	if !ok1 || !ok2 {
		return false
	}
	return cmpLo >= 0 && cmpHi <= 0
}

type beginsWithExpr struct{ attr, prefix operand }

func (e *beginsWithExpr) Evaluate(item Item) bool {
	av, aok := e.attr.Resolve(item)
	pv, pok := e.prefix.Resolve(item)
	if !aok || !pok {
		return false
	}
	as, at := getAttrValue(av)
	ps, pt := getAttrValue(pv)
	if at != pt || at != "S" {
		return false
	}
	return strings.HasPrefix(fmt.Sprint(as), fmt.Sprint(ps))
}

type containsExpr struct{ attr, substr operand }

func (e *containsExpr) Evaluate(item Item) bool {
	av, aok := e.attr.Resolve(item)
	sv, sok := e.substr.Resolve(item)
	if !aok || !sok {
		return false
	}
	as, at := getAttrValue(av)
	ss, st := getAttrValue(sv)
	if at != st || at != "S" {
		return false
	}
	return strings.Contains(fmt.Sprint(as), fmt.Sprint(ss))
}

type attrExistsExpr struct{ path string }

func (e *attrExistsExpr) Evaluate(item Item) bool {
	_, ok := item[e.path]
	return ok
}

type attrNotExistsExpr struct{ path string }

func (e *attrNotExistsExpr) Evaluate(item Item) bool {
	_, ok := item[e.path]
	return !ok
}

type sizeExpr struct {
	path  string
	op    string
	value operand
}

func (e *sizeExpr) Evaluate(item Item) bool {
	av, ok := item[e.path]
	if !ok {
		return false
	}
	sz := attrSize(av)
	rv, rok := e.value.Resolve(item)
	if !rok {
		return false
	}
	rhs := AttributeValue{"N": fmt.Sprintf("%d", sz)}
	cmp, cok := compareValues(rhs, rv)
	if !cok {
		return false
	}
	switch e.op {
	case "=":
		return cmp == 0
	case "<>":
		return cmp != 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	}
	return false
}

type inExpr struct {
	val  operand
	list []operand
}

func (e *inExpr) Evaluate(item Item) bool {
	v, vok := e.val.Resolve(item)
	if !vok {
		return false
	}
	for _, li := range e.list {
		lv, lok := li.Resolve(item)
		if !lok {
			continue
		}
		cmp, ok := compareValues(v, lv)
		if ok && cmp == 0 {
			return true
		}
	}
	return false
}

// trueExpr is a no-op expression that always returns true (empty condition).
type trueExpr struct{}

func (trueExpr) Evaluate(_ Item) bool { return true }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fastCompare compares two AttributeValues without allocations for common types.
// For S and B types, it extracts the string directly without fmt.Sprint.
// For N type, it parses strings directly with strconv, falling back to big.Float
// only for very large numbers.
func fastCompare(a, b AttributeValue) (int, bool) {
	// Fast path: both are string type (S).
	if sa, ok := a["S"]; ok {
		if sb, ok := b["S"]; ok {
			as, aok := sa.(string)
			bs, bok := sb.(string)
			if aok && bok {
				if as < bs {
					return -1, true
				}
				if as > bs {
					return 1, true
				}
				return 0, true
			}
		}
		return 0, false
	}
	// Fast path: both are number type (N).
	if na, ok := a["N"]; ok {
		if nb, ok := b["N"]; ok {
			as, aok := na.(string)
			bs, bok := nb.(string)
			if aok && bok {
				return compareNumericStrings(as, bs)
			}
		}
		return 0, false
	}
	// Fast path: both are binary type (B).
	if ba, ok := a["B"]; ok {
		if bb, ok := b["B"]; ok {
			as, aok := ba.(string)
			bs, bok := bb.(string)
			if aok && bok {
				if as < bs {
					return -1, true
				}
				if as > bs {
					return 1, true
				}
				return 0, true
			}
		}
		return 0, false
	}
	// Fallback to the general-purpose compareValues.
	return compareValues(a, b)
}

// compareNumericStrings compares two numeric strings without allocating big.Float
// for simple integer/float cases. Falls back to big.Float for edge cases.
func compareNumericStrings(a, b string) (int, bool) {
	// Try fast float64 parse first (covers most DynamoDB numeric values).
	fa, erra := strconv.ParseFloat(a, 64)
	fb, errb := strconv.ParseFloat(b, 64)
	if erra == nil && errb == nil {
		// Check if values are small enough that float64 is exact.
		if fa < fb {
			return -1, true
		}
		if fa > fb {
			return 1, true
		}
		return 0, true
	}
	// Fallback to big.Float for arbitrary precision.
	na, oka := new(big.Float).SetString(a)
	nb, okb := new(big.Float).SetString(b)
	if !oka || !okb {
		return 0, false
	}
	return na.Cmp(nb), true
}

// attrSize returns the "size" of a DynamoDB attribute value.
func attrSize(av AttributeValue) int {
	if s, ok := av["S"]; ok {
		return len(fmt.Sprint(s))
	}
	if b, ok := av["B"]; ok {
		return len(fmt.Sprint(b))
	}
	if l, ok := av["L"]; ok {
		if list, ok := l.([]any); ok {
			return len(list)
		}
	}
	if m, ok := av["M"]; ok {
		if mp, ok := m.(map[string]any); ok {
			return len(mp)
		}
	}
	if ss, ok := av["SS"]; ok {
		if list, ok := ss.([]any); ok {
			return len(list)
		}
	}
	if ns, ok := av["NS"]; ok {
		if list, ok := ns.([]any); ok {
			return len(list)
		}
	}
	if bs, ok := av["BS"]; ok {
		if list, ok := bs.([]any); ok {
			return len(list)
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// Compiler
// ---------------------------------------------------------------------------

// CompileCondition parses a condition expression string once, resolving
// #name placeholders from names and binding :value placeholders from values,
// returning a ConditionExpr whose Evaluate method does no string parsing.
func CompileCondition(expr string, names map[string]string, values map[string]AttributeValue) ConditionExpr {
	expr = strings.TrimSpace(resolveNames(expr, names))
	if expr == "" {
		return trueExpr{}
	}
	return compileOr(expr, values)
}

func compileOr(expr string, values map[string]AttributeValue) ConditionExpr {
	parts := splitTopLevel(expr, " OR ")
	if len(parts) == 1 {
		return compileAnd(parts[0], values)
	}
	// Build left-associative tree.
	node := compileAnd(strings.TrimSpace(parts[0]), values)
	for i := 1; i < len(parts); i++ {
		node = &orExpr{left: node, right: compileAnd(strings.TrimSpace(parts[i]), values)}
	}
	return node
}

func compileAnd(expr string, values map[string]AttributeValue) ConditionExpr {
	parts := splitTopLevel(expr, " AND ")
	if len(parts) == 1 {
		return compileSingle(parts[0], values)
	}
	node := compileSingle(strings.TrimSpace(parts[0]), values)
	for i := 1; i < len(parts); i++ {
		node = &andExpr{left: node, right: compileSingle(strings.TrimSpace(parts[i]), values)}
	}
	return node
}

func compileSingle(expr string, values map[string]AttributeValue) ConditionExpr {
	expr = strings.TrimSpace(expr)

	// Handle NOT prefix.
	upper := strings.ToUpper(expr)
	if strings.HasPrefix(upper, "NOT ") {
		inner := strings.TrimSpace(expr[4:])
		return &notExpr{inner: compileSingle(inner, values)}
	}

	// Handle parenthesized expressions.
	if strings.HasPrefix(expr, "(") && matchingParen(expr, 0) == len(expr)-1 {
		return compileOr(expr[1:len(expr)-1], values)
	}

	lower := strings.ToLower(expr)

	// begins_with(path, :val)
	if strings.HasPrefix(lower, "begins_with(") || strings.HasPrefix(lower, "begins_with (") {
		return compileBeginsWith(expr, values)
	}

	// contains(path, :val)
	if strings.HasPrefix(lower, "contains(") || strings.HasPrefix(lower, "contains (") {
		return compileContains(expr, values)
	}

	// attribute_exists(path)
	if strings.HasPrefix(lower, "attribute_exists(") || strings.HasPrefix(lower, "attribute_exists (") {
		path := extractFuncArg(expr)
		return &attrExistsExpr{path: path}
	}

	// attribute_not_exists(path)
	if strings.HasPrefix(lower, "attribute_not_exists(") || strings.HasPrefix(lower, "attribute_not_exists (") {
		path := extractFuncArg(expr)
		return &attrNotExistsExpr{path: path}
	}

	// size(path) op :val
	if strings.HasPrefix(lower, "size(") || strings.HasPrefix(lower, "size (") {
		return compileSizeExpr(expr, values)
	}

	// BETWEEN: path BETWEEN :lo AND :hi
	upperExpr := strings.ToUpper(expr)
	if idx := strings.Index(upperExpr, " BETWEEN "); idx >= 0 {
		return compileBetween(expr, idx, values)
	}

	// IN: path IN (:a, :b, :c)
	if idx := strings.Index(upperExpr, " IN ("); idx >= 0 {
		return compileIn(expr, idx, values)
	}

	// Comparison operators: <=, >=, <>, <, >, =
	for _, op := range []string{"<=", ">=", "<>", "<", ">", "="} {
		idx := findComparisonOp(expr, op)
		if idx >= 0 {
			lhs := strings.TrimSpace(expr[:idx])
			rhs := strings.TrimSpace(expr[idx+len(op):])
			return &compareExpr{
				lhs: makeOperand(lhs, values),
				op:  op,
				rhs: makeOperand(rhs, values),
			}
		}
	}

	// Fallback: treat as always-false.
	return trueExpr{}
}

// findComparisonOp finds a comparison operator at top level (not inside parens).
func findComparisonOp(expr string, op string) int {
	depth := 0
	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth == 0 && i+len(op) <= len(expr) && expr[i:i+len(op)] == op {
				// For multi-char ops, make sure we're not matching a substring of another op.
				if len(op) == 1 {
					// For single-char ops like "<", ">" and "=", make sure next/prev char doesn't form a multi-char op.
					if op == "<" && i+1 < len(expr) && (expr[i+1] == '=' || expr[i+1] == '>') {
						continue
					}
					if op == ">" && i+1 < len(expr) && expr[i+1] == '=' {
						continue
					}
					if op == "=" && i > 0 && (expr[i-1] == '<' || expr[i-1] == '>' || expr[i-1] == '!') {
						continue
					}
				}
				return i
			}
		}
	}
	return -1
}

func makeOperand(s string, values map[string]AttributeValue) operand {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, ":") {
		v := values[s]
		return literalOperand{value: v}
	}
	return pathOperand{path: s}
}

func extractFuncArg(expr string) string {
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return ""
	}
	return strings.TrimSpace(expr[open+1 : close])
}

func compileBeginsWith(expr string, values map[string]AttributeValue) ConditionExpr {
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return trueExpr{}
	}
	args := expr[open+1 : close]
	comma := strings.Index(args, ",")
	if comma < 0 {
		return trueExpr{}
	}
	path := strings.TrimSpace(args[:comma])
	valRef := strings.TrimSpace(args[comma+1:])
	return &beginsWithExpr{
		attr:   makeOperand(path, values),
		prefix: makeOperand(valRef, values),
	}
}

func compileContains(expr string, values map[string]AttributeValue) ConditionExpr {
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return trueExpr{}
	}
	args := expr[open+1 : close]
	comma := strings.Index(args, ",")
	if comma < 0 {
		return trueExpr{}
	}
	path := strings.TrimSpace(args[:comma])
	valRef := strings.TrimSpace(args[comma+1:])
	return &containsExpr{
		attr:   makeOperand(path, values),
		substr: makeOperand(valRef, values),
	}
}

func compileBetween(expr string, idx int, values map[string]AttributeValue) ConditionExpr {
	path := strings.TrimSpace(expr[:idx])
	rest := expr[idx+len(" BETWEEN "):]
	upperRest := strings.ToUpper(rest)
	andIdx := strings.Index(upperRest, " AND ")
	if andIdx < 0 {
		return trueExpr{}
	}
	lo := strings.TrimSpace(rest[:andIdx])
	hi := strings.TrimSpace(rest[andIdx+5:])
	return &betweenExpr{
		val: makeOperand(path, values),
		lo:  makeOperand(lo, values),
		hi:  makeOperand(hi, values),
	}
}

func compileIn(expr string, idx int, values map[string]AttributeValue) ConditionExpr {
	lhs := strings.TrimSpace(expr[:idx])
	// Find the parenthesized list.
	rest := expr[idx+4:] // skip " IN "
	open := strings.Index(rest, "(")
	close := strings.LastIndex(rest, ")")
	if open < 0 || close < 0 {
		return trueExpr{}
	}
	inner := rest[open+1 : close]
	parts := strings.Split(inner, ",")
	list := make([]operand, 0, len(parts))
	for _, p := range parts {
		list = append(list, makeOperand(strings.TrimSpace(p), values))
	}
	return &inExpr{
		val:  makeOperand(lhs, values),
		list: list,
	}
}

func compileSizeExpr(expr string, values map[string]AttributeValue) ConditionExpr {
	// size(path) op :val
	closeParen := strings.Index(expr, ")")
	if closeParen < 0 {
		return trueExpr{}
	}
	openParen := strings.Index(expr, "(")
	if openParen < 0 {
		return trueExpr{}
	}
	path := strings.TrimSpace(expr[openParen+1 : closeParen])
	rest := strings.TrimSpace(expr[closeParen+1:])

	for _, op := range []string{"<=", ">=", "<>", "<", ">", "="} {
		if strings.HasPrefix(rest, op) {
			valRef := strings.TrimSpace(rest[len(op):])
			return &sizeExpr{
				path:  path,
				op:    op,
				value: makeOperand(valRef, values),
			}
		}
	}
	return trueExpr{}
}

// matchingParen returns the index of the closing paren matching the open paren at pos.
func matchingParen(expr string, pos int) int {
	depth := 0
	for i := pos; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// Compiled wrappers for Update and Projection
// ---------------------------------------------------------------------------

// ApplyUpdate applies an update expression to an item, resolving names upfront.
// It delegates to the existing applySet/applyRemove logic.
func ApplyUpdate(item Item, expr string, names map[string]string, values map[string]AttributeValue) Item {
	return parseUpdateExpression(item, expr, names, values)
}

// ApplyProjection applies a projection expression to an item.
func ApplyProjection(item Item, projExpr string, names map[string]string) Item {
	return applyProjection(item, projExpr, names)
}

// ---------------------------------------------------------------------------
// Expression Cache
// ---------------------------------------------------------------------------

// ExprCache caches compiled condition expressions keyed by the raw expression
// string. It is safe for concurrent use.
type ExprCache struct {
	mu    sync.RWMutex
	conds map[string]ConditionExpr
}

// NewExprCache creates a new expression cache.
func NewExprCache() *ExprCache {
	return &ExprCache{
		conds: make(map[string]ConditionExpr),
	}
}

// GetOrCompile returns a cached ConditionExpr or compiles and caches a new one.
// NOTE: The cache key is the raw expression string. Because literal values are
// bound at compile time, different values maps with the same expression string
// will return the first-compiled version. This is correct when the same
// expression+values pair is always used together (the common DynamoDB pattern).
func (c *ExprCache) GetOrCompile(expr string, names map[string]string, values map[string]AttributeValue) ConditionExpr {
	c.mu.RLock()
	if ce, ok := c.conds[expr]; ok {
		c.mu.RUnlock()
		return ce
	}
	c.mu.RUnlock()

	ce := CompileCondition(expr, names, values)

	c.mu.Lock()
	c.conds[expr] = ce
	c.mu.Unlock()

	return ce
}
