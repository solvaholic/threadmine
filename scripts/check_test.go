package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFormatting(t *testing.T) {
	script, err := filepath.Abs("check-fmt.sh")
	if err != nil {
		t.Fatal(err)
	}
	formatter := filepath.Join(runtime.GOROOT(), "bin", "gofmt")
	for _, test := range []struct {
		name, source, formatter string
		fail                    bool
	}{
		{"formatted", "package fixture\n", formatter, false},
		{"unformatted", "package fixture\nfunc example( ){ }\n", formatter, true},
		{"invalid", "not Go source\n", formatter, true},
		{"missing formatter", "package fixture\n", "/no/such/gofmt", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "new source.go")
			writeFile(t, source, test.source)
			writeFile(t, filepath.Join(dir, "vendor", "bad.go"), "not Go source")
			writeFile(t, filepath.Join(dir, ".git", "bad.go"), "not Go source")
			cmd := exec.Command("sh", script, test.formatter)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if (err != nil) != test.fail {
				t.Fatalf("error = %v, want failure %v: %s", err, test.fail, output)
			}
			if test.name == "unformatted" && !strings.Contains(string(output), "new source.go") {
				t.Errorf("missing offending file: %s", output)
			}
			after, err := os.ReadFile(source)
			if err != nil || string(after) != test.source {
				t.Fatalf("format check changed source: %q, %v", after, err)
			}
		})
	}
}

func TestMakeChecks(t *testing.T) {
	makefile, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("check-fmt.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, target, cgo, failCommand, goCommand, want string
		fail                                            bool
	}{
		{"aggregate", "check", "1", "", "", "-race ./...", false},
		{"bad formatting", "check", "1", "format", "", "new source.go", true},
		{"test failure", "check", "1", "test", "", "intentional test failure", true},
		{"vet failure", "check", "1", "vet", "", "intentional vet failure", true},
		{"module failure", "check", "1", "mod", "", "intentional mod failure", true},
		{"build failure", "check", "1", "build", "", "intentional build failure", true},
		{"smoke failure", "smoke", "1", "smoke", "", "intentional smoke failure", true},
		{"disabled CGO", "test", "0", "", "", "CGO_ENABLED=1", true},
		{"missing Go", "test", "1", "", "/no/such/go", "Go executable", true},
		{"missing compiler", "test", "1", "compiler", "", "C compiler", true},
		{"ordinary test", "test", "1", "", "", "test ", false},
		{"build", "build", "1", "", "", "./cmd/mine", false},
		{"install", "install", "1", "", "", "install ", false},
		{"run", "run", "1", "", "", "Running mine", false},
		{"coverage", "test-coverage", "1", "", "", "tool cover -html=coverage.out -o coverage.html", false},
		{"format", "fmt", "1", "", "", "fmt ", false},
		{"clean", "clean", "1", "", "", "Clean complete", false},
		{"help", "help", "1", "", "", "test-coverage", false},
		{"vulnerability scan", "vuln", "1", "", "", "govulncheck@v1.9.0 -tags fts5 -show verbose ./cmd/mine", false},
		{"vulnerability failure", "vuln", "1", "run", "", "intentional run failure", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "Makefile"), string(makefile))
			writeFile(t, filepath.Join(dir, "scripts", "check-fmt.sh"), string(script))
			if test.failCommand == "format" {
				writeFile(t, filepath.Join(dir, "new source.go"), "package fixture\nfunc example( ){ }\n")
			}
			goCommand := filepath.Join(dir, "fake-go")
			writeFile(t, goCommand, `#!/bin/sh
set -eu
if [ "$1" = env ]; then
    case "$2" in
        CGO_ENABLED) printf '%s\n' "$CGO_ENABLED" ;;
        CC) if [ "$FAIL_COMMAND" = compiler ]; then echo /no/such/cc; else echo sh; fi ;;
        GOROOT) printf '%s\n' "$TEST_GOROOT" ;;
    esac
    exit 0
fi
printf '%s\n' "$*"
if [ "$1" = "$FAIL_COMMAND" ]; then
    echo "intentional $1 failure" >&2
    exit 42
fi
if [ "$1" = build ]; then
    previous=
    for arg do
        if [ "$previous" = -o ]; then
            printf '#!/bin/sh\n[ "$1" = --help ] || exit 1\n' > "$arg"
            if [ "$FAIL_COMMAND" = smoke ]; then
                printf 'echo "intentional smoke failure" >&2\nexit 43\n' >> "$arg"
            fi
            chmod +x "$arg"
        fi
        previous=$arg
    done
fi
`)
			if test.goCommand != "" {
				goCommand = test.goCommand
			}
			cmd := exec.Command("make", "--no-print-directory", test.target, "GO="+goCommand, "GOFLAGS=-trimpath -mod=mod", "TAGS=custom")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "CGO_ENABLED="+test.cgo, "FAIL_COMMAND="+test.failCommand, "TEST_GOROOT="+runtime.GOROOT())
			output, err := cmd.CombinedOutput()
			if (err != nil) != test.fail || !strings.Contains(string(output), test.want) {
				t.Fatalf("error = %v, want failure %v, output containing %q:\n%s", err, test.fail, test.want, output)
			}
			if test.name == "aggregate" {
				for _, want := range []string{"mod verify", "vet ", "build ", "-trimpath -mod=mod -tags=custom,fts5 -mod=readonly", "./cmd/mine"} {
					if !strings.Contains(string(output), want) {
						t.Errorf("missing %q:\n%s", want, output)
					}
				}
				if strings.Contains(string(output), "govulncheck") {
					t.Errorf("aggregate ran network scan:\n%s", output)
				}
			}
			if test.name == "ordinary test" && strings.Contains(string(output), "-race") {
				t.Errorf("ordinary test enabled race:\n%s", output)
			}
		})
	}
}
