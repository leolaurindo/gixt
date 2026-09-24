# gixt

Keep versioned scripts, prompts, snippets, and templates in GitHub Gists,
give them names, and use them from anywhere.

Retrieve prompt and text artifacts safely:

```sh
gixt add leolaurindo/review-prompt.md --as review_prompt
gixt cat review_prompt >> AGENTS.md
gixt cat leolaurindo/some_skill.md >> .agents/skills/some_skill/SKILL.md
```

Run executable artifacts explicitly:

```sh
gixt add leolaurindo/hello-world --as hello
gixt run hello
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
