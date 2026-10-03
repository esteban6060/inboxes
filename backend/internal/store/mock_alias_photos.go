package store

import "context"

func (m *MockStore) CanManageAliasPhoto(ctx context.Context, aliasID, orgID, userID, role string) (bool, error) {
	if m.CanManageAliasPhotoFn != nil {
		return m.CanManageAliasPhotoFn(ctx, aliasID, orgID, userID, role)
	}
	return false, nil
}

func (m *MockStore) SetAliasPhoto(ctx context.Context, aliasID, orgID, userID, role string, data []byte, version *string) (int64, error) {
	if m.SetAliasPhotoFn != nil {
		return m.SetAliasPhotoFn(ctx, aliasID, orgID, userID, role, data, version)
	}
	return 0, nil
}

func (m *MockStore) GetPublicAliasPhoto(ctx context.Context, aliasID, version string) ([]byte, error) {
	if m.GetPublicAliasPhotoFn != nil {
		return m.GetPublicAliasPhotoFn(ctx, aliasID, version)
	}
	return nil, nil
}

func (m *MockStore) GetSenderAliasPhoto(ctx context.Context, orgID, address string) (*AliasPhoto, error) {
	if m.GetSenderAliasPhotoFn != nil {
		return m.GetSenderAliasPhotoFn(ctx, orgID, address)
	}
	return nil, nil
}
