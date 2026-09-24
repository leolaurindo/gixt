package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddMineRejectsAlias(t *testing.T) {
	add := newAddCmd()
	var mine *cobra.Command
	for _, command := range add.Commands() {
		if command.Name() == "mine" {
			mine = command
			break
		}
	}
	if mine == nil {
		t.Fatal("add mine subcommand not registered")
	}
	if err := mine.Flags().Set("as", "mine-alias"); err != nil {
		t.Fatal(err)
	}
	if err := mine.RunE(mine, nil); err == nil || !strings.Contains(err.Error(), "--as") {
		t.Fatalf("expected --as rejection, got %v", err)
	}
}

func TestAddMineIsAnExactSubcommand(t *testing.T) {
	add := newAddCmd()
	for _, command := range add.Commands() {
		if command.Name() == "mine" {
			return
		}
	}
	t.Fatal("add mine subcommand not registered")
}
