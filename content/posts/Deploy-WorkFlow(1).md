+++
date = '2025-03-24T22:27:11+08:00'
draft = false
title = 'Deploy WorkFlow(1)'
description = 'A guide to deploying a static blog with Hugo, covering site creation, workflow configuration, and GitHub Pages setup.'
tags = ["Deploy"]
author = ["nostalgia"]
+++

# Deploy of Blog

Here is a record of deploying a static blog with `hugo`.

## Preparation

Toolchain:

- hugo
- git

## Create Site

```bash
hugo new site <your-name>
cd <your-name>
git init
git submodule add https://gitclone.com/github.com/adityatelange/hugo-PaperMod.git themes/PaperMod
```

Setting the theme to "PaperMod"

```toml
# ... above content
theme = "PaperMod"
```

## Workflow Configuration

### Git Ignore

Set up `.gitignore` to remove auto-generated content:
```
.hugo_build.lock
public/
resources/
```

### Issue Template

A readable template can help you classify issues and make them more understandable.

Set up issue templates located in `.github/ISSUE_TEMPLATE` with:
```
bug_report.md
feature_request.md
question_answer.md
config.yml
```

Here is a part of the demonstration:
```yaml
blank_issues_enabled: false
contact_links:
  - name: Report Issue
    url: https://github.com/lvyuemeng/Nexus-Blog/issues/new
    description: Report a bug or request a feature
  - name: Github Discussions
    url: https://github.com/lvyuemeng/Nexus-Blog/discussions
    description: Ask a question or start a discussion
```

### Action

An automated action can be used for deployment, test coverage in PRs, etc. Currently, we set `deploy` and `pr-validate` for these two cases.

Actions can be decomposed into the following main components:
```yaml
name: <Workflow Name>          # Display name in Actions tab
on: [<trigger events>]        # When to run the workflow
jobs:                          # Group of tasks to execute
  <job-name>:                  # Unique job identifier
    runs-on: <os-image>        # Execution environment
    steps:                     # Sequential operations
      - name: <Step Name>      # Human-readable step name
        uses/run: <command>    # Action or shell command
```

You can use others or given actions to ease your burden, for example:
```yaml
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

#### Deploy

Following the `hugo` tutorial, you can deploy on `GitHub`:

1. Go to the repo `Settings` and find `Pages`
2. Switch the `Build and deployment`'s `Source` to `GitHub Actions`

> [Deploy Action](.github/workflows/hugo.yml)

1. Install Hugo CLI
2. Get PR Code
3. Build Code
4. Upload Site
5. Deploy Site

#### Pull Request Validation

The logic is the same.

> [PR Validate Action](.github/workflows/pr-validate.yml)

1. Install Hugo CLI
2. Get PR Code
3. Build Code
4. Check `./public` consistency

## Beautify

The ways to beautify your blog depend on your theme. [`PaperMod`](https://github.com/adityatelange/hugo-PaperMod/wiki/Features) provides various configurations. Here is a subset for reference:

```toml
# Settings for PaperMod
[params]
showShareButtons = true
showReadingTime = true
showCodeCopyButtons = true
showToC = true
showBreadCrumbs = true

# Setting for a profile cover
[params.profileMode]
enabled = true
title = "Nexus"

# Settings for search 
[params.fuseOpts]
isCaseSensitive = false
shouldSort = true
location = 0
distance = 1_000
threshold = 0.4
minMatchCharLength = 0
keys = ["title", "permalink", "summary", "content"]

[outputs]
home = ["HTML", "RSS", "JSON"]
```


