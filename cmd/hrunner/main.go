package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/protocol"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/resolver"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
	"github.com/hrunner/hrunner/pkg/ui"
)

type HrunnerCore struct {
	baseDir      string
	reg          *registry.RegistryStore
	runtimes     *runtime.RuntimeStore
	pkgStore     *packages.PackageStore
	resolver     *resolver.Resolver
	storageMgr   *storage.StorageManager
	ui           *ui.UI
	server       *protocol.Server
}

func NewHrunnerCore(baseDir string) (*HrunnerCore, error) {
	if baseDir == "" {
		baseDir = registry.DefaultHrunnerDir()
	}

	reg, err := registry.NewRegistry(baseDir)
	if err != nil {
		return nil, fmt.Errorf("registry init failed: %w", err)
	}

	runtimes, err := runtime.NewRuntimeStore(baseDir)
	if err != nil {
		return nil, fmt.Errorf("runtimes init failed: %w", err)
	}

	pypi := packages.NewPyPIClient("")
	pkgStore, err := packages.NewPackageStore(baseDir, pypi)
	if err != nil {
		return nil, fmt.Errorf("package store init failed: %w", err)
	}

	res := resolver.NewResolver(pkgStore, pypi)
	storageMgr, err := storage.NewStorageManager(baseDir, reg, pkgStore)
	if err != nil {
		return nil, fmt.Errorf("storage manager init failed: %w", err)
	}

	core := &HrunnerCore{
		baseDir:    baseDir,
		reg:        reg,
		runtimes:   runtimes,
		pkgStore:   pkgStore,
		resolver:   res,
		storageMgr: storageMgr,
		ui:         ui.NewUI(),
	}

	return core, nil
}

func (c *HrunnerCore) HandleLaunchRequest(req *protocol.LaunchRequest, conn *protocol.ClientConn) (*protocol.LaunchResponse, error) {
	c.server.AddOperation()
	defer c.server.DoneOperation()

	// 1. Check Python runtime
	rt, hasExact, err := c.runtimes.Find(req.Python, false)
	if err != nil {
		return nil, fmt.Errorf("python check failed: %w", err)
	}

	var compatibleRt *runtime.RuntimeInfo
	var hasCompat bool
	if !hasExact {
		compatibleRt, hasCompat, _ = c.runtimes.Find(req.Python, true)
	}

	needInstallPython := false
	var selectedRuntime *runtime.RuntimeInfo

	if hasExact {
		selectedRuntime = rt
	} else if hasCompat && req.AllowCompatiblePython {
		selectedRuntime = compatibleRt
	} else if hasCompat && !req.AllowCompatiblePython {
		// Ask user whether to install exact or use compatible
		choice, confirmed := c.ui.AskPythonRuntime(req.Python, compatibleRt.Version.String())
		if !confirmed {
			return &protocol.LaunchResponse{
				BaseMessage:  protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchResponse},
				Status:       protocol.StatusError,
				ErrorMessage: "installation cancelled by user",
			}, nil
		}
		if choice == "use_compatible" {
			selectedRuntime = compatibleRt
		} else {
			needInstallPython = true
		}
	} else {
		needInstallPython = true
	}

	// 2. Parse version for package resolution
	pyVer, err := runtime.ParseVersion(req.Python)
	if err != nil {
		return nil, fmt.Errorf("invalid python version: %w", err)
	}

	// 3. Resolve dependencies
	plan, err := c.resolver.Resolve(req.Dependencies, pyVer)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// 4. Determine if installation is required
	needsInstall := needInstallPython || len(plan.MissingPackages) > 0

	var missingPkgInfos []protocol.MissingPackageInfo
	for _, mp := range plan.MissingPackages {
		missingPkgInfos = append(missingPkgInfos, protocol.MissingPackageInfo{
			Name:          mp.Name,
			Version:       mp.Version,
			DownloadBytes: mp.DownloadBytes,
		})
	}

	totalDownloadBytes := plan.TotalDownloadBytes
	if needInstallPython {
		totalDownloadBytes += 15 * 1024 * 1024 // Estimated ~15MB for embeddable python zip
	}

	// Register or update app as discovered
	_ = c.reg.Register(&registry.ApplicationRecord{
		ApplicationID:  req.ApplicationID,
		Name:           req.Name,
		Version:        req.Version,
		ExecutablePath: req.ExecutablePath,
		PythonVersion:  req.Python,
		Dependencies:   plan.AllPackages,
		Entrypoint:     req.Entrypoint,
		State:          registry.StateDiscovered,
	})

	if needsInstall {
		// Show confirmation dialog to user
		confirmed := c.ui.ConfirmInstallation(req.Name, req.Python, needInstallPython, missingPkgInfos, totalDownloadBytes)
		if !confirmed {
			_ = c.reg.UpdateState(req.ApplicationID, registry.StateBroken)
			return &protocol.LaunchResponse{
				BaseMessage:  protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchResponse},
				Status:       protocol.StatusError,
				ErrorMessage: "installation cancelled by user",
			}, nil
		}

		_ = c.reg.UpdateState(req.ApplicationID, registry.StateInstalling)

		// Download & install Python if needed
		if needInstallPython {
			_ = conn.SendMessage(&protocol.InstallProgress{
				BaseMessage: protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgInstallProgress},
				Step:        protocol.StepDownloadingPython,
				Message:     fmt.Sprintf("Downloading Python %s...", req.Python),
			})

			installedRt, err := c.runtimes.DownloadAndInstall(req.Python, func(dl, tot int64, pct float64) {
				_ = conn.SendMessage(&protocol.InstallProgress{
					BaseMessage:    protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgInstallProgress},
					Step:           protocol.StepDownloadingPython,
					BytesCompleted: dl,
					TotalBytes:     tot,
					Percent:        pct,
					Message:        fmt.Sprintf("Downloading Python %s: %.1f%%", req.Python, pct),
				})
			})
			if err != nil {
				_ = c.reg.UpdateState(req.ApplicationID, registry.StateBroken)
				return nil, fmt.Errorf("failed to install Python: %w", err)
			}
			selectedRuntime = installedRt
		}

		// Download & install missing packages into shared pool
		totalItems := len(plan.MissingPackages)
		for idx, mp := range plan.MissingPackages {
			_ = conn.SendMessage(&protocol.InstallProgress{
				BaseMessage: protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgInstallProgress},
				Step:        protocol.StepDownloadingPackages,
				ItemName:    mp.Name,
				ItemIndex:   idx + 1,
				TotalItems:  totalItems,
				Message:     fmt.Sprintf("Installing %s %s (%d/%d)...", mp.Name, mp.Version, idx+1, totalItems),
			})

			_, err := c.pkgStore.EnsurePackage(mp.Name, mp.Version, pyVer, func(dl, tot int64, pct float64) {
				_ = conn.SendMessage(&protocol.InstallProgress{
					BaseMessage:    protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgInstallProgress},
					Step:           protocol.StepDownloadingPackages,
					ItemName:       mp.Name,
					ItemIndex:      idx + 1,
					TotalItems:     totalItems,
					BytesCompleted: dl,
					TotalBytes:     tot,
					Percent:        pct,
					Message:        fmt.Sprintf("Downloading %s: %.1f%%", mp.Name, pct),
				})
			})
			if err != nil {
				_ = c.reg.UpdateState(req.ApplicationID, registry.StateBroken)
				return nil, fmt.Errorf("failed to install package %s %s: %w", mp.Name, mp.Version, err)
			}
		}

		_ = c.reg.UpdateState(req.ApplicationID, registry.StateReady)

		// Show completion dialog
		launchNow := c.ui.ShowInstallComplete(req.Name)
		if !launchNow {
			return &protocol.LaunchResponse{
				BaseMessage: protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchResponse},
				Status:      protocol.StatusReady,
			}, nil
		}
	}

	// Prepare package paths for execution
	var pkgPaths []string
	for pkg, ver := range plan.AllPackages {
		pkgPaths = append(pkgPaths, c.pkgStore.GetPackageDir(pkg, ver))
	}

	// Send LaunchReady details
	_ = conn.SendMessage(&protocol.LaunchReady{
		BaseMessage:   protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchReady},
		PythonExePath: selectedRuntime.PythonExe,
		PackagePaths:  pkgPaths,
		Entrypoint:    req.Entrypoint,
	})

	_ = c.reg.UpdateLastLaunched(req.ApplicationID)

	return &protocol.LaunchResponse{
		BaseMessage:     protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchResponse},
		Status:          protocol.StatusReady,
		PythonAvailable: true,
	}, nil
}

func (c *HrunnerCore) OnClientConnected()    {}
func (c *HrunnerCore) OnClientDisconnected() {}

func main() {
	pipeServerFlag := flag.Bool("pipe-server", false, "Run as on-demand Named Pipe IPC server")
	managerFlag := flag.Bool("manager", false, "Launch Hrunner Manager GUI")
	flag.Parse()

	core, err := NewHrunnerCore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Hrunner: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) > 0 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case "apps", "list":
			apps := core.reg.List()
			fmt.Printf("Installed Applications (%d):\n", len(apps))
			for _, a := range apps {
				fmt.Printf("  • %s v%s (ID: %s, Python: %s, State: %s, Deps: %d)\n",
					a.Name, a.Version, a.ApplicationID, a.PythonVersion, a.State, len(a.Dependencies))
			}
			return
		case "runtimes":
			rts, _ := core.runtimes.List()
			fmt.Printf("Installed Python Runtimes (%d):\n", len(rts))
			for _, r := range rts {
				fmt.Printf("  • Python %s (%s, %s)\n", r.Version.String(), r.Path, ui.FormatBytes(r.SizeBytes))
			}
			return
		case "packages":
			usages, _ := core.storageMgr.GetPackageUsageMap()
			fmt.Printf("Shared Package Pool (%d):\n", len(usages))
			for _, u := range usages {
				status := "SHARED"
				if u.RefCount == 0 {
					status = "UNUSED"
				}
				fmt.Printf("  • %s %s [%s] (Used by %d apps, %s)\n",
					u.Name, u.Version, status, u.RefCount, ui.FormatBytes(u.SizeBytes))
			}
			return
		case "clean", "cleanup":
			deleted, freed, err := core.storageMgr.CleanUnusedPackages()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Clean failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Cleaned %d unused package(s), freed %s\n", deleted, ui.FormatBytes(freed))
			return
		case "storage":
			sum, _ := core.storageMgr.GetSummary()
			fmt.Printf("Hrunner Storage Usage:\n")
			fmt.Printf("  Python Runtimes:  %s\n", ui.FormatBytes(sum.RuntimesBytes))
			fmt.Printf("  Shared Packages:  %s\n", ui.FormatBytes(sum.PackagesBytes))
			fmt.Printf("  Applications:     %s\n", ui.FormatBytes(sum.ApplicationsBytes))
			fmt.Printf("  Download Cache:   %s\n", ui.FormatBytes(sum.CacheBytes))
			fmt.Printf("  -------------------------\n")
			fmt.Printf("  Total:            %s\n", ui.FormatBytes(sum.TotalBytes))
			return
		case "remove":
			if len(args) < 2 {
				fmt.Println("Usage: hrunner remove <app_id> [--clean-unused]")
				return
			}
			appID := args[1]
			cleanUnused := len(args) > 2 && args[2] == "--clean-unused"
			if err := core.storageMgr.RemoveApplicationAndCleanup(appID, cleanUnused); err != nil {
				fmt.Fprintf(os.Stderr, "Remove failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Application %s removed successfully.\n", appID)
			return
		case "manager":
			*managerFlag = true
		default:
			fmt.Printf("Unknown command '%s'. Available: apps, runtimes, packages, clean, storage, remove, manager\n", cmd)
			return
		}
	}

	if *managerFlag || (!*pipeServerFlag && len(args) == 0) {
		nativeMgr := ui.NewNativeManager(core.reg, core.runtimes, core.pkgStore, core.storageMgr)
		nativeMgr.RunInteractiveLoop()
		return
	}

	if *pipeServerFlag {
		// Run on-demand IPC server with auto-shutdown
		server := protocol.NewServer(protocol.DefaultPipeName, 5*time.Second, core)
		core.server = server
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start IPC server: %v\n", err)
			os.Exit(1)
		}

		server.WaitShutdown()
	}
}
