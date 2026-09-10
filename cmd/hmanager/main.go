package main

import (
	"fmt"
	"os"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
	"github.com/hrunner/hrunner/pkg/ui"
)

func main() {
	baseDir := registry.DefaultHrunnerDir()

	reg, err := registry.NewRegistry(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load registry: %v\n", err)
		os.Exit(1)
	}

	runtimes, err := runtime.NewRuntimeStore(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load runtimes: %v\n", err)
		os.Exit(1)
	}

	pypi := packages.NewPyPIClient("")
	pkgStore, err := packages.NewPackageStore(baseDir, pypi)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load package store: %v\n", err)
		os.Exit(1)
	}

	storageMgr, err := storage.NewStorageManager(baseDir, reg, pkgStore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load storage manager: %v\n", err)
		os.Exit(1)
	}

	manager := ui.NewNativeManager(reg, runtimes, pkgStore, storageMgr)
	manager.RunInteractiveLoop()
}
