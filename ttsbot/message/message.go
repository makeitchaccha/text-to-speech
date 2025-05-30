package message

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
)

func ConvertMarkdownToPlainText(content string) string {
	// TODO: Implement a more robust markdown to plain text conversion if needed.
	return content
}

func LimitLength(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func AddAttachments(content string, attachments []discord.Attachment) string {
	// If there are no attachments, return the content as is.
	if len(attachments) == 0 {
		return content
	}

	numberOfAttachments := len(attachments)
	return content + " " + fmt.Sprintf("%d attachments:", numberOfAttachments)
}
