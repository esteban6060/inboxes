package queue

import "testing"

func TestSendPayloadWithInlinePhotoCannotUseBatch(t *testing.T) {
	for _, tc := range []struct {
		payload string
		single  bool
	}{
		{`{"html":"<p>Hello</p>"}`, false},
		{`{"attachments":[]}`, false},
		{`{"attachments":[{"content_id":"alias-profile-v1","content":"image"}]}`, true},
		{`{"attachments":[{"filename":"invoice.pdf"}]}`, true},
		{`broken JSON`, true},
	} {
		if got := sendPayloadHasAttachments([]byte(tc.payload)); got != tc.single {
			t.Errorf("%s: got %v, want %v", tc.payload, got, tc.single)
		}
	}
}
