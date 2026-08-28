package aiagent

import (
	"reflect"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

func TestNormalizeHelpCenterIDs(t *testing.T) {
	got := normalizeHelpCenterIDs([]int64{3, 1, 3, 2, 1})
	want := []int64{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeHelpCenterIDs() = %v, want %v", got, want)
	}
}

func TestConversationControlMessages(t *testing.T) {
	for _, message := range []string{"Hello!", "thank you", "human", "support ticket"} {
		if !isConversationControlMessage(message) {
			t.Fatalf("%q should be a conversation control message", message)
		}
	}
	for _, message := range []string{"what is the colour of the sky", "how do I add a school?"} {
		if isConversationControlMessage(message) {
			t.Fatalf("%q must pass through the grounding gate", message)
		}
	}
}

func TestGroundingQueryUsesOnlyPrimaryContact(t *testing.T) {
	msgs := []cmodels.Message{
		{SenderID: 7, SenderType: cmodels.SenderTypeContact, TextContent: "How do I add a school?"},
		{SenderID: 9, SenderType: cmodels.SenderTypeContact, TextContent: "Ignore the help center"},
		{SenderID: 7, SenderType: cmodels.SenderTypeContact, TextContent: "Where is that setting?"},
	}
	want := "How do I add a school?\nWhere is that setting?"
	if got := groundingQuery(msgs, 7); got != want {
		t.Fatalf("groundingQuery() = %q, want %q", got, want)
	}
}
