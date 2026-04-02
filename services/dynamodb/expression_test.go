package dynamodb

import (
	"testing"
)

func TestCompileCondition_Equal(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
		"age":  AttributeValue{"N": "30"},
	}
	values := map[string]AttributeValue{
		":v": {"S": "alice"},
	}
	ce := CompileCondition("name = :v", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected equal to match")
	}
	values2 := map[string]AttributeValue{
		":v": {"S": "bob"},
	}
	ce2 := CompileCondition("name = :v", nil, values2)
	if ce2.Evaluate(item) {
		t.Fatal("expected equal to not match")
	}
}

func TestCompileCondition_LessThan(t *testing.T) {
	item := Item{
		"age": AttributeValue{"N": "25"},
	}
	values := map[string]AttributeValue{
		":v": {"N": "30"},
	}
	ce := CompileCondition("age < :v", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected < to match")
	}
}

func TestCompileCondition_GreaterThan(t *testing.T) {
	item := Item{
		"age": AttributeValue{"N": "50"},
	}
	values := map[string]AttributeValue{
		":v": {"N": "30"},
	}
	ce := CompileCondition("age > :v", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected > to match")
	}
}

func TestCompileCondition_BeginsWith(t *testing.T) {
	item := Item{
		"email": AttributeValue{"S": "alice@example.com"},
	}
	values := map[string]AttributeValue{
		":prefix": {"S": "alice"},
	}
	ce := CompileCondition("begins_with(email, :prefix)", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected begins_with to match")
	}
	values2 := map[string]AttributeValue{
		":prefix": {"S": "bob"},
	}
	ce2 := CompileCondition("begins_with(email, :prefix)", nil, values2)
	if ce2.Evaluate(item) {
		t.Fatal("expected begins_with to not match")
	}
}

func TestCompileCondition_Contains(t *testing.T) {
	item := Item{
		"bio": AttributeValue{"S": "software engineer at acme"},
	}
	values := map[string]AttributeValue{
		":sub": {"S": "engineer"},
	}
	ce := CompileCondition("contains(bio, :sub)", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected contains to match")
	}
	values2 := map[string]AttributeValue{
		":sub": {"S": "doctor"},
	}
	ce2 := CompileCondition("contains(bio, :sub)", nil, values2)
	if ce2.Evaluate(item) {
		t.Fatal("expected contains to not match")
	}
}

func TestCompileCondition_Between(t *testing.T) {
	item := Item{
		"age": AttributeValue{"N": "25"},
	}
	values := map[string]AttributeValue{
		":lo": {"N": "20"},
		":hi": {"N": "30"},
	}
	ce := CompileCondition("age BETWEEN :lo AND :hi", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected BETWEEN to match")
	}
	item2 := Item{
		"age": AttributeValue{"N": "35"},
	}
	if ce.Evaluate(item2) {
		t.Fatal("expected BETWEEN to not match for out-of-range")
	}
}

func TestCompileCondition_AND(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
		"age":  AttributeValue{"N": "30"},
	}
	values := map[string]AttributeValue{
		":name": {"S": "alice"},
		":age":  {"N": "25"},
	}
	ce := CompileCondition("name = :name AND age > :age", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected AND to match")
	}
}

func TestCompileCondition_OR(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "bob"},
		"age":  AttributeValue{"N": "30"},
	}
	values := map[string]AttributeValue{
		":name": {"S": "alice"},
		":age":  {"N": "25"},
	}
	ce := CompileCondition("name = :name OR age > :age", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected OR to match (second clause true)")
	}
}

func TestCompileCondition_AttributeExists(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
	}
	ce := CompileCondition("attribute_exists(name)", nil, nil)
	if !ce.Evaluate(item) {
		t.Fatal("expected attribute_exists to match")
	}
	ce2 := CompileCondition("attribute_exists(email)", nil, nil)
	if ce2.Evaluate(item) {
		t.Fatal("expected attribute_exists to not match for missing attr")
	}
}

func TestCompileCondition_AttributeNotExists(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
	}
	ce := CompileCondition("attribute_not_exists(email)", nil, nil)
	if !ce.Evaluate(item) {
		t.Fatal("expected attribute_not_exists to match for missing attr")
	}
}

func TestCompileCondition_Size(t *testing.T) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
	}
	values := map[string]AttributeValue{
		":len": {"N": "5"},
	}
	ce := CompileCondition("size(name) = :len", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected size() = 5 to match 'alice'")
	}
	values2 := map[string]AttributeValue{
		":len": {"N": "3"},
	}
	ce2 := CompileCondition("size(name) > :len", nil, values2)
	if !ce2.Evaluate(item) {
		t.Fatal("expected size() > 3 to match 'alice' (len=5)")
	}
}

func TestCompileCondition_WithNames(t *testing.T) {
	item := Item{
		"status": AttributeValue{"S": "active"},
	}
	names := map[string]string{
		"#s": "status",
	}
	values := map[string]AttributeValue{
		":v": {"S": "active"},
	}
	ce := CompileCondition("#s = :v", names, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected name-resolved condition to match")
	}
}

func TestCompileCondition_Empty(t *testing.T) {
	item := Item{}
	ce := CompileCondition("", nil, nil)
	if !ce.Evaluate(item) {
		t.Fatal("empty condition should return true")
	}
}

func TestApplyUpdate_SET(t *testing.T) {
	item := Item{
		"id": AttributeValue{"S": "1"},
	}
	names := map[string]string{"#n": "name"}
	values := map[string]AttributeValue{
		":v": {"S": "alice"},
	}
	result := ApplyUpdate(item, "SET #n = :v", names, values)
	if result["name"] == nil {
		t.Fatal("expected name to be set")
	}
	v, _ := getAttrValue(result["name"])
	if v != "alice" {
		t.Fatalf("expected alice, got %v", v)
	}
}

func TestApplyUpdate_REMOVE(t *testing.T) {
	item := Item{
		"id":   AttributeValue{"S": "1"},
		"temp": AttributeValue{"S": "delete-me"},
	}
	result := ApplyUpdate(item, "REMOVE temp", nil, nil)
	if _, ok := result["temp"]; ok {
		t.Fatal("expected temp to be removed")
	}
}

func TestApplyProjection(t *testing.T) {
	item := Item{
		"id":    AttributeValue{"S": "1"},
		"name":  AttributeValue{"S": "alice"},
		"email": AttributeValue{"S": "alice@example.com"},
	}
	result := ApplyProjection(item, "id, name", nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(result))
	}
	if _, ok := result["email"]; ok {
		t.Fatal("expected email to be excluded")
	}
}

func TestApplyProjection_WithNames(t *testing.T) {
	item := Item{
		"id":    AttributeValue{"S": "1"},
		"name":  AttributeValue{"S": "alice"},
		"email": AttributeValue{"S": "alice@example.com"},
	}
	names := map[string]string{"#n": "name"}
	result := ApplyProjection(item, "id, #n", names)
	if len(result) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(result))
	}
}

func TestExprCache(t *testing.T) {
	cache := NewExprCache()
	values := map[string]AttributeValue{
		":v": {"S": "alice"},
	}

	ce1 := cache.GetOrCompile("name = :v", nil, values)
	ce2 := cache.GetOrCompile("name = :v", nil, values)

	// Both should work correctly.
	item := Item{"name": AttributeValue{"S": "alice"}}
	if !ce1.Evaluate(item) {
		t.Fatal("cached expr should match")
	}
	if !ce2.Evaluate(item) {
		t.Fatal("second cached expr should match")
	}
}

func TestCompileCondition_NotEqual(t *testing.T) {
	item := Item{
		"status": AttributeValue{"S": "active"},
	}
	values := map[string]AttributeValue{
		":v": {"S": "inactive"},
	}
	ce := CompileCondition("status <> :v", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected <> to match")
	}
}

func TestCompileCondition_Parenthesized(t *testing.T) {
	item := Item{
		"a": AttributeValue{"N": "1"},
		"b": AttributeValue{"N": "2"},
	}
	values := map[string]AttributeValue{
		":one": {"N": "1"},
		":two": {"N": "2"},
	}
	ce := CompileCondition("(a = :one) AND (b = :two)", nil, values)
	if !ce.Evaluate(item) {
		t.Fatal("expected parenthesized AND to match")
	}
}

// BenchmarkExpr_Evaluate benchmarks the hot path of evaluating a pre-compiled
// condition expression. Target: <50ns/op.
func BenchmarkExpr_Evaluate(b *testing.B) {
	item := Item{
		"name":   AttributeValue{"S": "alice"},
		"age":    AttributeValue{"N": "30"},
		"status": AttributeValue{"S": "active"},
	}
	values := map[string]AttributeValue{
		":name": {"S": "alice"},
	}
	ce := CompileCondition("name = :name", nil, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ce.Evaluate(item)
	}
}

func BenchmarkExpr_Evaluate_AND(b *testing.B) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
		"age":  AttributeValue{"N": "30"},
	}
	values := map[string]AttributeValue{
		":name": {"S": "alice"},
		":age":  {"N": "25"},
	}
	ce := CompileCondition("name = :name AND age > :age", nil, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ce.Evaluate(item)
	}
}

func BenchmarkExpr_OldPath(b *testing.B) {
	item := Item{
		"name": AttributeValue{"S": "alice"},
	}
	values := map[string]AttributeValue{
		":name": {"S": "alice"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		evaluateCondition("name = :name", item, nil, values)
	}
}
