package normalize

import (
	"regexp"
	"strings"
)

// ExtractCodeBlocks extracts code blocks from message content
// Supports:
// - Fenced code blocks: ```code``` or ```\ncode\n```
// - Fenced code blocks: ~~~code~~~ or ~~~\ncode\n~~~
// - Inline code: `code`
// - HTML code tags: <code>code</code>
//
// Note: Language detection is intentionally omitted as it's unreliable
// and Slack doesn't use language specifiers in fenced blocks.
func ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock

	// Pattern 1: Fenced code blocks with triple backticks or triple tildes
	// Match everything between ``` or ~~~ markers
	// (?s) makes . match newlines
	fencedPattern := regexp.MustCompile("(?s)```(.*?)```|~~~(.*?)~~~")
	fencedMatches := fencedPattern.FindAllStringSubmatch(content, -1)

	for _, match := range fencedMatches {
		var code string

		if match[1] != "" {
			// Triple-backtick block
			code = match[1]
		} else {
			// Triple-tilde block
			code = match[2]
		}

		// Trim leading/trailing whitespace and newlines
		code = strings.TrimSpace(code)

		if len(code) > 0 {
			blocks = append(blocks, CodeBlock{
				Language: "fenced",
				Code:     code,
			})
		}
	}

	// Pattern 2: Inline code with single backticks
	// Only capture if it's reasonably substantial (to avoid noise)
	inlinePattern := regexp.MustCompile("`([^`\n]{1,200})`")
	inlineMatches := inlinePattern.FindAllStringSubmatch(content, -1)

	for _, match := range inlineMatches {
		code := strings.TrimSpace(match[1])
		// Only include inline code that looks code-ish
		// Skip if it's just a single word with no special characters
		if len(code) > 0 && (strings.ContainsAny(code, "(){}[]<>.:;=+-*/%&|!") || strings.Contains(code, " ")) {
			blocks = append(blocks, CodeBlock{
				Language: "inline",
				Code:     code,
			})
		}
	}

	// Pattern 3: HTML <code> tags
	htmlPattern := regexp.MustCompile("(?s)<code[^>]*>(.*?)</code>")
	htmlMatches := htmlPattern.FindAllStringSubmatch(content, -1)

	for _, match := range htmlMatches {
		code := strings.TrimSpace(match[1])
		if len(code) > 0 {
			blocks = append(blocks, CodeBlock{
				Language: "html-code",
				Code:     code,
			})
		}
	}

	return blocks
}

// ExtractURLs extracts URLs from message content
// Matches http://, https://, and common URL patterns
func ExtractURLs(content string) []string {
	// Pattern for URLs - matches http(s):// URLs
	// Also matches <url> format that Slack uses
	pattern := regexp.MustCompile(`<?(https?://[^\s<>]+)>?`)

	matches := pattern.FindAllStringSubmatch(content, -1)

	urls := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		url := match[1]
		// Deduplicate URLs
		if !seen[url] {
			urls = append(urls, url)
			seen[url] = true
		}
	}

	return urls
}
