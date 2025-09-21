# 📚 Web Development & Hugo Summary

---

## 🏗 HTML + CSS + JS

### HTML: Core Elements

HTML provides the **skeleton** for your feature.

- **Structural Elements**
  - `<html>`, `<head>`, `<body>` – basic page structure.
  - `<header>`, `<main>`, `<footer>` – semantic layout.
  - `<section>`, `<article>`, `<aside>` – content grouping for better semantics.
  - `<nav>` – for navigation menus.
- **Content Elements**
  - `<h1>`...`<h6>` – heading hierarchy for SEO & accessibility.
  - `<p>` – paragraphs.
  - `<a href="">` – links (internal, external, anchors).
  - `<img src="">` – images with `alt` attributes for accessibility.
  - `<ul>`, `<ol>`, `<li>` – lists.
  - `<table>` with `<thead>`, `<tbody>`, `<tr>`, `<th>`, `<td>` – tabular data.
- **Media & Script**
  - `<script>` – inline or external JavaScript.
  - `<link rel="stylesheet">` – linking CSS files.
- **Forms (if any used)**
  - `<form>`, `<input>`, `<textarea>`, `<button>`, `<label>` – user input.

Example from our PDF viewer:

```html
<div class="pdf-card">
  <div class="pdf-controls">
    <button class="pdf-prev">◀ Prev</button>
    <button class="pdf-next">Next ▶</button>
    <span>Page <span class="pdf-page">1</span> / <span class="pdf-pages">?</span></span>
    <a class="pdf-download" href="/pdfs/report.pdf" download>⬇ Download</a>
  </div>

  <div class="pdf-container">
    <canvas class="pdf-canvas"></canvas>
  </div>
</div>
```

---

## CSS: Beautification

- **Selectors**
  - Element (`h1 {}`), class (`.className {}`), ID (`#idName {}`), attribute, pseudo-class (`:hover`, `:focus`).
- **Box Model**
  - `margin`, `border`, `padding`, `width`, `height`.
- **Positioning**
  - `static`, `relative`, `absolute`, `fixed`, `sticky`.
- **Flexbox/Grid**
  - Used `display: flex` or `grid` for layout.
- **Typography**
  - Controlled font size, line height, and font family.
- **Colors & Themes**
  - `color`, `background-color`, gradients.
- **Responsive Design**
  - Media queries (`@media (max-width: 768px)`).
- **Custom Properties**
  - `:root { --main-color: #333; }` then `color: var(--main-color);`.

---

## JavaScript: Interaction Logic

- **DOM Manipulation**
  - `document.querySelector()`, `getElementById()`, `addEventListener()`.
- **Dynamic Content**
  - Updating inner text/HTML, toggling classes.
- **Event Handling**
  - `click`, `input`, `submit` events.
- **Progressive Enhancement**
  - JS adds interactivity but page still works without JS.
  
We apply the example of pdf reader here:

```html
<script>
    (function () {
        const container = document.getElementById("{{ $id }}");
        if (!container) return;

        const canvas = container.querySelector('.pdf-canvas');
        const ctx = canvas.getContext('2d');
        const prevBtn = container.querySelector('.pdf-prev');
        const nextBtn = container.querySelector('.pdf-next');
        const pageSpan = container.querySelector('.pdf-page');
        const pagesSpan = container.querySelector('.pdf-pages');
        const url = "{{ $pdf_url }}";

        pdfjsLib.GlobalWorkerOptions.workerSrc = "https://cdn.jsdelivr.net/npm/pdfjs-dist@3.9.179/build/pdf.worker.js";

        let pdfDoc = null;
        let pageNum = 1;

        function renderPage(num) {
            pdfDoc.getPage(num).then(page => {
				...
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
			... // same as `prevBtn`.
        });

        window.addEventListener('resize', () => renderPage(pageNum));
    })();
</script>
```

We only denote that:

- Self-invoking function: `(function () {...})()` to immediately invoke function in itself avoiding polluting global scope.
- `const container = document.getElementById("{{ $id }}")` to retrieve element by id.
- `container.querySelector(...)` to retrieve inner element inside the container element.

```js
pdfjsLib.getDocument(url).promise.then(pdf => {
    pdfDoc = pdf;
    pagesSpan.textContent = pdf.numPages;
    renderPage(pageNum);
});
```
- Use the corresponding method to get document and render.
- `addEventListener` of certain element for interaction.

---

## Hugo

For more content management, please refer [hugo content management](https://gohugo.io/content-management/)

### Basic Layout of Site

- `/static` → published as root `/`.
- `/assets` → processed by Hugo Pipes (for SCSS/JS bundling).
- `/content` → Markdown files become pages.

For more please refer to [hugo directory structure](https://gohugo.io/getting-started/directory-structure/)

### Template

Template of go is a convenient tool for style or content injection.

For basic usage, refer to [hugo templates introduction](https://gohugo.io/getting-started/directory-structure/)

For hugo predefined function can be used, refer to [hugo quick reference function](https://gohugo.io/quick-reference/functions/)

However, hugo also build the order to seek and apply template, thus beside basic usage, to apply html component, you also have to refer to [hugo templates types](https://gohugo.io/templates/types/).

It should be clarified that hugo use directory path to seek thing, otherwise a **wrong nomination of path** will cause seeking nothing or render error.

### Module

Hugo use go module to manage external code. However, to use it, you must `hugo mod init <your remote repo>` for your site to create go module on yourself! Otherwise go module can't apply on your repo!

You could refer to [hugo use modules](https://gohugo.io/hugo-modules/use-modules/)

### Edge Case

#### Template

**Wrong Section Name**

If you put `layouts/blog/single.html` but your content is under `/content/posts/` → Hugo won’t use your blog layout (type mismatch).

Fix: either rename folder to blog/ or set front matter:

```yaml
type: "blog" 
```

---

**Fallback Behavior**

- If no matching layout exists → falls back to `_default/single.html` (or list.html).

- If no `_default` exists → build fails

#### Assets & Static

**Assets Folder**: 

- Files in `/assets/` are not copied directly — you must process them with Hugo Pipes.

- If you just put a file there and reference `/assets/file.css`, it will 404.

**Static Folder**:

- Good for images, favicons, or unprocessed JS.

- Files in `/static/` are copied as-is to the final `/public/` which is root directory `/` in deployment.

**Summary**:

- `/assets` → for files you want Hugo to process (SCSS, JS pipelines)

- `/static` → for files you just want to serve directly

---

#### Short Code

**Scope of Shortcodes**:

- Shortcodes run in the page context, so `.Title, .Params, .Content` work; but `.Site` is still available, so you can access global settings.

**Common Problems**:

- Closing Tag Missing:

```
{{< myshortcode >}}
content here
<!-- forgot closing {{< /myshortcode >}} -->
```

→ build error.

**Naming Conflicts**:

If you create `layouts/shortcodes/youtube.html` but your theme also defines one, yours overrides it — sometimes breaking theme functionality.

---

### Context Pitfalls

When passing data to partials:

```
{{ partial "menu.html" . }}
```

`.` here means *current context* — usually the page.

If you need only site-wide data, pass `.Site`:

```
{{ partial "menu.html" .Site }}
```

Inside `menu.html`, you can then safely do:

```
{{ .Menus.main }}
```

---

### Performance & Caching

`partialCached` with multiple arguments:

```
{{ partialCached "header.html" . .Title }}
```

→ Hugo caches based on both .Site + .Title.
If `.Title` **changes**, cache invalidates.

**Hugo Pipes Cache**:

If you change `SCSS` variables but not file names, Hugo may reuse cached `CSS` → force rebuild with `hugo --ignoreCache`.