-- +goose Up
-- is_owner grants instance-wide system/cron access. Workspace ownership must
-- never grant those privileges to founders of additional organizations.
ALTER TABLE users ADD COLUMN is_org_owner BOOLEAN NOT NULL DEFAULT false;
UPDATE users SET is_org_owner = true
WHERE id IN (
  SELECT DISTINCT ON (org_id) id FROM users
  WHERE role = 'admin' AND account_id IS NOT NULL
  ORDER BY org_id, created_at, id
);

-- +goose Down
ALTER TABLE users DROP COLUMN is_org_owner;
