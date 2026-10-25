package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/AgentNex/delvo/internal/app"
	"github.com/AgentNex/delvo/internal/config"
	"github.com/AgentNex/delvo/internal/index"
	"github.com/AgentNex/delvo/internal/scanner"
)

const (
	Version = "0.1.0"
	Banner  = `
  ██████╗ ███████╗██╗     ██╗   ██╗ ██████╗ 
  ██╔══██╗██╔════╝██║     ██║   ██║██╔═══██╗
  ██║  ██║█████╗  ██║     ██║   ██║██║   ██║
  ██║  ██║██╔══╝  ██║     ╚██╗ ██╔╝██║   ██║
  ██████╔╝███████╗███████╗ ╚████╔╝ ╚██████╔╝
  ╚═════╝ ╚══════╝╚══════╝  ╚═══╝   ╚═════╝ 
`
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("Delvo CLI v%s\n", Version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "roots":
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Configured Roots:")
			for _, r := range cfg.Roots {
				fmt.Printf("  • %s\n", r)
			}
			fmt.Printf("Database: %s\n", cfg.DBPath)
			return
		case "rebuild":
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}
			idx, err := index.New(cfg.DBPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
				os.Exit(1)
			}
			defer idx.Close()

			fmt.Println("Rebuilding index database...")
			if err := idx.Rebuild(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "Rebuild failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Index successfully rebuilt.")
			return
		case "scan":
			runCliScan(os.Args[2:])
			return
		}
	}

	// Interactive TUI mode
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Delvo error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(Banner)
	fmt.Printf("Delvo CLI v%s — High-Performance Go Terminal File Manager & Disk-Usage Explorer\n\n", Version)
	fmt.Println("USAGE:")
	fmt.Println("  dvo                    Launch the interactive full-screen TUI")
	fmt.Println("  dvo scan [path]        Run command-line parallel scan and report disk stats")
	fmt.Println("  dvo roots              List configured roots and database location")
	fmt.Println("  dvo rebuild            Clear and rebuild the local SQLite index")
	fmt.Println("  dvo version            Print version information")
	fmt.Println("  dvo help               Show this help message")
	fmt.Println("")
	fmt.Println("INTERACTIVE KEYBINDINGS:")
	fmt.Println("  ↑ / k, ↓ / j           Navigate entries")
	fmt.Println("  Enter                  Open directory / Navigate")
	fmt.Println("  Backspace / h          Go to parent directory")
	fmt.Println("  Space                  Toggle select current item")
	fmt.Println("  a                      Select all / Invert selection")
	fmt.Println("  c                      Clear all selections")
	fmt.Println("  /                      Search (fuzzy filename & path matching)")
	fmt.Println("  s                      Cycle sorting (Name, Size, Date, Extension)")
	fmt.Println("  d                      Safe delete (with confirmation & trash support)")
	fmt.Println("  1, 2, 3, 4             Switch tabs (Browse, Search, Disk Usage, Filters)")
	fmt.Println("  i                      Toggle item details inspection modal")
	fmt.Println("  r                      Rescan current root / directory")
	fmt.Println("  q / Esc                Quit application")
}

func runCliScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	workers := fs.Int("workers", 4, "Number of parallel scanner workers")
	_ = fs.Parse(args)

	targetRoots := fs.Args()
	if len(targetRoots) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error obtaining cwd: %v\n", err)
			os.Exit(1)
		}
		targetRoots = []string{cwd}
	}

	for i, r := range targetRoots {
		abs, err := filepath.Abs(r)
		if err == nil {
			targetRoots[i] = abs
		}
	}

	fmt.Printf("Scanning roots: %v with %d workers...\n", targetRoots, *workers)
	start := time.Now()

	s := scanner.New(scanner.ScanOptions{
		Roots:     targetRoots,
		Workers:   *workers,
		BatchSize: 500,
	})

	res, err := s.Scan(context.Background(), func(p scanner.ScanProgress) {
		fmt.Printf("\rScanned %d files (%d dirs) — %s...", p.FilesScanned, p.DirsScanned, scanner.FormatSize(p.BytesScanned))
	}, nil)

	fmt.Println("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan completed with error: %v\n", err)
	}

	if res != nil {
		fmt.Printf("\n✓ Scan complete in %v\n", time.Since(start).Round(time.Millisecond))
		fmt.Printf("  • Total Files:  %d\n", res.Progress.FilesScanned)
		fmt.Printf("  • Total Dirs:   %d\n", res.Progress.DirsScanned)
		fmt.Printf("  • Total Size:   %s\n", scanner.FormatSize(res.Progress.BytesScanned))

		// Sort top 10 largest entries
		entries := res.Entries
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Size > entries[j].Size
		})

		fmt.Println("\nTop 10 Largest Items:")
		count := 10
		if len(entries) < count {
			count = len(entries)
		}
		for i := 0; i < count; i++ {
			e := entries[i]
			tag := "[FILE]"
			if e.IsDir {
				tag = "[DIR] "
			}
			fmt.Printf("  %2d. %s %10s  %s\n", i+1, tag, scanner.FormatSize(e.Size), e.Path)
		}
	}
}
