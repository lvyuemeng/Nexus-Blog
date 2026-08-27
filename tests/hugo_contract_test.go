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

	for _, resource := range [][]string{
		{"posts", "markdown", "diagram.svg"},
		{"posts", "markdown", "images", "detail.svg"},
		{"posts", "markdown", "attachment.txt"},
		{"posts", "pdf", "paper.pdf"},
		{"posts", "legacy-pdf", "legacy.pdf"},
	} {
		assertOutputExists(t, outputDir, resource...)
	}

	pdfPage := readOutput(t, outputDir, "posts", "pdf", "index.html")
	for _, expected := range []string{`data-pdf-viewer`, `/posts/pdf/paper.pdf`, `pdfjs-dist@6.2.108`} {
		if !strings.Contains(pdfPage, expected) {
			t.Errorf("PDF output missing %q", expected)
		}
	}

	legacyPage := readOutput(t, outputDir, "posts", "legacy-pdf", "index.html")
	for _, expected := range []string{`data-pdf-viewer`, `/posts/legacy-pdf/legacy.pdf`, `pdfjs-dist@6.2.108`} {
		if !strings.Contains(legacyPage, expected) {
			t.Errorf("legacy PDF output missing %q", expected)
		}
	}
}

func TestInvalidContentContract(t *testing.T) {
	tests := []struct {
		fixture string
		message string
	}{
		{fixture: "missing-image", message: `required image resource "missing.png"`},
		{fixture: "missing-attachment", message: `required attachment resource "missing.pdf"`},
		{fixture: "missing-pdf", message: `required PDF resource "missing.pdf"`},
		{fixture: "empty-pdf", message: "PDF post requires a nonempty document parameter"},
		{fixture: "non-pdf-document", message: `PDF document "note.txt" has media type "text/plain"`},
		{fixture: "root-relative-image", message: `required image resource "/images/missing.png" must be a bundle-relative path`},
		{fixture: "traversal-image", message: `required image resource "../shared/missing.png" must be a bundle-relative path`},
		{fixture: "remote-pdf", message: `required PDF resource "https://example.com/paper.pdf" must be a bundle-relative path`},
		{fixture: "machine-image", message: `required image resource "C:/notes/image.png" must be a bundle-relative path or use a permitted remote protocol`},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			_, log, err := buildFixture(t, tt.fixture)
			if err == nil {
				t.Fatalf("expected Hugo to reject %s", tt.fixture)
			}
			if !strings.Contains(log, tt.message) {
				t.Fatalf("failure did not contain %q:\n%s", tt.message, log)
			}
		})
	}
}

func buildFixture(t *testing.T, fixture string) (string, string, error) {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), ".."))
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
		"--contentDir", filepath.Join(root, "testdata", "contract", fixture),
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
