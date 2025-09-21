# Contribution Guidelines

We welcome and appreciate contributions from the community! Whether you want to fix a typo, improve code, add new features, or submit guest posts, please follow these guidelines to ensure a smooth collaboration process.

## How to Contribute

### Prerequisite

- `git`
- `editor`(vscode/neovim ...)

### 1. Contributing Code / Bug Fixes

- Install `hugo` by [hugo installation](https://gohugo.io/installation/)
- Report bugs or suggest features by [opening an issue](https://github.com/lvyuemeng/opencamp-blog/issues/new)
- Fork the repository
- Create a feature/bugfix branch:  
  `git checkout -b feature/your-feature-name` or `git checkout -b fix/your-bugfix-name`
- Commit your changes with descriptive messages
- Push to your branch: `git push origin your-branch-name`
- Open a Pull Request (PR) against the `main` branch

---

### 2. Contributing Articles

- Create a new branch for each article: `git checkout -b new_branch`

---

#### Single Article

- Following below criterion or `hugo new content <repo-root>/content/posts/<blog-name>` which will create header automatically.
- Any possible variables for template could be checked on [Variables](https://hugo-docs.netlify.app/en/variables/page/) and [PaperMod-Wiki](https://github.com/adityatelange/hugo-PaperMod/wiki/Features)

#### Multiple Articles

This is suitable for a *isolated* repo with hugo initialization.

- You should have a isolated repo contains all your notes, then use `hugo mod init <your remote repo>` to initialize hugo module.
- (Optional) create a default template for header by placing below code in `/archetypes/default.md`:

```yaml
---
title: '{{ replace .File.ContentBaseName "-" " " | title }}'
date: {{ .Date }}
draft: true
tags: []
author: []
---
```

However, it's optional to replace the format in `yaml/toml` or your custom template referring Hugo docs!

- In **Nexus** repo, insert your repo like this:

```toml
[module]
...

[[module.imports]]
path = "github.com/yourusername/your notes repo"
mount = [
  { source = "posts", target = "content/posts/notes" },
  { source = "assets/images", target = "static/notes/images" }
]
```

Where it place your `posts/` in `content/posts/notes` and `assets/images` in `static/notes/images`. It's up to you to create your own path resolution.

---

- Check your content by `hugo server -D`(`-D` means draft shown)
- If success, change `draft = true` to `draft = false`
- Commit your changes and push to your branch
- Open a Pull Request (PR) against the `main` branch

---

### Format

- All articles should be written in Markdown (.md)
- Place files in `content/posts/` directory
- Follow naming convention: `your_article_title_YYYY-MM-DD.md`
- Include proper front matter: 

  ```md
  ---
  title = "Your Article Title"
  date = YYYY-MM-DD
  draft = true 
  author = Your Name (or pseudonym)
  tags = [tag1, tag2]
  categories = [category1]
  ---
  
  Your article content goes here...
  ```

#### Latex

If you want to render latex, please place below inline or block:

```md
$$
x + y = 3 \
$$

The equation of $x^2 + y^2 = 1$ is ...
```