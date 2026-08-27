package contract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidContentContract(t *testing.T) {
	outputDir, log, err := buildFixture(t, "valid")
	if err != nil {
		t.Fatalf("valid content contract failed: %v\n%s", err, log)
	}

	markdown := readOutput(t, outputDir, "posts", "markdown", "index.html")
	for _, expected := range []string{
		`src="/posts/markdown/diagram.svg"`,
		`src="/posts/markdown/diagram.svg?variant=contract#preview"`,
		`src="/posts/markdown/images/detail.svg"`,
		`href="/posts/markdown/attachment.txt"`,
		`href="../target/"`,
		`href=".github/workflows/example.yml"`,
		`src="https://example.com/remote.png"`,
		`href="https://example.com/reference"`,
		`mathjax@4.1.3/tex-chtml.js`,
	} {
		if !strings.Contains(markdown, expected) {
			t.Errorf("Markdown output missing %q", expected)
		}
	}
	if strings.Contains(markdown, "typeset: false") {
		t.Error("MathJax disables initial typesetting")
	}
	plainPage := readOutput(t, outputDir, "posts", "target", "index.html")
	if strings.Contains(plainPage, "MathJax-script") {
		t.Error("page without math opt-in loads MathJax")
	}

	for _, resource := range [][]string{
		{"posts", "markdown", "diagram.svg"},
		{"posts", "markdown", "images", "detail.svg"},
		{"posts", "markdown", "attachment.txt"},
		{"posts", "pdf", "paper.pdf"},
	} {
		assertOutputExists(t, outputDir, resource...)
	}

	pdfPage := readOutput(t, outputDir, "posts", "pdf", "index.html")
	for _, expected := range []string{`<object`, `type="application/pdf"`, `/posts/pdf/paper.pdf`, `Download PDF`} {
		if !strings.Contains(pdfPage, expected) {
			t.Errorf("PDF output missing %q", expected)
		}
	}
	if strings.Contains(pdfPage, "pdfjs-dist") {
		t.Error("PDF output still loads the retired PDF.js runtime")
	}

}

func TestInvalidContentContract(t *testing.T) {
	tests := []struct {
		name        string
		frontMatter string
		body        string
		files       map[string]string
		message     string
	}{
		{name: "missing-image", body: `![Missing](missing.png)`, message: `required image resource "missing.png"`},
		{name: "missing-attachment", body: `[Missing](./missing.dataset)`, message: `required attachment resource "./missing.dataset"`},
		{name: "missing-pdf", frontMatter: "type: pdf\ndocument: missing.pdf\n", message: `required PDF resource "missing.pdf"`},
		{name: "empty-pdf", frontMatter: "type: pdf\ndocument: \"\"\n", message: "PDF post requires a nonempty document parameter"},
		{name: "non-pdf-document", frontMatter: "type: pdf\ndocument: note.txt\n", files: map[string]string{"note.txt": "not a PDF\n"}, message: `PDF document "note.txt" has media type "text/plain"`},
		{name: "root-relative-image", body: `![Invalid](/images/missing.png)`, message: `required image resource "/images/missing.png" must be a bundle-relative path`},
		{name: "traversal-image", body: `![Invalid](../shared/missing.png)`, message: `required image resource "../shared/missing.png" must be a bundle-relative path`},
		{name: "remote-pdf", frontMatter: "type: pdf\ndocument: https://example.com/paper.pdf\n", message: `required PDF resource "https://example.com/paper.pdf" must be a bundle-relative path`},
		{name: "machine-image", body: `![Invalid](C:/notes/image.png)`, message: `required image resource "C:/notes/image.png" must be a bundle-relative path or use a permitted remote protocol`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, log, err := buildInvalidFixture(t, tt.frontMatter, tt.body, tt.files)
			if err == nil {
				t.Fatalf("expected Hugo to reject %s", tt.name)
			}
			if !strings.Contains(log, tt.message) {
				t.Fatalf("failure did not contain %q:\n%s", tt.message, log)
			}
		})
	}
}

func TestNoLegacyNamespace(t *testing.T) {
	root := projectRoot(t)
	legacy := "lvyue" + "meng"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && (info.Name() == ".git" || info.Name() == ".agents") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(strings.ToLower(string(content)), legacy) {
			t.Errorf("legacy namespace found in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func buildFixture(t *testing.T, fixture string) (string, string, error) {
	t.Helper()
	root := projectRoot(t)
	return buildContentDir(t, root, filepath.Join(root, "testdata", "contract", fixture))
}

func buildInvalidFixture(t *testing.T, frontMatter, body string, files map[string]string) (string, string, error) {
	t.Helper()
	contentDir := t.TempDir()
	bundleDir := filepath.Join(contentDir, "posts", "broken")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: Broken contract\ndate: 2026-08-27\ndraft: false\n" + frontMatter + "---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(bundleDir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildContentDir(t, projectRoot(t), contentDir)
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), ".."))
}

func buildContentDir(t *testing.T, root, contentDir string) (string, string, error) {
	t.Helper()
	hugo, err := exec.LookPath("hugo")
	if err != nil {
		t.Fatal("hugo executable is required")
	}

	temp := t.TempDir()
	outputDir := filepath.Join(temp, "public")
	cmd := exec.Command(
		hugo,
		"--source", root,
		"--config", filepath.Join(root, "testdata", "contract", "hugo.toml"),
		"--contentDir", contentDir,
		"--destination", outputDir,
		"--cacheDir", filepath.Join(temp, "cache"),
		"--noBuildLock",
		"--printPathWarnings",
	)
	cmd.Env = append(os.Environ(), "HUGO_ENVIRONMENT=contract")
	combined, runErr := cmd.CombinedOutput()
	return outputDir, string(combined), runErr
}

func readOutput(t *testing.T, outputDir string, elements ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{outputDir}, elements...)...)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated output %s: %v", path, err)
	}
	return string(content)
}

func assertOutputExists(t *testing.T, outputDir string, elements ...string) {
	t.Helper()
	path := filepath.Join(append([]string{outputDir}, elements...)...)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected generated resource %s: %v", path, err)
	}
}
