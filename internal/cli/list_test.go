package cli

import "testing"

func TestListHasNoMutatingSubcommands(t *testing.T) {
	cmd := newListCmd()
	for _, sub := range cmd.Commands() {
		if sub.Name() == "clear" || sub.Name() == "refresh" {
			t.Fatalf("list still has mutating subcommand %q", sub.Name())
		}
	}
}
