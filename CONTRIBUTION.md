# Contribution Guidelines

We welcome and appreciate contributions from the community! Whether you want to fix a typo, improve code, add new features, or submit guest posts, please follow these guidelines to ensure a smooth collaboration process.

## How to Contribute

### Prerequisites

- `git`
- An editor (VS Code, Neovim, etc.)

### 1. Contributing Code / Bug Fixes

- Install `hugo` via [Hugo installation](https://gohugo.io/installation/)
- Report bugs or suggest features by [opening an issue](https://github.com/lvyuemeng/Nexus-Blog/issues/new)
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

- Follow the criteria below, or use `hugo new content <repo-root>/content/posts/<blog-name>` which will create the front matter automatically.
- Available template variables can be checked at [Variables](https://hugo-docs.netlify.app/en/variables/page/) and [PaperMod Wiki](https://github.com/adityatelange/hugo-PaperMod/wiki/Features).

#### Multiple Articles

This is suitable for an *isolated* repo with Hugo initialization.

- You should have an isolated repo containing all your notes. Use `hugo mod init <your remote repo>` to initialize the Hugo module.
- Push your local repo to a remote like GitHub.
- (Optional) Create a default archetype template by placing the code below in `/archetypes/default.md`:

```yaml
---
title: '{{ replace .File.ContentBaseName "-" " " | title }}'
date: {{ .Date }}
draft: true
tags: []
author: []
---
```

You can replace the format with `yaml/toml` or your custom template. Refer to the Hugo docs for options.

Then create new posts with `hugo new -c "./your-dir" "post-name.md"`. Note: you **cannot** create posts in the **root dir** because Hugo resolves the parent dir for the archetype template.

- In the **Nexus** repo, add your module import:

```toml
[module]
[[module.imports]]
...
path = "github.com/yourusername/your-notes-repo"
```

- In **your notes** repo, add your mount paths:

```toml
[[module.mounts]]
source = "posts/"
target = "content/posts/your-name" # Use a unique path to avoid conflicts!

[[module.mounts]]
source = "posts/single-post.md"
target = "content/posts/your-name/single-post.md"
```

This places your `posts/` in `content/posts/your-name`. You control your own path resolution.

If you want to insert images or other assets, be careful with path resolution. The recommended choices compatible with both local viewing and Hugo are:

- Page Bundle:

Place the specific post as above where the markdown file should be named **`index.md`**.

```text
my-post-1/
  index.md
  picture.png
```

Resolve path as `![a picture](picture.png)`.

- Relative path in `static/`:

Two posts sharing the same `assets/logo.png`:

```text
My-Notes/
  post-1/
    index.md
    picture.png
  post-2.md
  assets/
    logo.png
```

Use relative link `![logo](assets/logo.png)`. Mount it in the module as:

```toml
[[module.mounts]]
source = "My-Notes/"
target = "content/posts/your-name/"

[[module.mounts]]
source = "assets"
target = "static/posts/your-name/assets"
```

Given a file with link in `content/posts/your-name/post-2.md`, it is mapped as `/posts/your-name/post-2.md`,
with assets mapped as `/posts/your-name/assets/`, so the relative link `assets/logo.png` resolves correctly.

Be careful with link relations to prevent broken paths.

**Caveat**:

- If multiple people mount the same path, Hugo will merge all.
- If the post names are **the same**, Hugo will choose the one with higher import priority.
- To avoid conflicts, use a custom path name!

---

- Check your content with `hugo server -D` (`-D` means drafts are shown)
- If successful, change `draft = true` to `draft = false`
- Commit your changes and push to your branch
- Open a Pull Request (PR) against the `main` branch

---

### Format

- All articles should be written in Markdown (`.md`)
- Place files in the `content/posts/` directory
- Follow the naming convention: `your_article_title.md`
- Include proper front matter (TOML format):

  ```toml
  +++
  title = "Your Article Title"
  date = YYYY-MM-DD
  draft = true
  author = ["Your Name"]
  tags = ["tag1", "tag2"]
  +++

  Your article content goes here...
  ```

#### LaTeX

For inline or block math snippets in Markdown, use MathJax syntax rendered on the page:

```latex
$$
x + y = 3
$$

The equation of $x^2 + y^2 = 1$ is ...
```

LaTeX math delimiters: block uses `$$...$$` or `\[...\]`, inline uses `$...$` or `\(...\)`.

For full LaTeX documents, compile to PDF and embed via the PDF shortcode.

**Workflow**:

1. Write your document (`.tex` file):

```latex
\documentclass{article}
\usepackage{amsmath}

\begin{document}

\title{My Article}
\author{Your Name}
\date{\today}
\maketitle

The equation of $x^2 + y^2 = 1$ defines a unit circle.

\begin{equation}
  \int_0^1 x^2 \, dx = \frac{1}{3}
\end{equation}

\end{document}
```

2. Compile to PDF:

```bash
pdflatex my-document.tex
```

Or use [Overleaf](https://www.overleaf.com/) to edit and compile online.

3. Embed the compiled PDF in your Markdown post:

```markdown
{{< pdf src="./my-document.pdf" >}}
```

#### Typst

[Typst](https://typst.app/) is a modern alternative to LaTeX for typesetting. The workflow is the same: compile to PDF and embed.

**Workflow**:

1. Write your document (`.typ` file):

```typst
#set page(width: 210mm, height: auto)
#set text(font: "New Computer Modern", size: 12pt)

= My Article

The equation of $x^2 + y^2 = 1$ defines a unit circle.

$
  integral_0^1 x^2 dif x = 1/3
$
```

2. Compile to PDF:

```bash
typst compile my-document.typ
```

Or use the [Typst Web App](https://typst.app/) to compile online.

3. Embed the compiled PDF in your Markdown post:

```markdown
{{< pdf src="./my-document.pdf" >}}
```

**Typst vs LaTeX math quick reference**:

| Feature | LaTeX | Typst |
|---|---|---|
| Inline math | `$x^2$` | `$x^2$` |
| Block math | `$$...$$` | `$ ... $` |
| Fractions | `\frac{a}{b}` | `a/b` or `(a)/(b)` |
| Integral | `\int_0^1` | `integral_0^1` |
| Sum | `\sum_{i=1}^{n}` | `sum_(i=1)^n` |
| Square root | `\sqrt{x}` | `sqrt(x)` |
| Greek letters | `\alpha, \beta` | `alpha, beta` |

If you contribute LaTeX or Typst content, include the source file (`.tex` or `.typ`) alongside the compiled PDF so others can edit it.

### PDF

We use a custom PDF shortcode. Refer to `layouts/shortcodes/pdf.html` for implementation details.

```
{{< pdf src="./path/to/pdf/file/example.pdf" >}}
```
