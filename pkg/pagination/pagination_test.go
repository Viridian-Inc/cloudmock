package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeItems(n int) []int {
	items := make([]int, n)
	for i := range items {
		items[i] = i + 1
	}
	return items
}

func TestPaginate_FirstPage(t *testing.T) {
	items := makeItems(100)
	page := Paginate(items, "", 10, 20)

	require.Len(t, page.Items, 10)
	assert.Equal(t, 1, page.Items[0])
	assert.Equal(t, 10, page.Items[9])
	assert.True(t, page.HasMore)
	assert.NotEmpty(t, page.NextToken)
}

func TestPaginate_SecondPage(t *testing.T) {
	items := makeItems(100)
	first := Paginate(items, "", 10, 20)
	require.NotEmpty(t, first.NextToken)

	second := Paginate(items, first.NextToken, 10, 20)
	require.Len(t, second.Items, 10)
	assert.Equal(t, 11, second.Items[0])
	assert.Equal(t, 20, second.Items[9])
	assert.True(t, second.HasMore)
	assert.NotEmpty(t, second.NextToken)
}

func TestPaginate_LastPage(t *testing.T) {
	items := makeItems(25)

	// Walk through all pages until the last.
	token := ""
	var lastPage Page[int]
	for {
		p := Paginate(items, token, 10, 20)
		lastPage = p
		if !p.HasMore {
			break
		}
		token = p.NextToken
	}

	assert.False(t, lastPage.HasMore)
	assert.Empty(t, lastPage.NextToken)
	// Last page should contain items 21-25 (5 items).
	require.Len(t, lastPage.Items, 5)
	assert.Equal(t, 21, lastPage.Items[0])
	assert.Equal(t, 25, lastPage.Items[4])
}

func TestPaginate_EmptySlice(t *testing.T) {
	page := Paginate([]int{}, "", 10, 20)

	assert.Empty(t, page.Items)
	assert.False(t, page.HasMore)
	assert.Empty(t, page.NextToken)
}

func TestPaginate_DefaultMax(t *testing.T) {
	items := makeItems(100)
	// maxResults=0 should use defaultMax=15.
	page := Paginate(items, "", 0, 15)

	require.Len(t, page.Items, 15)
	assert.True(t, page.HasMore)
}

func TestPaginate_WithFilter(t *testing.T) {
	items := makeItems(20) // 1..20

	// Keep only even numbers (10 total).
	even := func(n int) bool { return n%2 == 0 }
	page := PaginateWithFilter(items, "", 5, 10, even)

	require.Len(t, page.Items, 5)
	for _, v := range page.Items {
		assert.Equal(t, 0, v%2, "expected even number, got %d", v)
	}
	assert.True(t, page.HasMore)
	assert.NotEmpty(t, page.NextToken)

	// Second page of evens.
	page2 := PaginateWithFilter(items, page.NextToken, 5, 10, even)
	require.Len(t, page2.Items, 5)
	assert.False(t, page2.HasMore)
	assert.Empty(t, page2.NextToken)
}

func TestPaginate_TokenResilience(t *testing.T) {
	items := makeItems(50)

	// Garbage token should return first page, not crash.
	page := Paginate(items, "!!not-valid-base64!!", 10, 20)

	require.Len(t, page.Items, 10)
	assert.Equal(t, 1, page.Items[0])
	assert.True(t, page.HasMore)
}
