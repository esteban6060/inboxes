package service

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/inboxes/backend/internal/store"
)

func TestSenderPhotoInlineSnapshotPreservesAttachments(t *testing.T) {
	photo := &store.AliasPhoto{AliasID: "a1", Version: "v1", Name: `<script>alert(1)</script>`, Data: []byte("photo snapshot")}
	st := &store.MockStore{GetSenderAliasPhotoFn: func(ctx context.Context, org, address string) (*store.AliasPhoto, error) {
		if org != "o1" || address != "hello@example.com" {
			t.Fatal("wrong sender scope")
		}
		return photo, nil
	}}
	payload := map[string]interface{}{
		"attachments": []map[string]string{{"filename": "invoice.pdf", "content": "existing"}},
		"headers":     map[string]string{"In-Reply-To": "<reply>"},
	}
	stored, err := AddSenderPhoto(context.Background(), st, "o1", "hello@example.com", "https://mail.example.com/", "<html><body><p>Hello</p></body></html>", "Hello", payload)
	if err != nil {
		t.Fatal(err)
	}
	sent := payload["html"].(string)
	if !strings.Contains(sent, `src="cid:alias-profile-v1"`) || strings.Index(sent, "<table") > strings.Index(sent, "</body>") {
		t.Fatal("inline footer is misplaced")
	}
	if strings.Contains(sent, "<script>") || !strings.Contains(sent, "&lt;script&gt;") {
		t.Fatal("sender name was not escaped")
	}
	if !strings.Contains(stored, "https://mail.example.com/api/public/alias-photos/a1/v1") || strings.Contains(stored, "cid:") {
		t.Fatal("stored preview does not use public photo")
	}
	atts := payload["attachments"].([]map[string]string)
	if len(atts) != 2 || atts[0]["filename"] != "invoice.pdf" || atts[1]["content"] != base64.StdEncoding.EncodeToString(photo.Data) {
		t.Fatal("attachments were lost or snapshot is wrong")
	}
	if payload["headers"].(map[string]string)["In-Reply-To"] != "<reply>" {
		t.Fatal("reply headers were lost")
	}
}

func TestSenderPhotoPlainTextAndNoPhoto(t *testing.T) {
	st := &store.MockStore{}
	payload := map[string]interface{}{"text": "Hello"}
	stored, err := AddSenderPhoto(context.Background(), st, "o1", "hi@example.com", "", "", "Hello", payload)
	if err != nil || stored != "" || len(payload) != 1 {
		t.Fatal("no-photo sender changed")
	}
	st.GetSenderAliasPhotoFn = func(context.Context, string, string) (*store.AliasPhoto, error) {
		return &store.AliasPhoto{AliasID: "a1", Version: "v1", Data: []byte("photo")}, nil
	}
	_, err = AddSenderPhoto(context.Background(), st, "o1", "hi@example.com", "", "", "Hello <world>\nline 2", payload)
	if err != nil || !strings.Contains(payload["html"].(string), "Hello &lt;world&gt;\nline 2") {
		t.Fatal("plain text did not survive HTML conversion")
	}
}

func TestSenderPhotoKeepsComposerCIDImages(t *testing.T) {
	st := &store.MockStore{GetSenderAliasPhotoFn: func(context.Context, string, string) (*store.AliasPhoto, error) {
		return &store.AliasPhoto{AliasID: "a1", Version: "v1", Data: []byte("photo")}, nil
	}}
	storedBody := `<p>Before</p><img src="/api/attachments/image-id/inline"><p>After</p>`
	payload := map[string]interface{}{"html": `<p>Before</p><img src="cid:inboxes-image-image-id"><p>After</p>`}
	stored, err := AddSenderPhoto(context.Background(), st, "org", "hello@example.com", "https://mail.example.com", storedBody, "", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload["html"].(string), `src="cid:inboxes-image-image-id"`) || !strings.Contains(stored, `/api/attachments/image-id/inline`) {
		t.Fatal("composer image source was replaced by the sender photo step")
	}
}
