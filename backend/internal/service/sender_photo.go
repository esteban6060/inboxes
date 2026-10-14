package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"strings"

	"github.com/inboxes/backend/internal/store"
)

// AddSenderPhoto snapshots the image into the queued payload, so replacing or
// removing a profile photo later does not change mail already sent/scheduled.
// The stored message uses the public URL for the app's sent-message preview;
// recipients receive an inline MIME image through Resend's content_id support.
func AddSenderPhoto(ctx context.Context, st store.AliasStore, orgID, address, appURL, bodyHTML, bodyPlain string, payload map[string]interface{}) (string, error) {
	photo, err := st.GetSenderAliasPhoto(ctx, orgID, address)
	if err != nil {
		return bodyHTML, err
	}
	if photo == nil || len(photo.Data) == 0 {
		return bodyHTML, nil
	}
	if bodyHTML == "" {
		bodyHTML = "<div style=\"white-space:pre-wrap\">" + html.EscapeString(bodyPlain) + "</div>"
	}
	contentID := "alias-profile-" + photo.Version
	footer := func(src string) string {
		return fmt.Sprintf(`<table role="presentation" cellpadding="0" cellspacing="0" style="margin-top:20px;border-top:1px solid #e5e7eb;padding-top:12px;font-family:Arial,sans-serif"><tr><td style="padding-right:12px"><img src="%s" alt="%s" width="56" height="56" style="display:block;border-radius:50%%;object-fit:cover"></td><td style="font-size:13px;color:#374151"><strong>%s</strong><br>%s</td></tr></table>`,
			html.EscapeString(src), html.EscapeString("Photo of "+address), html.EscapeString(photo.Name), html.EscapeString(address))
	}
	sendHTML, _ := payload["html"].(string)
	if sendHTML == "" {
		sendHTML = bodyHTML
	}
	payload["html"] = insertPhotoFooter(sendHTML, footer("cid:"+contentID))
	attachments, _ := payload["attachments"].([]map[string]string)
	payload["attachments"] = append(attachments, map[string]string{
		"content":  base64.StdEncoding.EncodeToString(photo.Data),
		"filename": "profile-photo.jpg", "content_type": "image/jpeg", "content_id": contentID,
	})
	return insertPhotoFooter(bodyHTML, footer(strings.TrimRight(appURL, "/")+photo.PublicPath())), nil
}

func insertPhotoFooter(body, footer string) string {
	if i := strings.LastIndex(strings.ToLower(body), "</body>"); i >= 0 {
		return body[:i] + footer + body[i:]
	}
	if i := strings.LastIndex(strings.ToLower(body), "</html>"); i >= 0 {
		return body[:i] + footer + body[i:]
	}
	return body + footer
}
