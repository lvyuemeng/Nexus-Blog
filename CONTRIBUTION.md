# Contributing notes

Nexus owns the Hugo theme, rendering rules, validation, CI, and deployment.
Contributors provide portable Markdown or PDF page bundles; do not copy Nexus
layouts, scripts, or machine-specific paths into a note repository.

## Markdown notes

A note without local files may be a single Markdown file. When a note includes
images, attachments, or other resources, use a Hugo leaf bundle:

```text
my-note/
  index.md
  diagram.svg
  results.dataset
  images/
    detail.png
```

Images are relative to `index.md`:

```markdown
![Overview](diagram.svg)
![Detail](images/detail.png)
```

Use an explicit `./` prefix for downloadable bundle attachments. This works for
any file extension and distinguishes an attachment from navigation to another
page:

```markdown
[Download the results](./results.dataset)
```

Missing local images and explicit attachments fail the build. Remote `http` and
`https` URLs remain valid. Root-relative paths, machine paths, and `../`
cross-bundle traversal are not portable note resources.

## Math

Declare math on pages that need it:

```yaml
---
title: "A mathematical note"
date: 2026-08-27
math: true
---
```

Supported delimiters are `\(...\)` for inline math and `\[...\]` or `$$...$$`
for display math.

## PDF posts

A PDF post is a leaf bundle with a small metadata wrapper:

```text
paper/
  index.md
  paper.pdf
```

```yaml
---
title: "Paper title"
date: 2026-08-27
type: pdf
document: paper.pdf
---
```

`document` must identify a colocated PDF. Nexus publishes it, embeds the native
browser PDF view, and provides a download fallback. Rendering shortcodes are not
part of the posting contract.

## Ways to contribute

- Occasional contributors submit a namespaced page bundle to the shared content
  repository selected by the maintainers.
- Independent authors may maintain a minimal Hugo module containing their notes,
  resources, `go.mod`, and content mount. Nexus must explicitly review,
  namespace, and pin each module before it becomes part of the site.

See Hugo's [module documentation](https://gohugo.io/hugo-modules/) for the
upstream module format. A provider module owns content only; host layouts,
MathJax, PDF presentation, validation, and deployment stay in Nexus.

## Verify Nexus

From the Nexus repository root, with Go and Hugo Extended installed:

```sh
go test ./... -count=1
hugo --gc --minify
```

The first command verifies valid and intentionally invalid note fixtures. The
second proves the pinned provider revisions integrate with the production host.
