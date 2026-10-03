package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// AliasPhoto contains the current image. A versioned public URL stops browsers
// from displaying the previous image after a replacement.
type AliasPhoto struct {
	AliasID string
	Version string
	Name    string
	Data    []byte
}

func (p *AliasPhoto) PublicPath() string {
	return "/api/public/alias-photos/" + p.AliasID + "/" + p.Version
}

func (s *PgStore) CanManageAliasPhoto(ctx context.Context, aliasID, orgID, userID, role string) (bool, error) {
	var allowed bool
	err := s.q.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM aliases a WHERE a.id = $1 AND a.org_id = $2 AND a.deleted_at IS NULL
		AND ($4 OR EXISTS (SELECT 1 FROM alias_users au
		WHERE au.alias_id = a.id AND au.user_id = $3 AND au.can_send_as)))`,
		aliasID, orgID, userID, role == "admin").Scan(&allowed)
	return allowed, err
}

func (s *PgStore) SetAliasPhoto(ctx context.Context, aliasID, orgID, userID, role string, data []byte, version *string) (int64, error) {
	tag, err := s.q.Exec(ctx, `UPDATE aliases a SET photo_data = $5, photo_version = $6
		WHERE a.id = $1 AND a.org_id = $2 AND a.deleted_at IS NULL
		AND ($4 OR EXISTS (SELECT 1 FROM alias_users au
		WHERE au.alias_id = a.id AND au.user_id = $3 AND au.can_send_as))`,
		aliasID, orgID, userID, role == "admin", data, version)
	return tag.RowsAffected(), err
}

func (s *PgStore) GetPublicAliasPhoto(ctx context.Context, aliasID, version string) ([]byte, error) {
	var data []byte
	err := s.q.QueryRow(ctx, `SELECT a.photo_data FROM aliases a
		JOIN domains d ON d.id = a.domain_id JOIN orgs o ON o.id = a.org_id
		WHERE a.id = $1 AND a.photo_version = $2 AND a.photo_data IS NOT NULL
		AND a.deleted_at IS NULL AND d.status != 'deleted' AND o.deleted_at IS NULL`,
		aliasID, version).Scan(&data)
	return data, err
}

func (s *PgStore) GetSenderAliasPhoto(ctx context.Context, orgID, address string) (*AliasPhoto, error) {
	p := &AliasPhoto{}
	err := s.q.QueryRow(ctx, `SELECT id, photo_version, name, photo_data FROM aliases
		WHERE org_id = $1 AND lower(address) = lower($2) AND deleted_at IS NULL
		AND photo_data IS NOT NULL`, orgID, address).Scan(&p.AliasID, &p.Version, &p.Name, &p.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}
