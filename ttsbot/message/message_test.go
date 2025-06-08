package message

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestReplaceUserMentions(t *testing.T) {

	type testCase struct {
		name     string
		content  string
		mentions map[snowflake.ID]string

		expected string
	}

	testCases := []testCase{
		{
			name:    "Single mention",
			content: "Hello <@123456>",
			mentions: map[snowflake.ID]string{
				123456: "User1",
			},
			expected: "Hello @User1",
		},
		{
			name:    "Multiple mentions",
			content: "Hello <@123456> and <@789012>",
			mentions: map[snowflake.ID]string{
				123456: "User1",
				789012: "User2",
			},
			expected: "Hello @User1 and @User2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ReplaceUserMentions(tc.content, tc.mentions)
			if result != tc.expected {
				t.Errorf("ReplaceUserMentions(%q, %v) = %q, want %q", tc.content, tc.mentions, result, tc.expected)
			}
		})
	}
}

func TestConvertMarkdownToPlainText(t *testing.T) {
	// No test because this function will be changed near future.
	t.Skip("ConvertMarkdownToPlainText is not implemented yet")
}

func TestReplaceUrlsWithPlaceholders(t *testing.T) {
	type testCase struct {
		name     string
		content  string
		expected string
	}

	testCases := []testCase{
		{
			name:     "Single URL",
			content:  "Check this link: https://example.com",
			expected: "Check this link: [URL]",
		},
		{
			name:     "Multiple URLs",
			content:  "Links: https://example.com and http://test.com",
			expected: "Links: [URL] and [URL]",
		},
		{
			name:     "Complex urls with parameters",
			content:  "Visit https://example.com/path?query=123&other=456 for more info.",
			expected: "Visit [URL] for more info.",
		},
		{
			name:     "No URLs",
			content:  "No links here.",
			expected: "No links here.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ReplaceUrlsWithPlaceholders(tc.content)
			if result != tc.expected {
				t.Errorf("ReplaceUrlsWithPlaceholders(%q) = %q, want %q", tc.content, result, tc.expected)
			}
		})
	}
}

func TestLimitContentLength(t *testing.T) {
	type testCase struct {
		name     string
		content  string
		maxLen   int
		expected string
	}

	testCases := []testCase{
		{
			name:     "Short content",
			content:  "Hello, world!",
			maxLen:   20,
			expected: "Hello, world!",
		},
		{
			name:     "Exact length",
			content:  "1234567890",
			maxLen:   10,
			expected: "1234567890",
		},
		{
			name:     "Exceeds length",
			content:  "abcdefghijklmnopqrstuvwxyz",
			maxLen:   10,
			expected: "abcdefghij",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := LimitContentLength(tc.content, tc.maxLen)
			if result != tc.expected {
				t.Errorf("LimitContentLength(%q, %d) = %q, want %q", tc.content, tc.maxLen, result, tc.expected)
			}
		})
	}
}
