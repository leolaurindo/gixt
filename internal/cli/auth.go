package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/gist"
)

func newAuthCmd() *cobra.Command {
	login := &cobra.Command{
		Use:   "login",
		Short: "store a GitHub personal access token",
		Args:  cobra.NoArgs,
		RunE:  authLogin,
	}
	login.Flags().String("token", "", "token to store (otherwise prompted)")

	c := &cobra.Command{
		Use:   "auth",
		Short: "manage gixt's GitHub authentication",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		login,
		&cobra.Command{Use: "status", Short: "show authentication status", Args: cobra.NoArgs, RunE: authStatus},
		&cobra.Command{Use: "logout", Short: "remove the stored token", Args: cobra.NoArgs, RunE: authLogout},
	)
	return c
}

func authLogin(cmd *cobra.Command, args []string) error {
	token := strings.TrimSpace(mustString(cmd, "token"))
	if token == "" {
		if !isTTY(os.Stdin) {
			return errors.New("no token provided; pass --token or run interactively")
		}
		fmt.Fprint(os.Stderr, "Paste a GitHub personal access token (scope: gist): ")
		line, err := readLine()
		if err != nil {
			return err
		}
		token = line
	}
	if token == "" {
		return errors.New("no token provided")
	}

	client := gist.New(token)
	user, err := client.CurrentUser(cmd.Context())
	if err != nil {
		return fmt.Errorf("token rejected: %w", err)
	}

	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	buf, err := json.MarshalIndent(map[string]string{"token": token, "user": user}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(paths.AuthFile, buf, 0o600); err != nil {
		return err
	}
	logf("logged in as %s", user)
	return nil
}

func authStatus(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	token := loadToken(paths.AuthFile)
	if token == "" {
		logf("not logged in (public gists still work; run `gixt auth login`)")
		return nil
	}
	user, err := gist.New(token).CurrentUser(cmd.Context())
	if err != nil {
		return err
	}
	logf("logged in as %s", user)
	return nil
}

func authLogout(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	if err := os.Remove(paths.AuthFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	logf("logged out")
	return nil
}
