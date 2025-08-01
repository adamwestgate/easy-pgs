// build_chip_panels.go
// -----------------------------------------------------------------------------
// Build chip-specific reference panels (23andMe and Ancestry only) from a large
// unfiltered genome, using PLINK 2.
//
// * .range files must sit in setup/chip_refs/ and be named 23andme.range,
//   ancestry.range, etc.   Any other .range files (e.g. combined.range) are
//   ignored.
// * Output pfiles land in setup/chip_panels/<chip>/.
//
// Usage examples (run from repo root):
//   go run ./setup                 # builds both panels
//   go run ./setup --chip ancestry # just Ancestry
//   go build -o build_chip_panels ./setup && ./build_chip_panels --threads 8
// -----------------------------------------------------------------------------
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ───────────────────────────── Configurable paths ─────────────────────────────
const (
	GenomeDir   = "setup/genome"          // unfiltered .pgen/.pvar/.psam live here
	ChipRefsDir = "setup/chip_refs"       // *.range files
	OutRoot     = "setup/chip_panels"     // output panels
)

// ─────────────────────────────── CLI flags ───────────────────────────────────
var (
	chipFlag = flag.String("chip", "all", "Chip to build: 23andme | ancestry | all")
	threads  = flag.Int("threads", runtime.NumCPU(), "CPU threads for PLINK")
	memory   = flag.Int("mem-mb", 32000, "Memory limit (MB) for PLINK")
)

// ────────────────────────────── main ─────────────────────────────────────────
func main() {
	flag.Parse()

	root, _ := os.Getwd() // repo root = current working dir

	gDir   := filepath.Join(root, GenomeDir)
	refsDir := filepath.Join(root, ChipRefsDir)
	outRoot := filepath.Join(root, OutRoot)

	prefix := findGenomePrefix(gDir)
	exe    := detectPlink()

	// ---- gather allowed .range files ---------------------------------------
	allowed := map[string]bool{"23andme": true, "ancestry": true}

	all, _ := filepath.Glob(filepath.Join(refsDir, "*.range"))
	var rangeFiles []string
	for _, rf := range all {
		tag := strings.ToLower(strings.TrimSuffix(filepath.Base(rf), ".range"))
		if allowed[tag] {
			rangeFiles = append(rangeFiles, rf)
		}
	}
	if len(rangeFiles) == 0 {
		log.Fatalf("no 23andme/ancestry .range files found in %s", refsDir)
	}

	// optional --chip filter
	if tgt := strings.ToLower(*chipFlag); tgt != "all" {
		var selected []string
		for _, rf := range rangeFiles {
			if strings.TrimSuffix(filepath.Base(rf), ".range") == tgt {
				selected = []string{rf}
			}
		}
		if len(selected) == 0 {
			log.Fatalf("chip '%s' not found (must be 23andme or ancestry)", tgt)
		}
		rangeFiles = selected
	}

	// ------------------------------------------------------------------------
	os.MkdirAll(outRoot, 0o755)

	for _, rf := range rangeFiles {
		tag    := strings.TrimSuffix(filepath.Base(rf), ".range")
		outDir := filepath.Join(outRoot, tag)
		os.MkdirAll(outDir, 0o755)

		log.Printf("\n=== Building panel for %s ===", tag)
		buildPanel(exe, prefix, rf, outDir, tag)
	}
}

// buildPanel executes the 3-step PLINK pipeline for one chip type.
func buildPanel(exe, prefix, rangeFile, outDir, tag string) {
	tmp := filepath.Join(outDir, "tmp_"+tag)

	run(exe, []string{
		"--pfile", prefix,
		"--extract", "range", rangeFile,
		"--make-pgen",
		"--out", tmp + "_step1",
		"--threads", itoa(*threads),
		"--memory", itoa(*memory),
	})

	final := filepath.Join(outDir, tag)
	run(exe, []string{
		"--pfile", tmp + "_step1",
		"--set-missing-var-ids", "@:#$1_$2",
		"--rm-dup", "exclude-all",
		"--make-pgen",
		"--out", final,
		"--threads", itoa(*threads),
		"--memory", itoa(*memory),
	})

	run(exe, []string{
		"--pfile", final,
		"--freq",
		"--out", final,
		"--threads", itoa(*threads),
		"--memory", itoa(*memory),
	})

	cleanup(tmp + "_step1")
	log.Printf("✔ Done: %s.[pgen|pvar|psam] (+ .afreq)", final)
}

// ───────────────────────────── helper functions ─────────────────────────────
func findGenomePrefix(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("cannot read genome dir %s: %v", dir, err)
	}
	var prefixes []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pgen") {
			prefixes = append(prefixes, strings.TrimSuffix(filepath.Join(dir, e.Name()), ".pgen"))
		}
	}
	if len(prefixes) == 0 {
		log.Fatalf("no .pgen file found in %s", dir)
	}
	sort.Strings(prefixes)
	if len(prefixes) > 1 {
		log.Printf("⚠ Multiple .pgen files found; using %s", filepath.Base(prefixes[0]))
	}
	mustExist(prefixes[0] + ".pvar")
	mustExist(prefixes[0] + ".psam")
	return prefixes[0]
}

func run(exe string, args []string) {
	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("→ %s %v", exe, args)
	if err := cmd.Run(); err != nil {
		log.Fatalf("plink failed: %v", err)
	}
}

func detectPlink() string {
	if p := os.Getenv("PLINK_PATH"); p != "" {
		return p
	}
	exe := "plink2"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if path, err := exec.LookPath(exe); err == nil {
		return path
	}
	log.Fatalf("plink2 not found; install it or set PLINK_PATH")
	return ""
}

func mustExist(p string) {
	if _, err := os.Stat(p); err != nil {
		log.Fatalf("required file not found: %s", p)
	}
}

func cleanup(prefix string) {
	for _, ext := range []string{".pgen", ".pvar", ".psam"} {
		os.Remove(prefix + ext)
	}
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
