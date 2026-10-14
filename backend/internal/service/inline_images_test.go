package service

import (
	"context"
	"strings"
	"testing"

	"github.com/inboxes/backend/internal/store"
)

func TestPrepareEmailAttachmentsKeepsPositionAndSeparatesFiles(t *testing.T) {
	imageID := "11111111-1111-4111-8111-111111111111"
	fileID := "22222222-2222-4222-8222-222222222222"
	body := `<p>Before</p><div data-inline-image="" style="text-align: center"><img src="/api/attachments/` + imageID + `/inline" width="320" style="width: 320px; max-width: 100%; height: auto"></div><p>After</p>`
	st := &store.MockStore{LoadAttachmentsForResendFn: func(_ context.Context, ids []string, _ string) ([]map[string]string, error) {
		return []map[string]string{
			{"id": imageID, "filename": "photo.png", "content": "aW1hZ2U=", "content_type": "image/png"},
			{"id": fileID, "filename": "report.pdf", "content": "ZmlsZQ==", "content_type": "application/pdf"},
		}, nil
	}}
	sent, attachments, err := PrepareEmailAttachments(context.Background(), st, "org", body, []string{fileID, imageID})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sent, `src="cid:inboxes-image-`+imageID+`"`) || !strings.Contains(sent, `text-align: center`) || !strings.Contains(sent, `width="320"`) || strings.Index(sent, "Before") > strings.Index(sent, "cid:") || strings.Index(sent, "cid:") > strings.Index(sent, "After") {
		t.Fatalf("image position or formatting lost: %s", sent)
	}
	if len(attachments) != 2 || attachments[0]["content_id"] != "" || attachments[1]["content_id"] != "inboxes-image-"+imageID {
		t.Fatalf("incorrect MIME attachment roles: %#v", attachments)
	}
}

func TestPrepareEmailAttachmentsRejectsNonImageInline(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	st := &store.MockStore{LoadAttachmentsForResendFn: func(_ context.Context, _ []string, _ string) ([]map[string]string, error) {
		return []map[string]string{{"id": id, "filename": "note.txt", "content": "dGV4dA==", "content_type": "text/plain"}}, nil
	}}
	_, _, err := PrepareEmailAttachments(context.Background(), st, "org", `<img src="/api/attachments/`+id+`/inline">`, nil)
	if err == nil {
		t.Fatal("expected non-image rejection")
	}
}
