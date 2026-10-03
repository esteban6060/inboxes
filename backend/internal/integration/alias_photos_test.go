//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestAliasPhotoPersistencePermissionsAndRevocation(t *testing.T) {
	ctx := context.Background()
	org, admin := seedOrg(t, "photo-org", "photo-admin@test.io", "Password1")
	otherOrg, otherAdmin := seedOrg(t, "other-photo-org", "other-photo-admin@test.io", "Password1")
	t.Cleanup(func() { cleanupOrg(t, org); cleanupOrg(t, otherOrg) })
	domain := seedDomain(t, org, "photo.test")
	alias := seedAlias(t, org, domain, "hello@photo.test", "Hello")
	secondAlias := seedAlias(t, org, domain, "support@photo.test", "Support")
	member := seedUser(t, org, "photo-member@test.io", "member")
	if err := testStore.AddAliasUser(ctx, alias, org, member, false); err != nil {
		t.Fatal(err)
	}
	version, data := uuid.NewString(), []byte("stored photo bytes")
	for _, tc := range []struct {
		org, user, role string
		allowed         bool
	}{
		{org, admin, "admin", true}, {otherOrg, otherAdmin, "admin", false},
		{org, member, "member", false},
	} {
		allowed, err := testStore.CanManageAliasPhoto(ctx, alias, tc.org, tc.user, tc.role)
		if err != nil || allowed != tc.allowed {
			t.Fatalf("permission got %v, %v", allowed, err)
		}
		if !tc.allowed {
			n, err := testStore.SetAliasPhoto(ctx, alias, tc.org, tc.user, tc.role, data, &version)
			if err != nil || n != 0 {
				t.Fatalf("unauthorized write: %d, %v", n, err)
			}
		}
	}
	if err := testStore.AddAliasUser(ctx, alias, org, member, true); err != nil {
		t.Fatal(err)
	}
	n, err := testStore.SetAliasPhoto(ctx, alias, org, member, "member", data, &version)
	if err != nil || n != 1 {
		t.Fatalf("save photo: %d, %v", n, err)
	}
	got, err := testStore.GetPublicAliasPhoto(ctx, alias, version)
	if err != nil || !bytes.Equal(data, got) {
		t.Fatalf("public photo: %q, %v", got, err)
	}
	photo, err := testStore.GetSenderAliasPhoto(ctx, org, "HELLO@PHOTO.TEST")
	if err != nil || photo == nil || photo.AliasID != alias || photo.Version != version || !bytes.Equal(photo.Data, data) {
		t.Fatalf("sender photo: %+v, %v", photo, err)
	}
	photo, err = testStore.GetSenderAliasPhoto(ctx, otherOrg, "hello@photo.test")
	if err != nil || photo != nil {
		t.Fatalf("cross-workspace photo leaked: %+v, %v", photo, err)
	}
	for _, list := range []func() ([]map[string]any, error){
		func() ([]map[string]any, error) { return testStore.ListAliases(ctx, org, domain) },
		func() ([]map[string]any, error) { return testStore.ListMyAliases(ctx, member, org) },
	} {
		aliases, err := list()
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range aliases {
			if a["id"] == alias && a["avatar_url"] != "/api/public/alias-photos/"+alias+"/"+version {
				t.Fatal("photo missing from alias response")
			}
			if a["id"] == secondAlias && a["avatar_url"] != "" {
				t.Fatal("second alias got the wrong photo")
			}
		}
	}
	newVersion := uuid.NewString()
	if _, err := testStore.SetAliasPhoto(ctx, alias, org, admin, "admin", data, &newVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.GetPublicAliasPhoto(ctx, alias, version); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("old photo URL still accessible")
	}
	if _, err := testStore.SetAliasPhoto(ctx, alias, org, member, "member", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.GetPublicAliasPhoto(ctx, alias, newVersion); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("removed photo still accessible")
	}
	if _, err := testStore.SetAliasPhoto(ctx, alias, org, admin, "admin", data, &version); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.DeleteAlias(ctx, alias, org); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.GetPublicAliasPhoto(ctx, alias, version); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("deleted alias photo still public")
	}
}
