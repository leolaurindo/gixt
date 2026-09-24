package cli

import "testing"

func TestRemoveOwnerRequiresFlagWithoutTarget(t *testing.T) {
	cmd := newRemoveCmd()
	if err := cmd.Flags().Set("owner", "me"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("expected owner-only removal to validate: %v", err)
	}
	if err := cmd.Args(cmd, []string{"tool"}); err == nil {
		t.Fatal("expected owner and target to be mutually exclusive")
	}
}

func TestRemoveTargetRequiresExactlyOneWithoutOwner(t *testing.T) {
	cmd := newRemoveCmd()
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("expected missing target error")
	}
	if err := cmd.Args(cmd, []string{"one", "two"}); err == nil {
		t.Fatal("expected multiple-target error")
	}
}
