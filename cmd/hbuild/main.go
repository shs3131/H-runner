package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hrunner/hrunner/pkg/builder"
	"github.com/hrunner/hrunner/pkg/ui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	if cmd != "build" {
		fmt.Printf("Unknown command '%s'. Available: build\n", cmd)
		os.Exit(1)
	}

	buildFlags := flag.NewFlagSet("build", flag.ExitOnError)
	outputFlag := buildFlags.String("output", "", "Output executable path (e.g. MyApplication.exe)")
	nameFlag := buildFlags.String("name", "", "Application display name")
	verFlag := buildFlags.String("version", "1.0.0", "Application version (semver)")
	appIDFlag := buildFlags.String("app-id", "", "Unique application ID (e.g. com.example.myapp)")
	pyFlag := buildFlags.String("python", "3.13.7", "Required Python version (e.g. 3.13.7)")
	compatFlag := buildFlags.Bool("allow-compatible", false, "Allow compatible Python runtime (e.g. 3.13.8 satisfies 3.13.7)")
	entryFlag := buildFlags.String("entrypoint", "", "Entrypoint Python file (e.g. main.py)")
	pubFlag := buildFlags.String("publisher", "", "Publisher name")
	stubFlag := buildFlags.String("launcher-stub", "", "Path to hlauncher.exe stub")

	_ = buildFlags.Parse(os.Args[2:])

	projectPath := "."
	if buildFlags.NArg() > 0 {
		projectPath = buildFlags.Arg(0)
	}

	cfg := &builder.BuildConfig{
		ProjectPath:     projectPath,
		OutputPath:      *outputFlag,
		Name:            *nameFlag,
		Version:         *verFlag,
		ApplicationID:   *appIDFlag,
		PythonVersion:   *pyFlag,
		AllowCompatible: *compatFlag,
		Entrypoint:      *entryFlag,
		Publisher:       *pubFlag,
		LauncherStub:    *stubFlag,
	}

	fmt.Printf("Building Hrunner package for: %s\n", projectPath)
	res, err := builder.Build(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully built %s in %v\n", res.OutputPath, res.Duration)
	fmt.Printf("✓ Final executable size: %s (%d bytes)\n", ui.FormatBytes(res.SizeBytes), res.SizeBytes)
	fmt.Println("  (Contains application code and manifest; zero bundled Python/wheels)")
}

func printUsage() {
	fmt.Println("Hrunner Builder (hbuild)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hbuild build <project_path> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --output, -o <path>       Output executable path (default: <Name>.exe)")
	fmt.Println("  --name <name>             Application name")
	fmt.Println("  --version <ver>           Application version (default: 1.0.0)")
	fmt.Println("  --app-id <id>             Unique application ID (default: com.hrunner.<name>)")
	fmt.Println("  --python <ver>            Python version requirement (default: 3.13.7)")
	fmt.Println("  --allow-compatible        Allow compatible minor Python runtimes")
	fmt.Println("  --entrypoint <file>       Application entrypoint script (default: auto-detected)")
	fmt.Println("  --publisher <pub>         Application publisher")
	fmt.Println("  --launcher-stub <exe>     Custom launcher stub path")
}
