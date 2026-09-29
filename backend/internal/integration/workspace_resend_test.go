//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inboxes/backend/internal/store"
)

func TestWorkspaceResendIsolation(t *testing.T) {
	ctx := context.Background()
	a, ua := seedOrg(t, "Resend A", "resend-a@test.io", "Password1")
	b, ub := seedOrg(t, "Resend B", "resend-b@test.io", "Password1")
	t.Cleanup(func() { cleanupOrg(t, a); cleanupOrg(t, b) })
	da, err := testStore.UpsertDomain(ctx, a, "shared-workspace.test", "provider-domain", "active", json.RawMessage("[]"), 0)
	if err != nil {
		t.Fatal(err)
	}
	db, err := testStore.UpsertDomain(ctx, b, "shared-workspace.test", "provider-domain", "pending", json.RawMessage("[]"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if da == db {
		t.Fatal("onboarding returned another workspace's domain")
	}
	if err := testStore.SyncDomains(ctx, b, []store.ResendDomainInfo{{ID: "provider-domain", Name: "shared-workspace.test", Status: "pending"}}); err != nil {
		t.Fatal(err)
	}
	if status, _ := testStore.GetDomainStatus(ctx, da, a); status != "active" {
		t.Fatalf("workspace B changed A's domain: %s", status)
	}
	shared, err := testStore.IsResendDomainShared(ctx, a, "provider-domain")
	if err != nil || !shared {
		t.Fatalf("expected shared provider domain: %v, %v", shared, err)
	}
	ta := seedThread(t, a, ua, da, "A")
	ea := seedEmail(t, a, ua, da, ta, "outbound", "from@shared-workspace.test", "A")
	if _, err := testPool.Exec(ctx, "UPDATE emails SET resend_email_id=$1 WHERE id=$2", "shared-provider-email", ea); err != nil {
		t.Fatal(err)
	}
	exists, err := testStore.EmailExistsByResendID(ctx, b, "shared-provider-email")
	if err != nil || exists {
		t.Fatalf("A's email suppressed B's import: %v, %v", exists, err)
	}
	tb := seedThread(t, b, ub, db, "B")
	eb := seedEmail(t, b, ub, db, tb, "outbound", "from@shared-workspace.test", "B")
	if _, err := testPool.Exec(ctx, "UPDATE emails SET resend_email_id=$1 WHERE id=$2", "shared-provider-email", eb); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.CreateFetchJob(ctx, a, "shared-job-email", "fetch"); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.CreateFetchJob(ctx, b, "shared-job-email", "fetch"); err != nil {
		t.Fatalf("A's pending job suppressed B's job: %v", err)
	}
	if _, err := testStore.CreateFetchJob(ctx, b, "shared-job-email", "fetch"); err == nil {
		t.Fatal("duplicate job in one workspace was accepted")
	}
	if _, err := testStore.UpdateEmailStatus(ctx, a, "shared-provider-email", "bounced"); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := testPool.QueryRow(ctx, "SELECT status FROM emails WHERE id=$1", eb).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" {
		t.Fatalf("A's webhook changed B's email: %s", status)
	}
	if err := testStore.WithTx(ctx, func(tx store.Store) error { return tx.SyncDomains(ctx, b, nil) }); err != nil {
		t.Fatalf("transactional disconnect failed: %v", err)
	}
	if status, _ := testStore.GetDomainStatus(ctx, db, b); status != "disconnected" {
		t.Fatalf("missing provider domain was not disconnected: %s", status)
	}
	if _, err := testStore.SoftDeleteDomain(ctx, db, b); err != nil {
		t.Fatal(err)
	}
	domains, err := testStore.ListDomains(ctx, b, true)
	if err != nil || len(domains) != 0 {
		t.Fatalf("deleted domain remains listed: %v, %v", domains, err)
	}
	shared, err = testStore.IsResendDomainShared(ctx, a, "provider-domain")
	if err != nil || shared {
		t.Fatalf("deleted domain still blocks provider deletion: %v, %v", shared, err)
	}
}

func TestWorkspaceFounderPermissions(t *testing.T) {
	ctx := context.Background()
	a, ua := seedOrg(t, "Founder A", "founder-workspaces@test.io", "Password1")
	t.Cleanup(func() { cleanupOrg(t, a) })
	var b, ub string
	if err := testStore.WithTx(ctx, func(tx store.Store) error {
		var err error
		b, ub, err = tx.CreateWorkspaceForUser(ctx, ua, "Founder B")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupOrg(t, b) })
	owner, err := testStore.IsOrgOwner(ctx, ub, b)
	if err != nil || !owner {
		t.Fatalf("creator cannot manage own workspace: %v, %v", owner, err)
	}
	var instanceOwner bool
	if err := testPool.QueryRow(ctx, "SELECT is_owner FROM users WHERE id=$1", ub).Scan(&instanceOwner); err != nil {
		t.Fatal(err)
	}
	if instanceOwner {
		t.Fatal("new workspace founder received instance privileges")
	}
	var c, uc string
	if err := testStore.WithTx(ctx, func(tx store.Store) error {
		var err error
		c, uc, err = tx.CreateOrgForAccount(ctx, "Founder C", "founder-workspaces@test.io")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupOrg(t, c) })
	owner, err = testStore.IsOrgOwner(ctx, uc, c)
	if err != nil || !owner {
		t.Fatalf("signup founder cannot manage own workspace: %v, %v", owner, err)
	}
	owner, _ = testStore.IsOrgOwner(ctx, uc, a)
	if owner {
		t.Fatal("founder can delete another workspace")
	}
}
