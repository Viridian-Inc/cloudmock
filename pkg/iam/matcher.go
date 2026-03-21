package iam

// WildcardMatch reports whether value matches pattern.
// Supported wildcards:
//   - '*' matches any sequence of characters (including empty)
//   - '?' matches any single character
func WildcardMatch(pattern, value string) bool {
	p := []rune(pattern)
	v := []rune(value)
	pLen := len(p)
	vLen := len(v)

	// dp[i][j] = true if p[:i] matches v[:j]
	dp := make([][]bool, pLen+1)
	for i := range dp {
		dp[i] = make([]bool, vLen+1)
	}

	// Empty pattern matches empty value.
	dp[0][0] = true

	// A pattern of all '*' can match empty string.
	for i := 1; i <= pLen; i++ {
		if p[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		}
	}

	for i := 1; i <= pLen; i++ {
		for j := 1; j <= vLen; j++ {
			switch p[i-1] {
			case '*':
				// '*' can match zero characters (dp[i-1][j]) or one more (dp[i][j-1])
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			case '?':
				// '?' matches exactly one character
				dp[i][j] = dp[i-1][j-1]
			default:
				dp[i][j] = dp[i-1][j-1] && p[i-1] == v[j-1]
			}
		}
	}

	return dp[pLen][vLen]
}
