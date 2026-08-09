package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/version"
)

const (
	clrTitle = "\033[1;36m"
	clrReset = "\033[0m"
)

var knownCommands = []string{
	"run", "add", "remove", "list", "trust", "gist", "cache", "auth", "self",
	"help", "completion",
}

func Execute(ctx context.Context, args []string) error {
	root := newRootCmd()
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gixt",
		Short:         "run code directly from GitHub gists",
		Long:          "gixt turns GitHub gists into ephemeral CLI commands.\n\nUse `gixt run <target>` or the shorthand `gixt <target>`, where <target> is a gist id, URL, owner/gist, or a name you remembered with `gixt add`.",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if s := suggestCommand(args[0]); s != "" {
				return fmt.Errorf("unknown command %q, did you mean %q?", args[0], s)
			}
			return runTarget(cmd, args)
		},
	}
	addRunFlags(root.Flags())
	root.AddCommand(
		newRunCmd(),
		newAddCmd(),
		newRemoveCmd(),
		newListCmd(),
		newTrustCmd(),
		newGistCmd(),
		newCacheCmd(),
		newAuthCmd(),
		newSelfCmd(),
	)
	return root
}

func ensurePaths(cacheOverride string) (config.Paths, error) {
	paths, err := config.Discover(cacheOverride)
	if err != nil {
		return config.Paths{}, err
	}
	if err := config.EnsureDirs(paths); err != nil {
		return config.Paths{}, err
	}
	return paths, nil
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func PrintError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func colorize(s, code string) string {
	if code == "" || os.Getenv("NO_COLOR") != "" || !isTTY(os.Stdout) {
		return s
	}
	return code + s + "\033[0m"
}

func isTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func readLine() (string, error) {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(strings.TrimSpace(line)) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// loadToken resolves the auth token: GITHUB_TOKEN env, the stored token from
// `gixt auth login`, then (optional convenience) `gh auth token`. Returns ""
// for unauthenticated access to public gists.
func loadToken(authFile string) string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t
	}
	if b, err := os.ReadFile(authFile); err == nil {
		var a struct {
			Token string `json:"token"`
		}
		if json.Unmarshal(b, &a) == nil && a.Token != "" {
			return a.Token
		}
	}
	if t := ghAuthToken(); t != "" {
		return t
	}
	return ""
}

func ghAuthToken() string {
	gh, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, gh, "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// suggestCommand returns a likely command name when the token is a typo of a
// known command, so root execution doesn't silently treat it as a gist target.
func suggestCommand(token string) string {
	best, bestDist := "", 0
	for _, name := range knownCommands {
		if d := editDistance(strings.ToLower(token), name); d <= 2 && (best == "" || d < bestDist) {
			best, bestDist = name, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur := make([]int, len(br)+1)
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min(min(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(br)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mustBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func mustDuration(cmd *cobra.Command, name string) time.Duration {
	v, _ := cmd.Flags().GetDuration(name)
	return v
}
