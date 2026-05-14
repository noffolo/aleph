package osint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasImport(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		code    string
		want    bool
	}{
		{"go double-quoted import", "os/exec", `import "os/exec"`, true},
		{"go single-quoted import", "os/exec", `import 'os/exec'`, true},
		{"not present", "net/http", `import "os"`, false},
		{"empty code", "fmt", "", false},
		{"partial match only", "os", `import "os/exec"`, false},
		{"multi-line import block", "encoding/json", "import (\n\t\"encoding/json\"\n\t\"fmt\"\n)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasImport(tt.pkg)(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		code    string
		want    bool
	}{
		{"exact match", "exec.Command", `cmd := exec.Command("ls")`, true},
		{"no match", "subprocess", `cmd := exec.Command("date")`, false},
		{"empty code", "password", "", false},
		{"case-sensitive", "Password", `password= ""`, false},
		{"substring match", "api", `api_key := "secret"`, true},
		{"multiline match", "SELECT", "query := fmt.Sprintf(\"SELECT * FROM %s\", table)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsPattern(tt.pattern)(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractImports(t *testing.T) {
	ti := NewToolIntel()

	t.Run("grouped imports", func(t *testing.T) {
		code := "import (\n\t\"fmt\"\n\t\"os/exec\"\n\t\"net/http\"\n)"
		imports := ti.extractImports(code)
		assert.Equal(t, []string{"fmt", "os/exec", "net/http"}, imports)
	})

	t.Run("single-line import", func(t *testing.T) {
		code := "import \"fmt\""
		imports := ti.extractImports(code)
		assert.Equal(t, []string{"fmt"}, imports)
	})

	t.Run("commented import ignored", func(t *testing.T) {
		code := "import (\n\t\"fmt\"\n\t// \"os/exec\"\n\t\"net/http\"\n)"
		imports := ti.extractImports(code)
		assert.Equal(t, []string{"fmt", "net/http"}, imports)
	})

	t.Run("empty input", func(t *testing.T) {
		imports := ti.extractImports("")
		assert.Empty(t, imports)
	})

	t.Run("no imports", func(t *testing.T) {
		imports := ti.extractImports("package main\n\nfunc main() {}")
		assert.Empty(t, imports)
	})

	t.Run("multiple single-line imports", func(t *testing.T) {
		code := "import \"fmt\"\nimport \"os\""
		imports := ti.extractImports(code)
		assert.Equal(t, []string{"fmt", "os"}, imports)
	})
}

func TestDeduplicate(t *testing.T) {
	t.Run("removes duplicates", func(t *testing.T) {
		got := deduplicate([]string{"a", "b", "a", "c", "b"})
		assert.Equal(t, []string{"a", "b", "c"}, got)
	})

	t.Run("empty slice", func(t *testing.T) {
		got := deduplicate([]string{})
		assert.Empty(t, got)
	})

	t.Run("no duplicates", func(t *testing.T) {
		got := deduplicate([]string{"a", "b", "c"})
		assert.Equal(t, []string{"a", "b", "c"}, got)
	})

	t.Run("all duplicates", func(t *testing.T) {
		got := deduplicate([]string{"x", "x", "x"})
		assert.Equal(t, []string{"x"}, got)
	})
}

func TestScanTool(t *testing.T) {
	ti := NewToolIntel()

	t.Run("empty name returns error", func(t *testing.T) {
		_, err := ti.ScanTool("", "some code")
		assert.Error(t, err)
	})

	t.Run("clean code returns low risk", func(t *testing.T) {
		code := `package main

import (
	"crypto/rand"
	"fmt"
)

func main() {
	fmt.Println("hello")
}
`
		report, err := ti.ScanTool("my-tool", code)
		assert.NoError(t, err)
		assert.Equal(t, "my-tool", report.ToolName)
		assert.LessOrEqual(t, report.RiskScore, float64(100))
		assert.GreaterOrEqual(t, report.RiskScore, float64(0))
		assert.Contains(t, report.ScannedImports, "crypto/rand")
	})

	t.Run("risky code returns warnings", func(t *testing.T) {
		code := `package main
import "os/exec"
import "syscall"

func main() {
	password := "hunter2"
	cmd := exec.Command("rm", "-rf", "/")
	cmd.Run()
}
`
		report, err := ti.ScanTool("bad-tool", code)
		assert.NoError(t, err)
		assert.Equal(t, "bad-tool", report.ToolName)
		assert.Greater(t, report.RiskScore, float64(0))
		assert.NotEmpty(t, report.Warnings)
		assert.NotEmpty(t, report.PatternsFound)
	})

	t.Run("critical pattern gets critical level", func(t *testing.T) {
		code := "api_key := \"abcd1234\""
		report, err := ti.ScanTool("sensitive-tool", code)
		assert.NoError(t, err)
		// hardcoded API key is critical, should push score high
		assert.GreaterOrEqual(t, report.RiskScore, float64(35))
	})
}
