package dynamodb

import (
	"fmt"
	"math/big"
	"strings"
)

// extractEqualityValue extracts the partition key value from a resolved KeyConditionExpression.
// Handles patterns like "pk = :val", ":val = pk", "pk=:val AND sk > :sk".
// Returns the attrString-formatted value (e.g. "S:user1") for direct partition lookup,
// or "" if the expression can't be parsed.
func extractEqualityValue(resolvedExpr string, hashKeyName string, exprValues map[string]AttributeValue) string {
	// Split on AND to isolate conditions.
	parts := strings.SplitN(strings.ToUpper(resolvedExpr), " AND ", 2)
	original := resolvedExpr
	if len(parts) > 1 {
		// Use original casing for the first part.
		original = strings.TrimSpace(resolvedExpr[:len(parts[0])])
	}

	// Try "attrName = :placeholder" and ":placeholder = attrName"
	for _, sep := range []string{" = ", "= ", " ="} {
		idx := strings.Index(original, sep)
		if idx < 0 {
			continue
		}
		lhs := strings.TrimSpace(original[:idx])
		rhs := strings.TrimSpace(original[idx+len(sep):])

		var placeholder string
		if lhs == hashKeyName && strings.HasPrefix(rhs, ":") {
			placeholder = rhs
		} else if rhs == hashKeyName && strings.HasPrefix(lhs, ":") {
			placeholder = lhs
		}
		if placeholder == "" {
			continue
		}

		// Trim any trailing tokens (e.g. from bad split).
		if spaceIdx := strings.IndexByte(placeholder, ' '); spaceIdx >= 0 {
			placeholder = placeholder[:spaceIdx]
		}

		val, ok := exprValues[placeholder]
		if !ok {
			return ""
		}
		return attrString(val)
	}
	return ""
}

// resolveNames substitutes ExpressionAttributeNames (#name -> actualName) in a string.
func resolveNames(expr string, names map[string]string) string {
	if names == nil {
		return expr
	}
	for placeholder, actual := range names {
		expr = strings.ReplaceAll(expr, placeholder, actual)
	}
	return expr
}

// getAttrValue extracts a scalar value from an AttributeValue for comparison.
// Returns the string or numeric value and the type indicator.
func getAttrValue(av AttributeValue) (any, string) {
	if av == nil {
		return nil, ""
	}
	if v, ok := av["S"]; ok {
		return fmt.Sprint(v), "S"
	}
	if v, ok := av["N"]; ok {
		return fmt.Sprint(v), "N"
	}
	if v, ok := av["B"]; ok {
		return fmt.Sprint(v), "B"
	}
	if v, ok := av["BOOL"]; ok {
		return v, "BOOL"
	}
	if _, ok := av["NULL"]; ok {
		return nil, "NULL"
	}
	return nil, ""
}

// compareValues compares two attribute values.
// Returns -1, 0, or 1 for less, equal, greater.
// For S: lexicographic. For N: numeric.
func compareValues(a, b AttributeValue) (int, bool) {
	va, ta := getAttrValue(a)
	vb, tb := getAttrValue(b)
	if ta != tb || ta == "" {
		return 0, false
	}
	switch ta {
	case "S", "B":
		sa := fmt.Sprint(va)
		sb := fmt.Sprint(vb)
		if sa < sb {
			return -1, true
		}
		if sa > sb {
			return 1, true
		}
		return 0, true
	case "N":
		na, oka := new(big.Float).SetString(fmt.Sprint(va))
		nb, okb := new(big.Float).SetString(fmt.Sprint(vb))
		if !oka || !okb {
			return 0, false
		}
		return na.Cmp(nb), true
	}
	return 0, false
}

// evaluateCondition evaluates a condition expression against an item.
// Supports: =, <, >, <=, >=, BETWEEN, begins_with, attribute_exists, attribute_not_exists, AND, OR.
// It compiles the expression into an AST once and then evaluates it.
func evaluateCondition(expr string, item Item, names map[string]string, values map[string]AttributeValue) bool {
	return CompileCondition(expr, names, values).Evaluate(item)
}

// evalOr splits on top-level OR.
func evalOr(expr string, item Item, values map[string]AttributeValue) bool {
	parts := splitTopLevel(expr, " OR ")
	for _, p := range parts {
		if evalAnd(strings.TrimSpace(p), item, values) {
			return true
		}
	}
	return false
}

// evalAnd splits on top-level AND.
func evalAnd(expr string, item Item, values map[string]AttributeValue) bool {
	parts := splitTopLevel(expr, " AND ")
	for _, p := range parts {
		if !evalSingle(strings.TrimSpace(p), item, values) {
			return false
		}
	}
	return true
}

// splitTopLevel splits an expression by a keyword, but only at the top level
// (not inside parentheses or function calls).
// When splitting by " AND ", it skips the AND that is part of "BETWEEN ... AND ...".
func splitTopLevel(expr, sep string) []string {
	sepUpper := strings.ToUpper(sep)
	exprUpper := strings.ToUpper(expr)

	var parts []string
	depth := 0
	start := 0
	i := 0
	for i < len(expr) {
		if expr[i] == '(' {
			depth++
			i++
		} else if expr[i] == ')' {
			depth--
			i++
		} else if depth == 0 && i+len(sep) <= len(expr) && exprUpper[i:i+len(sep)] == sepUpper {
			// When splitting on AND, check if this AND belongs to a BETWEEN clause.
			if sepUpper == " AND " && isBetweenAnd(exprUpper, i) {
				i += len(sep)
				continue
			}
			parts = append(parts, expr[start:i])
			start = i + len(sep)
			i = start
		} else {
			i++
		}
	}
	parts = append(parts, expr[start:])
	return parts
}

// isBetweenAnd checks if the " AND " at position idx is part of a BETWEEN ... AND ... clause.
// It looks backward from idx to see if there's a BETWEEN keyword without an intervening AND.
func isBetweenAnd(upper string, idx int) bool {
	// Look at the text before this AND for a BETWEEN keyword.
	before := upper[:idx]
	betweenIdx := strings.LastIndex(before, " BETWEEN ")
	if betweenIdx < 0 {
		return false
	}
	// Check that there is no other AND between the BETWEEN and this position.
	// The text between BETWEEN and this AND should be just the lower bound value.
	segment := before[betweenIdx+len(" BETWEEN "):]
	return !strings.Contains(segment, " AND ")
}

// evalSingle evaluates a single condition (no AND/OR at top level).
func evalSingle(expr string, item Item, values map[string]AttributeValue) bool {
	expr = strings.TrimSpace(expr)

	// Handle parenthesized expressions.
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evaluateCondition(expr[1:len(expr)-1], item, nil, values)
	}

	// begins_with(path, :val)
	lower := strings.ToLower(expr)
	if strings.HasPrefix(lower, "begins_with(") || strings.HasPrefix(lower, "begins_with (") {
		return evalBeginsWith(expr, item, values)
	}

	// attribute_exists(path)
	if strings.HasPrefix(lower, "attribute_exists(") || strings.HasPrefix(lower, "attribute_exists (") {
		return evalAttributeExists(expr, item)
	}

	// attribute_not_exists(path)
	if strings.HasPrefix(lower, "attribute_not_exists(") || strings.HasPrefix(lower, "attribute_not_exists (") {
		return !evalAttributeExists(strings.Replace(expr, "not_exists", "exists", 1), item)
	}

	// BETWEEN: path BETWEEN :a AND :b
	upperExpr := strings.ToUpper(expr)
	if idx := strings.Index(upperExpr, " BETWEEN "); idx >= 0 {
		return evalBetween(expr, idx, item, values)
	}

	// Comparison operators: <=, >=, <>, <, >, =
	for _, op := range []string{"<=", ">=", "<>", "<", ">", "="} {
		idx := strings.Index(expr, op)
		if idx >= 0 {
			lhs := strings.TrimSpace(expr[:idx])
			rhs := strings.TrimSpace(expr[idx+len(op):])
			return evalComparison(lhs, op, rhs, item, values)
		}
	}

	return false
}

// resolveOperand resolves an operand to an AttributeValue.
// If it starts with ":", it's a value placeholder. Otherwise, it's an attribute path.
func resolveOperand(operand string, item Item, values map[string]AttributeValue) (AttributeValue, bool) {
	operand = strings.TrimSpace(operand)
	if strings.HasPrefix(operand, ":") {
		v, ok := values[operand]
		return v, ok
	}
	v, ok := item[operand]
	return v, ok
}

func evalComparison(lhs, op, rhs string, item Item, values map[string]AttributeValue) bool {
	lhsVal, lhsOK := resolveOperand(lhs, item, values)
	rhsVal, rhsOK := resolveOperand(rhs, item, values)
	if !lhsOK || !rhsOK {
		return false
	}
	cmp, ok := compareValues(lhsVal, rhsVal)
	if !ok {
		return false
	}
	switch op {
	case "=":
		return cmp == 0
	case "<":
		return cmp < 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case ">=":
		return cmp >= 0
	case "<>":
		return cmp != 0
	}
	return false
}

func evalBetween(expr string, idx int, item Item, values map[string]AttributeValue) bool {
	path := strings.TrimSpace(expr[:idx])
	rest := expr[idx+len(" BETWEEN "):]

	// Split on AND (case-insensitive)
	upperRest := strings.ToUpper(rest)
	andIdx := strings.Index(upperRest, " AND ")
	if andIdx < 0 {
		return false
	}
	lo := strings.TrimSpace(rest[:andIdx])
	hi := strings.TrimSpace(rest[andIdx+5:])

	pathVal, pathOK := resolveOperand(path, item, values)
	loVal, loOK := resolveOperand(lo, item, values)
	hiVal, hiOK := resolveOperand(hi, item, values)
	if !pathOK || !loOK || !hiOK {
		return false
	}

	cmpLo, ok1 := compareValues(pathVal, loVal)
	cmpHi, ok2 := compareValues(pathVal, hiVal)
	if !ok1 || !ok2 {
		return false
	}
	return cmpLo >= 0 && cmpHi <= 0
}

func evalBeginsWith(expr string, item Item, values map[string]AttributeValue) bool {
	// Extract args from begins_with(path, :val)
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return false
	}
	args := expr[open+1 : close]
	comma := strings.Index(args, ",")
	if comma < 0 {
		return false
	}
	path := strings.TrimSpace(args[:comma])
	valRef := strings.TrimSpace(args[comma+1:])

	pathVal, pathOK := resolveOperand(path, item, values)
	prefixVal, prefixOK := resolveOperand(valRef, item, values)
	if !pathOK || !prefixOK {
		return false
	}

	ps, pt := getAttrValue(pathVal)
	vs, vt := getAttrValue(prefixVal)
	if pt != vt || pt != "S" {
		return false
	}
	return strings.HasPrefix(fmt.Sprint(ps), fmt.Sprint(vs))
}

func evalAttributeExists(expr string, item Item) bool {
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return false
	}
	path := strings.TrimSpace(expr[open+1 : close])
	_, ok := item[path]
	return ok
}

// parseUpdateExpression applies an UpdateExpression to an item.
// Supports SET and REMOVE clauses.
func parseUpdateExpression(item Item, expr string, names map[string]string, values map[string]AttributeValue) Item {
	if item == nil {
		item = make(Item)
	}
	expr = resolveNames(expr, names)
	expr = strings.TrimSpace(expr)

	// Split into clauses (SET ..., REMOVE ...)
	// We handle SET and REMOVE keywords.
	upper := strings.ToUpper(expr)

	setIdx := strings.Index(upper, "SET ")
	removeIdx := strings.Index(upper, "REMOVE ")

	if setIdx >= 0 {
		end := len(expr)
		if removeIdx > setIdx {
			end = removeIdx
		}
		setClause := strings.TrimSpace(expr[setIdx+4 : end])
		applySet(item, setClause, values)
	}

	if removeIdx >= 0 {
		end := len(expr)
		if setIdx > removeIdx {
			end = setIdx
		}
		removeClause := strings.TrimSpace(expr[removeIdx+7 : end])
		applyRemove(item, removeClause)
	}

	return item
}

func applySet(item Item, clause string, values map[string]AttributeValue) {
	// SET #a = :v1, #b = :v2
	assignments := splitAssignments(clause)
	for _, a := range assignments {
		eqIdx := strings.Index(a, "=")
		if eqIdx < 0 {
			continue
		}
		path := strings.TrimSpace(a[:eqIdx])
		valRef := strings.TrimSpace(a[eqIdx+1:])

		if strings.HasPrefix(valRef, ":") {
			if v, ok := values[valRef]; ok {
				item[path] = v
			}
		} else if strings.HasPrefix(valRef, "if_not_exists(") {
			// if_not_exists(path, :val) - simplified
			item[path] = resolveIfNotExists(item, valRef, values)
		} else if strings.Contains(valRef, "+") || strings.Contains(valRef, "-") {
			// Arithmetic: path + :val or path - :val
			item[path] = resolveArithmetic(item, valRef, values)
		}
	}
}

func splitAssignments(clause string) []string {
	// Split by comma, but not inside parentheses.
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(clause); i++ {
		switch clause[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(clause[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(clause[start:]))
	return parts
}

func resolveIfNotExists(item Item, expr string, values map[string]AttributeValue) AttributeValue {
	// if_not_exists(path, :val)
	open := strings.Index(expr, "(")
	close := strings.LastIndex(expr, ")")
	if open < 0 || close < 0 {
		return nil
	}
	args := expr[open+1 : close]
	comma := strings.Index(args, ",")
	if comma < 0 {
		return nil
	}
	path := strings.TrimSpace(args[:comma])
	valRef := strings.TrimSpace(args[comma+1:])

	if existing, ok := item[path]; ok {
		return existing
	}
	if v, ok := values[valRef]; ok {
		return v
	}
	return nil
}

func resolveArithmetic(item Item, expr string, values map[string]AttributeValue) AttributeValue {
	// Simple: path + :val or path - :val
	var parts []string
	var op byte
	for _, candidate := range []byte{'+', '-'} {
		idx := strings.IndexByte(expr, candidate)
		if idx > 0 {
			parts = []string{strings.TrimSpace(expr[:idx]), strings.TrimSpace(expr[idx+1:])}
			op = candidate
			break
		}
	}
	if len(parts) != 2 || op == 0 {
		return nil
	}

	resolve := func(s string) *big.Float {
		var av AttributeValue
		if strings.HasPrefix(s, ":") {
			av = values[s]
		} else {
			av = item[s]
		}
		if av == nil {
			return nil
		}
		if n, ok := av["N"]; ok {
			f, _, _ := new(big.Float).Parse(fmt.Sprint(n), 10)
			return f
		}
		return nil
	}

	a := resolve(parts[0])
	b := resolve(parts[1])
	if a == nil || b == nil {
		return nil
	}

	var result *big.Float
	if op == '+' {
		result = new(big.Float).Add(a, b)
	} else {
		result = new(big.Float).Sub(a, b)
	}
	return AttributeValue{"N": result.Text('f', -1)}
}

func applyRemove(item Item, clause string) {
	// REMOVE #a, #b
	parts := strings.Split(clause, ",")
	for _, p := range parts {
		attr := strings.TrimSpace(p)
		if attr != "" {
			delete(item, attr)
		}
	}
}

// applyProjection filters an item to only include the specified attributes.
func applyProjection(item Item, projExpr string, names map[string]string) Item {
	if projExpr == "" {
		return item
	}
	projExpr = resolveNames(projExpr, names)
	attrs := strings.Split(projExpr, ",")
	result := make(Item)
	for _, a := range attrs {
		a = strings.TrimSpace(a)
		if v, ok := item[a]; ok {
			result[a] = v
		}
	}
	return result
}
