package agent

import (
	"testing"

	"github.com/ktsoator/or/llm"
)

func TestUserMessageBuildsTextAndImages(t *testing.T) {
	message := UserMessage("look", llm.ImageContent{Data: "abc", MIMEType: "image/png"})

	wrapped, ok := message.(llmMessage)
	if !ok {
		t.Fatalf("message is %T, want llmMessage", message)
	}
	user, ok := wrapped.Message.(*llm.UserMessage)
	if !ok {
		t.Fatalf("wraps %T, want *llm.UserMessage", wrapped.Message)
	}
	if len(user.Content) != 2 {
		t.Fatalf("content blocks = %d, want 2 (text, image)", len(user.Content))
	}
	if text, ok := user.Content[0].(*llm.TextContent); !ok || text.Text != "look" {
		t.Fatalf("content[0] = %#v, want text %q", user.Content[0], "look")
	}
	image, ok := user.Content[1].(*llm.ImageContent)
	if !ok {
		t.Fatalf("content[1] = %T, want *llm.ImageContent", user.Content[1])
	}
	if image.Data != "abc" || image.MIMEType != "image/png" {
		t.Fatalf("image = %+v, want {abc image/png}", image)
	}
}

func TestUserMessageImagesDoNotAlias(t *testing.T) {
	message := UserMessage("two",
		llm.ImageContent{Data: "a", MIMEType: "image/png"},
		llm.ImageContent{Data: "b", MIMEType: "image/png"},
	)
	user := message.(llmMessage).Message.(*llm.UserMessage)

	first := user.Content[1].(*llm.ImageContent)
	second := user.Content[2].(*llm.ImageContent)
	if first.Data != "a" || second.Data != "b" {
		t.Fatalf("images aliased: got %q and %q, want a and b", first.Data, second.Data)
	}
}
