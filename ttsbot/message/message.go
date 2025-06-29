package message

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

var (
	urlRegex      = regexp.MustCompile(`https?://[^\s]+`)
	headingsRegex = regexp.MustCompile(`^ *#{1,3}`)
	emojiRegex    = regexp.MustCompile(`<a?:(\w+):\d+>`)
)

func ReplaceUserMentions(content string, mentions map[snowflake.ID]string) string {
	for id, name := range mentions {
		// Replace mentions like <@123456789012345678> with the placeholder
		content = strings.ReplaceAll(content, discord.UserMention(id), "@"+name)
	}
	return content
}

func ReplaceEmojis(content string) string {
	matches := emojiRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		emojiBlock := match[0]
		emojiName := match[1]

		// Replace emoji blocks like <a:emoji_name:123456789012345678> with emoji_name
		content = strings.ReplaceAll(content, emojiBlock, emojiName)
	}
	return content
}

func ConvertMarkdownToPlainText(content string) string {
	lines := strings.Split(content, "\n")
	lines = removeCodeBlocks(lines)
	for i, line := range lines {
		// Remove markdown formatting
		line = removeHeadings(line)
		line = replaceWithSkippingInlineCode(line, "**", "")
		line = replaceWithSkippingInlineCode(line, "__", "")
		line = replaceWithSkippingInlineCode(line, "*", "")
		line = replaceWithSkippingInlineCode(line, "_", " ")
		line = replaceWithSkippingInlineCode(line, "~~", "")
		line = strings.ReplaceAll(line, "`", "")
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func removeHeadings(line string) string {
	// Remove headings (e.g. "# Heading", "## Subheading")
	// This regex matches headings with 1 to 6 hashes at the start of the line.
	return headingsRegex.ReplaceAllString(line, "")
}

func replaceWithSkippingInlineCode(line string, replaced, replacement string) string {
	// e.g. "This is `inline code` and this is not."
	// -> ["This is ", "inline code", " and this is not."]
	parts := strings.Split(line, "`")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = strings.ReplaceAll(parts[i], replaced, replacement)
	}
	return strings.Join(parts, "`")
}

func removeCodeBlocks(lines []string) []string {
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock // Toggle code block state
			if inCodeBlock {
				// If we are entering a code block, we don't add the line to the result
				kind := strings.TrimPrefix(line, "```")
				result = append(result, fmt.Sprintf("code block: %s", kind))
			}
			continue
		}
		if !inCodeBlock {
			result = append(result, line)
		}
	}

	return result
}

func ReplaceUrlsWithPlaceholders(content string) string {
	return urlRegex.ReplaceAllString(content, "[URL]")
}

func LimitContentLength(content string, max int) string {
	runes := []rune(content)
	if len(runes) <= max {
		return content
	}
	return string(runes[:max])
}
