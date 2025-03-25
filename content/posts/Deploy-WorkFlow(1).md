+++
date = '2025-03-24T22:27:11+08:00'
draft = true
title = 'Deploy WorkFlow(1)'
tag = ["Deploy"]
author = ["nostalgia"]
+++

## Deploy of Blog

Here a record of deployment of static blog by `hugo`

### Preparation

Toolchain:

- hugo
- git

### Create Site

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

### Workflow Configuration

#### Git Ignore

Setting .gitignore to remove auto-generated content
```
.hugo_build.lock
public/
resources/
```

#### Issue Template

A readable template can help you classify issues, make it more understandable.

Setting issue template located in `.github/ISSUE_TEMPLATE` with:
```
bug_report.md
feature_request.md
question_answer.md
config.yml
```

Here a part of demonstration:
```yaml
blank_issues_enabled: false
contact_links:
  - name: 📃 Report Issue 
    url: https://github.com/lvyuemeng/opencamp-blog/issues/new
    description: Report a bug or request a feature
  - name: 👀 Github Discussions
    url: https://github.com/lvyuemeng/opencamp-blog/discussions
    description: Ask a question or start a discussion
```

#### Action

A automated action can be used for deployment, test coverage in pr, etc. Currently, we set the 'deploy' and 'pr-validate' for such two cases.

Action can be decomposed to below main factors:
```yaml
name: <Workflow Name>          # Display name in Actions tab
on: [<trigger events>]        # When to run the workflow
jobs:                       # Group of tasks to execute
  <job-name>:                 # Unique job identifier
	environment(optional):
		name: <env name>
		url: <url name>
    runs-on: <os-image>       # Execution environment
    steps:                  # Sequential operations
      - name: <Step Name>     # Human-readable step name
        uses/run: <command>   # Action or shell command
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

### Beautify

The ways to beautify your blog is based on your theme. Here [`PaperMod`](https://github.com/adityatelange/hugo-PaperMod/wiki/Features) provide various configurations, we list a part of it for our own:

```toml
# Setting for a profile cover
[params.profileMode]
enabled = true
title = "nostalgia"
showShareButtons = true
showReadingTime = true
showCodeCopyButtons = true
showToC = true
showBreadCrumbs = true

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


