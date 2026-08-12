package tuitui

import "testing"

func TestResolveTargetNormalizesAndDeduplicates(t *testing.T) {
	t.Parallel()
	target, err := resolveTarget(ToTarget{Accounts: []string{" alice ", "alice", "bob"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(target.accounts) != 2 || target.accounts[0] != "alice" || target.accounts[1] != "bob" {
		t.Fatalf("unexpected accounts: %#v", target.accounts)
	}
}

func TestResolveTargetRejectsMixedKinds(t *testing.T) {
	t.Parallel()
	if _, err := resolveTarget(ToTarget{Accounts: []string{"alice"}, GroupID: "group"}); err == nil {
		t.Fatal("expected mixed target error")
	}
}
