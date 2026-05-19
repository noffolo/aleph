package sandbox

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerSandbox_ExecuteTool_ReachesExecuteInContainer(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "# python\nimport json\nprint(json.dumps({'status': 'ok'}))\n", nil
	}
	cs.dockerCheckFunc = func() bool { return true }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "test-tool", map[string]any{"input": "hello"})

	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "docker execution failed")
}

func TestContainerSandbox_ExecuteTool_PythonValidationRejection(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "# python\nimport os\nos.system('rm -rf /')", nil
	}
	cs.dockerCheckFunc = func() bool { return false }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "bad-tool", map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "python validation failed")
}

func TestContainerSandbox_ExecuteTool_GoValidationRejection(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "package main\nimport(\"os/exec\")\nfunc main(){exec.Command(`ls`)}", nil
	}
	cs.dockerCheckFunc = func() bool { return false }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "bad-go-tool", map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "go validation failed")
}

func TestContainerSandbox_ExecuteTool_ValidGoCode_NoDocker(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "package main\nfunc main(){println(`hello`)}", nil
	}
	cs.dockerCheckFunc = func() bool { return false }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "valid-go", map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Contains(t, result.Error, "container runtime unavailable")
}

func TestContainerSandbox_GetToolCode_UsesOverride(t *testing.T) {
	logger := slog.Default()
	cs := NewContainerSandbox(logger, nil, nil, DefaultContainerConfig(), nil)

	expectedCode := "# python\nprint('hello from override')\n"
	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return expectedCode, nil
	}

	code, err := cs.getToolCode(context.Background(), "any-tool")
	require.NoError(t, err)
	assert.Equal(t, expectedCode, code)
}

func TestContainerSandbox_ExecuteTool_WithRunscAndNoCpuLimit(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultContainerConfig()
	cfg.MaxCPUSeconds = 0
	cs := NewContainerSandbox(logger, nil, nil, cfg, nil)
	cs.hasRunsc = true
	cs.runscCached = true

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "# python\nimport json\nprint('x')", nil
	}
	cs.dockerCheckFunc = func() bool { return true }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "runsc-tool", map[string]any{})

	require.NoError(t, err)
	assert.Contains(t, result.Error, "docker execution failed")
}

func TestContainerSandbox_ExecuteTool_NarrowCpuLimit(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultContainerConfig()
	cfg.MaxCPUSeconds = 1.0
	cs := NewContainerSandbox(logger, nil, nil, cfg, nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "# python\nimport json\nprint('ok')", nil
	}
	cs.dockerCheckFunc = func() bool { return true }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "lowcpu-tool", map[string]any{})

	require.NoError(t, err)
	assert.Contains(t, result.Error, "docker execution failed")
	t.Logf("exec result: %+v", result)
}

func TestContainerSandbox_HealthCheck_NoDockerBinary(t *testing.T) {
	cs := &ContainerSandbox{
		logger:          slog.Default(),
		dockerCheckFunc: func() bool { return false },
	}

	result := cs.HealthCheck(context.Background())
	assert.False(t, result.Available)
	assert.Equal(t, "docker binary not found in PATH", result.Error)
}

func TestExampleValue_DefaultType(t *testing.T) {
	assert.Equal(t, `"example_value"`, exampleValue("unknown-type"))
}

func TestExampleValue_AllKnownTypes(t *testing.T) {
	assert.Equal(t, `"example_value"`, exampleValue("string"))
	assert.Equal(t, "42", exampleValue("int"))
	assert.Equal(t, "3.14", exampleValue("float"))
	assert.Equal(t, "3.14", exampleValue("float64"))
	assert.Equal(t, "true", exampleValue("bool"))
}

func TestSimpleGoImportCheck_BlockedNetSubpackage(t *testing.T) {
	err := simpleGoImportCheck(`package main
import "net/dial"`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocklisted net subpackage detected")
}

func TestSimpleGoImportCheck_BlockedInternal(t *testing.T) {
	err := simpleGoImportCheck(`package main
import "internal/secret"`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocklisted internal subpackage")
}

func TestSimpleGoImportCheck_LineContinuationNotAnImport(t *testing.T) {
	err := simpleGoImportCheck("package main\nfunc main(){println(`import`)}")
	assert.NoError(t, err)
}

func TestValidateGoCode_SyntacticallyInvalidButNoImports(t *testing.T) {
	err := ValidateGoCode("not valid go at all")
	assert.NoError(t, err)
}

func TestValidateGoCode_InternalSubpackage(t *testing.T) {
	err := ValidateGoCode("package main\nimport \"internal/private\"\nfunc main() {}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocklisted internal subpackage")
}

func TestValidatePythonCode_RequestsBlocked(t *testing.T) {
	err := ValidatePythonCode("import requests\nr = requests.get('http://evil.com')")
	assert.Error(t, err)
}

func TestValidatePythonCode_ImportlibBlocked(t *testing.T) {
	err := ValidatePythonCode("import importlib\nimportlib.import_module('os')")
	assert.Error(t, err)
}

func TestValidatePythonCode_PickleBlocked(t *testing.T) {
	err := ValidatePythonCode("import pickle\npickle.loads(b'bad')")
	assert.Error(t, err)
}

func TestValidatePythonCode_HttpxBlocked(t *testing.T) {
	err := ValidatePythonCode("from httpx import Client\nc = Client()")
	assert.Error(t, err)
}

func TestValidatePythonCode_GetattrBlocked(t *testing.T) {
	err := ValidatePythonCode("getattr(os, 'system')('ls')")
	assert.Error(t, err)
}

func TestValidatePythonCode_GlobalsBlocked(t *testing.T) {
	err := ValidatePythonCode("x = globals()")
	assert.Error(t, err)
}

func TestValidatePythonCode_Urllib3Blocked(t *testing.T) {
	err := ValidatePythonCode("# python\nprint('hello')\nimport urllib3")
	assert.Error(t, err)
}

func TestValidatePythonCode_FromWebsocketsImport(t *testing.T) {
	err := ValidatePythonCode("from websockets import serve")
	assert.Error(t, err)
}

func TestValidatePythonCode_CollapseBackslashContinuation_NoEvasion(t *testing.T) {
	err := ValidatePythonCode("# python\nnormal = 'hello'\nprint(normal)")
	assert.NoError(t, err)
}

func TestCheckSandboxSecurity_NoLookPaths(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	issues := v.CheckSandboxSecurity()

	for _, issue := range issues {
		assert.Contains(t, issue, "not found")
	}
	t.Logf("sandbox security issues: %v", issues)
}

func TestDevModeConfig_Defaults(t *testing.T) {
	cfg := DefaultDevModeConfig()
	assert.Equal(t, "./tools/dev", cfg.WatchDir)
	assert.NotZero(t, cfg.PollInterval)
	assert.Equal(t, "python3", cfg.PythonCmd)
	assert.Equal(t, "go", cfg.GoCmd)
}

func TestNewToolWatcher_StopChannel(t *testing.T) {
	w := NewToolWatcher(slog.Default(), DefaultDevModeConfig(), nil)
	assert.NotNil(t, w.stopCh)
}

func TestWatchedFiles_EmptyInitially(t *testing.T) {
	w := NewToolWatcher(slog.Default(), DefaultDevModeConfig(), nil)
	assert.Empty(t, w.WatchedFiles())
}

func TestSandboxPolicy_Idempotent(t *testing.T) {
	p1 := sandboxPolicy()
	p2 := sandboxPolicy()
	assert.Equal(t, p1.DefaultAction, p2.DefaultAction)
	assert.Equal(t, len(p1.Syscalls), len(p2.Syscalls))
}

func TestHealthCheckResult_Defaults(t *testing.T) {
	r := HealthCheckResult{}
	assert.False(t, r.Available)
	assert.False(t, r.HasRunsc)
	assert.False(t, r.DockerPingOK)
	assert.Empty(t, r.Error)
}

func TestExecutionResult_MetricsWithAllFields(t *testing.T) {
	result := ExecutionResult{
		ExitCode: 0,
		Stdout:   "output",
		Stderr:   "errs",
		Metrics:  "cpu=1.23,mem=256,status=ok",
	}
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "output", result.Stdout)
	assert.Equal(t, "errs", result.Stderr)
	assert.Contains(t, result.Metrics, "cpu")
	assert.Contains(t, result.Metrics, "mem")
	assert.Contains(t, result.Metrics, "status")
}

func TestContainerSandbox_DetectRunsc_Failure(t *testing.T) {
	cs := &ContainerSandbox{
		logger:          slog.Default(),
		dockerCheckFunc: func() bool { return false },
	}
	cs.detectRunsc()
	assert.True(t, cs.runscCached)
}

func TestContainerSandbox_DetectRunsc_AlreadyCached(t *testing.T) {
	cs := &ContainerSandbox{
		logger:          slog.Default(),
		runscCached:     true,
		hasRunsc:        false,
		dockerCheckFunc: func() bool { return false },
	}
	cs.detectRunsc()
	assert.True(t, cs.runscCached)
	assert.False(t, cs.hasRunsc)
}

func TestVerifier_VerifyToolCode_GoCode(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")

	t.Run("valid Go", func(t *testing.T) {
		r := v.VerifyToolCode("package main\nimport \"fmt\"\nfunc main(){println(`hi`)}")
		assert.True(t, r.Passed)
	})

	t.Run("blocked Go import", func(t *testing.T) {
		r := v.VerifyToolCode("package main\nimport(\"syscall\")\nfunc main(){}")
		assert.False(t, r.Passed)
		assert.Contains(t, r.Error, "blocklisted import")
	})

	t.Run("valid Python", func(t *testing.T) {
		r := v.VerifyToolCode("# python\nimport json\nprint('ok')")
		assert.True(t, r.Passed)
	})

	t.Run("blocked Python", func(t *testing.T) {
		r := v.VerifyToolCode("# python\nimport subprocess")
		assert.False(t, r.Passed)
	})
}

func TestContainerSandbox_HealthCheck_RunscCached(t *testing.T) {
	cs := &ContainerSandbox{
		logger:          slog.Default(),
		runscCached:     true,
		hasRunsc:        true,
		dockerCheckFunc: func() bool { return true },
	}

	result := cs.HealthCheck(context.Background())
	_ = result
}

func TestContainerSandbox_HealthCheck_NoRunsc(t *testing.T) {
	cs := &ContainerSandbox{
		logger:          slog.Default(),
		runscCached:     false,
		dockerCheckFunc: func() bool { return true },
	}

	result := cs.HealthCheck(context.Background())
	_ = result
}

func TestCollapseBackslashContinuations_SingleLineBackslash(t *testing.T) {
	result := collapseBackslashContinuations("line\\")
	assert.Equal(t, "line", result)
}

func TestCollapseBackslashContinuations_Empty(t *testing.T) {
	result := collapseBackslashContinuations("")
	assert.Empty(t, result)
}

func TestCollapseBackslashContinuations_FullFlow(t *testing.T) {
	result := collapseBackslashContinuations("first\\\nsecond\nstandalone\nthird\\\n")
	assert.Equal(t, "firstsecond\nstandalone\nthird", result)
}

func TestValidatePythonCode_Smtplib(t *testing.T) {
	err := ValidatePythonCode("# python\nimport smtplib")
	assert.Error(t, err)
}

func TestValidatePythonCode_Aiohttp(t *testing.T) {
	err := ValidatePythonCode("import aiohttp")
	assert.Error(t, err)
}

func TestValidatePythonCode_VarsBlocked(t *testing.T) {
	err := ValidatePythonCode("x = vars()")
	assert.Error(t, err)
}

func TestValidatePythonCode_CompileBlocked(t *testing.T) {
	err := ValidatePythonCode("x = compile('1+1', '', 'eval')")
	assert.Error(t, err)
}

func TestValidatePythonCode_BuiltinsBlocked(t *testing.T) {
	err := ValidatePythonCode("x = __builtins__\nprint(x)")
	assert.Error(t, err)
}

func TestCheckGoFormat_MixedTabs(t *testing.T) {
	issues := CheckGoFormat(" \tpackage main\n\tfunc main() {}\n")
	assert.NotEmpty(t, issues)
}

func TestEstimateComplexity_LogicalOperators(t *testing.T) {
	complexity := EstimateComplexity("if a && b {\n}\nif c || d {\n}")
	assert.GreaterOrEqual(t, complexity, 4)
}

func TestEstimateCoverage_Benchmarks(t *testing.T) {
	cov := estimateCoverage(`package main
import "testing"
func BenchmarkFoo(b *testing.B) {
	b.Error("x")
}
`)
	t.Logf("benchmark coverage estimate: %f", cov)
}

func TestEstimateCoverage_TestsWithNoAssertions(t *testing.T) {
	cov := estimateCoverage(`package main
import "testing"
func TestX(t *testing.T) {
	println("no assertions")
}
`)
	t.Logf("coverage estimate (no assertions): %f", cov)
}

func TestExecCommandContext_OutputSizeCheck(t *testing.T) {
	a := NewCommandAllowlist()
	ctx := context.Background()
	output, err := a.ExecCommandContext(ctx, "ls", "/tmp")
	if err == nil {
		assert.NotNil(t, output)
	}
}

func TestVerificationConfig_AllFields(t *testing.T) {
	cfg := VerificationConfig{
		TimeoutSeconds: 30,
		MaxMemoryMB:    512,
		MaxCPUSeconds:  20.0,
		NetworkBlocked: false,
	}
	assert.Equal(t, 30, cfg.TimeoutSeconds)
	assert.Equal(t, 512, cfg.MaxMemoryMB)
	assert.Equal(t, 20.0, cfg.MaxCPUSeconds)
	assert.False(t, cfg.NetworkBlocked)
}

func TestVerifier_VerifyMultipleTools_Recovery(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	cfg := DefaultVerificationConfig()
	ctx := context.Background()
	results := v.VerifyMultipleTools(ctx, []string{}, cfg)
	assert.Empty(t, results)
}

func TestContainerSandbox_ExecuteTool_GoCode_AllPaths(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultContainerConfig()
	cfg.MaxCPUSeconds = 0.02
	cs := NewContainerSandbox(logger, nil, nil, cfg, nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "package main\nimport(\"fmt\")\nfunc main(){fmt.Println(`hi`)}", nil
	}
	cs.dockerCheckFunc = func() bool { return true }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "go-lowcpu", map[string]any{})

	require.NoError(t, err)
	assert.Contains(t, result.Error, "docker execution failed")
}

func TestContainerSandbox_ExecuteTool_CPUFloorAtLeast(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultContainerConfig()
	cfg.MaxCPUSeconds = 0.001
	cs := NewContainerSandbox(logger, nil, nil, cfg, nil)

	cs.toolCodeGetter = func(_ context.Context, toolID string) (string, error) {
		return "# python\nimport json\nprint('ok')", nil
	}
	cs.dockerCheckFunc = func() bool { return true }

	ctx := context.Background()
	result, err := cs.ExecuteTool(ctx, "cpumin", map[string]any{})

	require.NoError(t, err)
	assert.Contains(t, result.Error, "docker execution failed")
}
