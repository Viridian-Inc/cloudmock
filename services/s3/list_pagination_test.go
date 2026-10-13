package s3_test

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

type paginationEntry struct {
	key    string
	prefix bool
}

type paginationResponse struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	Name                  string   `xml:"Name"`
	Prefix                string   `xml:"Prefix"`
	Delimiter             string   `xml:"Delimiter"`
	MaxKeys               int      `xml:"MaxKeys"`
	KeyCount              int      `xml:"KeyCount"`
	IsTruncated           bool     `xml:"IsTruncated"`
	ContinuationToken     string   `xml:"ContinuationToken"`
	NextContinuationToken string   `xml:"NextContinuationToken"`
	Contents              []struct {
		Key string `xml:"Key"`
	} `xml:"Contents"`
	CommonPrefixes []struct {
		Prefix string `xml:"Prefix"`
	} `xml:"CommonPrefixes"`
}

func TestS3_ListObjectsV2_PaginationTraversal(t *testing.T) {
	cases := []struct {
		name      string
		keys      []string
		prefix    string
		delimiter string
		want      []paginationEntry
	}{
		{
			name: "flat-three",
			keys: []string{"c", "a", "b"},
			want: []paginationEntry{{key: "a"}, {key: "b"}, {key: "c"}},
		},
		{
			name: "flat-four-exact-multiple",
			keys: []string{"d", "b", "a", "c"},
			want: []paginationEntry{{key: "a"}, {key: "b"}, {key: "c"}, {key: "d"}},
		},
		{
			name: "flat-five",
			keys: []string{"e", "d", "b", "a", "c"},
			want: []paginationEntry{{key: "a"}, {key: "b"}, {key: "c"}, {key: "d"}, {key: "e"}},
		},
		{
			name: "query-and-xml-characters",
			keys: []string{"g\"quote'", "f space", "e#hash", "d%percent", "c?<xml>", "b&and", "a+plus"},
			want: []paginationEntry{{key: "a+plus"}, {key: "b&and"}, {key: "c?<xml>"}, {key: "d%percent"}, {key: "e#hash"}, {key: "f space"}, {key: "g\"quote'"}},
		},
		{
			name: "mixed-prefix-and-delimiter",
			keys: []string{
				"zzz", "aaa", "docs&y/outsider", "docs&x",
				"docs&x/c%group/two", "docs&x/c%group/one", "docs&x/b.txt",
				"docs&x/a group/two", "docs&x/a group/one", "docs&x/a group/", "docs&x/00.txt", "docs&x/",
			},
			prefix: "docs&x/", delimiter: "/",
			want: []paginationEntry{
				{key: "docs&x/"}, {key: "docs&x/00.txt"}, {key: "docs&x/a group/", prefix: true},
				{key: "docs&x/b.txt"}, {key: "docs&x/c%group/", prefix: true},
			},
		},
		{
			name:   "group-only-exact-multiple",
			keys:   []string{"reports/2025/b", "reports/2024/b", "reports/2025/a", "reports/2024/a"},
			prefix: "reports/", delimiter: "/",
			want: []paginationEntry{{key: "reports/2024/", prefix: true}, {key: "reports/2025/", prefix: true}},
		},
		{
			name:      "single-group-final-page",
			keys:      []string{"folder/third", "folder/second", "folder/first"},
			delimiter: "/",
			want:      []paginationEntry{{key: "folder/", prefix: true}},
		},
		{
			name:   "prefix-without-delimiter",
			keys:   []string{"zzz", "logs-old", "logs/b", "logs/a", "aaa"},
			prefix: "logs/",
			want:   []paginationEntry{{key: "logs/a"}, {key: "logs/b"}},
		},
		{
			name: "prefix-has-no-matches",
			keys: []string{"a", "z"}, prefix: "missing/", delimiter: "/",
		},
	}

	for _, tc := range cases {
		for _, pageSize := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/max-keys-%d", tc.name, pageSize), func(t *testing.T) {
				handler := newS3Gateway(t)
				const bucket = "pagination-regression"
				mustCreateBucket(t, handler, bucket)
				for _, key := range tc.keys {
					// The helper constructs a URL, so encode key characters before
					// sending them through the same HTTP path as a real client.
					mustPutObject(t, handler, bucket, url.PathEscape(key), "text/plain", key)
				}

				var collected []paginationEntry
				seenEntries := make(map[paginationEntry]bool)
				seenTokens := make(map[string]bool)
				token := ""
				for pageIndex := 0; pageIndex <= len(tc.want)+1; pageIndex++ {
					query := url.Values{
						"list-type": {"2"}, "max-keys": {strconv.Itoa(pageSize)},
						"prefix": {tc.prefix}, "delimiter": {tc.delimiter},
					}
					if token != "" {
						query.Set("continuation-token", token)
					}
					w := httptest.NewRecorder()
					handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"+bucket+"?"+query.Encode()))
					if w.Code != http.StatusOK {
						t.Fatalf("page %d: expected 200, got %d: %s", pageIndex, w.Code, w.Body.String())
					}
					var page paginationResponse
					if err := xml.Unmarshal(w.Body.Bytes(), &page); err != nil {
						t.Fatalf("page %d: invalid XML: %v: %s", pageIndex, err, w.Body.String())
					}
					if page.Name != bucket || page.Prefix != tc.prefix || page.Delimiter != tc.delimiter || page.ContinuationToken != token || page.MaxKeys != pageSize {
						t.Fatalf("page %d did not echo request parameters: %+v", pageIndex, page)
					}
					count := len(page.Contents) + len(page.CommonPrefixes)
					if count > pageSize || page.KeyCount != count {
						t.Fatalf("page %d: max-keys=%d, KeyCount=%d, contents+prefixes=%d", pageIndex, pageSize, page.KeyCount, count)
					}
					var entries []paginationEntry
					for i, object := range page.Contents {
						if i > 0 && page.Contents[i-1].Key >= object.Key {
							t.Fatalf("page %d Contents are not in unique lexicographic order: %+v", pageIndex, page.Contents)
						}
						entries = append(entries, paginationEntry{key: object.Key})
					}
					for i, group := range page.CommonPrefixes {
						if i > 0 && page.CommonPrefixes[i-1].Prefix >= group.Prefix {
							t.Fatalf("page %d CommonPrefixes are not in unique lexicographic order: %+v", pageIndex, page.CommonPrefixes)
						}
						entries = append(entries, paginationEntry{key: group.Prefix, prefix: true})
					}
					// XML separates Contents and CommonPrefixes; compare the logical
					// lexicographic page order across both kinds of result.
					sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
					for _, entry := range entries {
						if seenEntries[entry] {
							t.Fatalf("page %d repeated result %+v; prior results=%+v", pageIndex, entry, collected)
						}
						seenEntries[entry] = true
						collected = append(collected, entry)
					}
					if !page.IsTruncated {
						if page.NextContinuationToken != "" {
							t.Errorf("final page must not contain NextContinuationToken, got %q", page.NextContinuationToken)
						}
						if !reflect.DeepEqual(collected, tc.want) {
							t.Fatalf("complete traversal: got %+v, want %+v", collected, tc.want)
						}
						return
					}
					if len(collected) >= len(tc.want) {
						t.Fatalf("page %d returned every expected result but is still truncated: %+v", pageIndex, page)
					}
					if count == 0 || page.NextContinuationToken == "" || page.NextContinuationToken == token || seenTokens[page.NextContinuationToken] {
						t.Fatalf("page %d made no progress: %+v", pageIndex, page)
					}
					seenTokens[page.NextContinuationToken] = true
					token = page.NextContinuationToken
				}
				t.Fatalf("pagination did not finish within %d pages", len(tc.want)+2)
			})
		}
	}
}
