# Web Development & Hugo Summary

## HTML + CSS + JS

### HTML: Core Elements

HTML provides the **skeleton** for your feature.

- **Structural Elements**
  - `<html>`, `<head>`, `<body>` - basic page structure.
  - `<header>`, `<main>`, `<footer>` - semantic layout.
  - `<section>`, `<article>`, `<aside>` - content grouping for better semantics.
  - `<nav>` - for navigation menus.
- **Content Elements**
  - `<h1>`...`<h6>` - heading hierarchy for SEO & accessibility.
  - `<p>` - paragraphs.
  - `<a href="">` - links (internal, external, anchors).
  - `<img src="">` - images with `alt` attributes for accessibility.
  - `<ul>`, `<ol>`, `<li>` - lists.
  - `<table>` with `<thead>`, `<tbody>`, `<tr>`, `<th>`, `<td>` - tabular data.
- **Media & Script**
  - `<script>` - inline or external JavaScript.
  - `<link rel="stylesheet">` - linking CSS files.
- **Forms**
  - `<form>`, `<input>`, `<textarea>`, `<button>`, `<label>` - user input.

Example from the PDF viewer:

```html
<div class="pdf-card">
  <div class="pdf-controls">
    <button class="pdf-prev">Prev</button>
    <button class="pdf-next">Next</button>
    <span>Page <span class="pdf-page">1</span> / <span class="pdf-pages">?</span></span>
    <a class="pdf-download" href="/pdfs/report.pdf" download>Download</a>
  </div>
  <div class="pdf-container">
    <canvas class="pdf-canvas"></canvas>
  </div>
</div>
```

### CSS: Styling

- **Selectors** - Element (`h1 {}`), class (`.className {}`), ID (`#idName {}`), attribute, pseudo-class (`:hover`, `:focus`).
- **Box Model** - `margin`, `border`, `padding`, `width`, `height`.
- **Positioning** - `static`, `relative`, `absolute`, `fixed`, `sticky`.
- **Layout** - `display: flex` or `grid` for layout.
- **Typography** - font size, line height, font family.
- **Colors** - `color`, `background-color`, gradients.
- **Responsive Design** - Media queries (`@media (max-width: 768px)`).
- **Custom Properties** - `:root { --main-color: #333; }` then `color: var(--main-color);`.

### JavaScript: Interaction Logic

- **DOM Manipulation** - `document.querySelector()`, `getElementById()`, `addEventListener()`.
- **Dynamic Content** - Updating inner text/HTML, toggling classes.
- **Event Handling** - `click`, `input`, `submit` events.
- **Progressive Enhancement** - JS adds interactivity but page still works without JS.

Example of the PDF reader:

```html
<script type="module">
    import pdfjsLib from 'https://cdn.jsdelivr.net/npm/pdfjs-dist/+esm';

    const container = document.getElementById("{{ $id }}");
    if (!container) throw new Error("PDF container not found");

    const canvas = container.querySelector('.pdf-canvas');
    const ctx = canvas.getContext('2d');
    const containerEl = container.querySelector('.pdf-container');
    const prevBtn = container.querySelector('.pdf-prev');
    const nextBtn = container.querySelector('.pdf-next');
    const pageSpan = container.querySelector('.pdf-page');
    const pagesSpan = container.querySelector('.pdf-pages');
    const url = "{{ $pdf_url }}";

    pdfjsLib.GlobalWorkerOptions.workerSrc =
        'https://cdn.jsdelivr.net/npm/pdfjs-dist/build/pdf.worker.mjs';

    let pdfDoc = null;
    let pageNum = 1;

    function renderPage(num) {
        pdfDoc.getPage(num).then(page => {
            var viewport = page.getViewport({ scale: 1 });
            var containerWidth = containerEl.clientWidth;
            var scale = containerWidth / viewport.width;
            var scaledViewport = page.getViewport({ scale: scale });

            var outputScale = window.devicePixelRatio || 1;
            canvas.width = Math.floor(scaledViewport.width * outputScale);
            canvas.height = Math.floor(scaledViewport.height * outputScale);
            canvas.style.width = Math.floor(scaledViewport.width) + 'px';
            canvas.style.height = Math.floor(scaledViewport.height) + 'px';

            var transform = outputScale !== 1
                ? [outputScale, 0, 0, outputScale, 0, 0]
                : null;

            return page.render({
                canvasContext: ctx,
                transform: transform,
                viewport: scaledViewport
            }).promise;
        }).then(() => {
            pageSpan.textContent = num;
            prevBtn.disabled = num <= 1;
            nextBtn.disabled = num >= pdfDoc.numPages;
        });
    }

    pdfjsLib.getDocument(url).promise.then(pdf => {
        pdfDoc = pdf;
        pagesSpan.textContent = pdf.numPages;
        renderPage(pageNum);
    });

    prevBtn.addEventListener('click', () => {
        if (pageNum <= 1) return;
        pageNum--;
        renderPage(pageNum);
    });

    nextBtn.addEventListener('click', () => {
        if (!pdfDoc || pageNum >= pdfDoc.numPages) return;
        pageNum++;
        renderPage(pageNum);
    });
</script>
```

Key patterns:
- `<script type="module">` with `import` for ES module loading (no global namespace pollution).
- `page.getViewport({ scale })` to control rendering size.
- `outputScale = window.devicePixelRatio || 1` for HiDPI support.
- `transform: [outputScale, 0, 0, outputScale, 0, 0]` passed to `page.render()` instead of manual `ctx.setTransform()`.
- `page.render({ canvasContext, transform, viewport }).promise` for async rendering.

---

## Hugo

For more content management, refer to [Hugo Content Management](https://gohugo.io/content-management/).

### Site Directory Structure

- `/static` - published as root `/`.
- `/assets` - processed by Hugo Pipes (for SCSS/JS bundling).
- `/content` - Markdown files become pages.

Refer to [Hugo Directory Structure](https://gohugo.io/getting-started/directory-structure/).

### Templates

Go templates are used for style and content injection.

- [Hugo Templates Introduction](https://gohugo.io/getting-started/directory-structure/)
- [Hugo Quick Reference Functions](https://gohugo.io/quick-reference/functions/)
- [Hugo Template Types](https://gohugo.io/templates/types/)

Hugo uses directory paths to locate templates. A **wrong path name** will cause render errors.

### Modules

Hugo uses Go modules to manage external code. You must run `hugo mod init <your remote repo>` to enable Go modules on your site.

Refer to [Hugo Modules](https://gohugo.io/hugo-modules/use-modules/).

### Common Pitfalls

#### Template Issues

**Wrong Section Name**: If you put `layouts/blog/single.html` but your content is under `/content/posts/`, Hugo won't use your blog layout (type mismatch). Fix: rename the folder or set `type: "blog"` in front matter.

**Fallback Behavior**:
- If no matching layout exists, falls back to `_default/single.html` (or `list.html`).
- If no `_default` exists, the build fails.

#### Assets vs Static

- `/assets` - files processed by Hugo Pipes (SCSS, JS pipelines). Not copied directly; reference via Hugo functions.
- `/static` - files served as-is. Copied directly to `/public/`.

#### Shortcode Scope

Shortcodes run in the page context. `.Title`, `.Params`, `.Content` work; `.Site` is also available for global settings.

**Common Problems**:
- Missing closing tag causes a build error.
- Naming conflicts: if you define a shortcode that the theme also defines, yours overrides it.

#### Context in Partials

```go-html-template
{{ partial "menu.html" . }}       {{/* passes current page context */}}
{{ partial "menu.html" .Site }}   {{/* passes site-wide data */}}
```

#### Caching

`partialCached` with multiple arguments:

```go-html-template
{{ partialCached "header.html" . .Title }}
```

Hugo caches based on both `.Site` + `.Title`. If `.Title` changes, cache invalidates.

**Hugo Pipes Cache**: If you change SCSS variables but not file names, Hugo may reuse cached CSS. Force rebuild with `hugo --ignoreCache`.
