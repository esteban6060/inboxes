-- +goose Up
-- Resend credentials can belong to the same provider account in several
-- workspaces. Domain imports and email deduplication must remain tenant-local.
DROP INDEX idx_domains_unique_active;
CREATE UNIQUE INDEX idx_domains_unique_active ON domains(org_id, domain) WHERE status NOT IN ('deleted');
DROP INDEX idx_emails_resend_id;
CREATE UNIQUE INDEX idx_emails_resend_id ON emails(org_id, resend_email_id) WHERE resend_email_id IS NOT NULL;
DROP INDEX idx_email_jobs_resend_id_pending;
CREATE UNIQUE INDEX idx_email_jobs_resend_id_pending ON email_jobs(org_id, resend_email_id) WHERE status IN ('pending', 'running');

-- +goose Down
-- These statements fail safely if multiple workspaces now share provider IDs.
DROP INDEX idx_email_jobs_resend_id_pending;
CREATE UNIQUE INDEX idx_email_jobs_resend_id_pending ON email_jobs(resend_email_id) WHERE status IN ('pending', 'running');
DROP INDEX idx_emails_resend_id;
CREATE UNIQUE INDEX idx_emails_resend_id ON emails(resend_email_id) WHERE resend_email_id IS NOT NULL;
DROP INDEX idx_domains_unique_active;
CREATE UNIQUE INDEX idx_domains_unique_active ON domains(domain) WHERE status NOT IN ('deleted');
