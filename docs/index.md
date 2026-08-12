# gixt

Your personal, versioned cloud clipboard.

Keep tiny scripts, prompts, snippets, and templates in GitHub Gists, give them
names, versioning, and use them from anywhere.

Run executable artifacts as commands:

```sh
gixt add leolaurindo/hello-world --as hello
gixt hello
```

Print non-executable artifacts when you need their contents and plug into your workflow:

```sh
gixt add leolaurindo/review-prompt.md --as review_prompt
gixt review_prompt --view >> AGENTS.md
gixt leolaurindo/some_skill.md >> .agents/skills/some_skill/SKILL.md
```

## Install

Linux and macOS:

```sh
curl -fsSL https://gixt.leolaurindo.com/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://gixt.leolaurindo.com/install.ps1 | iex
```

## Documentation

- [CLI usage](cli-usage.md)
- [Caching and index](caching-and-index.md)
- [Trust and security](trust-and-security.md)

[GitHub repository](https://github.com/leolaurindo/gixt)
