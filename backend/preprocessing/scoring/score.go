// Package scoring handles scoring functionality for kits. 
// Performs end stage preprocessing steps for related files and scores them via plink2.
package scoring

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adamwestgate/easy-pgs/backend/config"
)

func refFreqFor(kitType string) string {
	switch strings.ToLower(kitType) {
	case "ancestry":
		return config.ReferenceFreqAncestry
	case "23andme":
		return config.ReferenceFreq23andme
	default:
		return ""
	}
}

// BatchResult holds the outcome of scoring a single PGS file:
//  - ScorePath: path to the generated .sscore file (empty on error)
//  - Err: any error encountered during scoring
type BatchResult struct {
	ScorePath string
	Err       error
}

// BatchScore locates pgen/pvar/psam files for the associated kit, then 
//  scores them for each PGS file requested by the user.
// Returns a map from trimmed PGS ID to BatchResult.
//  - pfileDir: directory containing .pgen/.pvar files for the kit
//  - kitType: data source name ("ancestry" or "23andme")
//  - scorePaths: list of PGS weight files to apply
//  - pvarDir: optional directory to search for a .pvar file
func BatchScore(pfileDir, kitType string, scorePaths []string, pvarDir string) map[string]BatchResult {
	res := make(map[string]BatchResult, len(scorePaths))

	// locate <prefix>.pgen in pfileDir
	var prefix string
	for _, e := range mustReadDir(pfileDir) {
		if strings.HasSuffix(e.Name(), ".pgen") {
			prefix = strings.TrimSuffix(filepath.Join(pfileDir, e.Name()), ".pgen")
			break
		}
	}
	if prefix == "" {
		err := fmt.Errorf("no .pgen in %s", pfileDir)
		for _, p := range scorePaths {
			res[trimID(p)] = BatchResult{"", err}
		}
		return res
	}

	for _, sp := range scorePaths {
		out, err := Score(prefix, kitType, sp, pvarDir)
		res[trimID(sp)] = BatchResult{out, err}
	}
	return res
}

// prepareScoreFile rewrites a raw PGS file to use RSIDs rather than chr:pos identifiers.
// It scans a .pvar file to build a map and outputs a .rsid.score file with only the score-relevant RSIDs found in the user kit
// Returns the path to the RSID-mapped score file, or the original if mapping is not needed.
func prepareScoreFile(scorePath, kitPrefix, pvarDir string) (string, error) {
	fmt.Printf("[prepareScoreFile] Mapping RSIDs for %s\n", scorePath)

	kitDir := filepath.Dir(kitPrefix)
	scoresDir := filepath.Join(kitDir, config.ScoreOutputDirName)
	if err := os.MkdirAll(scoresDir, 0755); err != nil {
		return "", err
	}

	// locate a .pvar
	searchDir := kitDir
	if pvarDir != "" {
		searchDir = pvarDir
	}
	var pvarPath string
	filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".pvar") && pvarPath == "" {
			pvarPath = path
		}
		return nil
	})
	if pvarPath == "" {
		return "", fmt.Errorf("pvar not found in %s", searchDir)
	}

	// build chr:pos → rsID map
	rsMap := make(map[string]string)
	pf, _ := os.Open(pvarPath)
	defer pf.Close()
	sc := bufio.NewScanner(pf)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "#") {
			continue
		}
		cols := strings.Split(sc.Text(), "\t")
		if len(cols) >= 3 {
			rsMap[fmt.Sprintf("%s:%s", cols[0], cols[1])] = cols[2]
		}
	}

	// if score already uses rsIDs, return as-is
	inF, _ := os.Open(scorePath)
	defer inF.Close()
	s := bufio.NewScanner(inF)
	if !s.Scan() {
		return "", fmt.Errorf("empty score file")
	}
	hdr := strings.Split(s.Text(), "\t")
	if len(hdr) < 4 || hdr[0] != "chr_name" {
		return scorePath, nil
	}

	out := filepath.Join(scoresDir,
		strings.TrimSuffix(filepath.Base(scorePath), filepath.Ext(scorePath))+".rsid.score")
	outF, _ := os.Create(out)
	defer outF.Close()

	for s.Scan() {
		cols := strings.Split(s.Text(), "\t")
		if len(cols) < 4 {
			continue
		}
		key := fmt.Sprintf("%s:%s", strings.TrimPrefix(cols[0], "chr"), cols[1])
		if rs, ok := rsMap[key]; ok && rs != "." {
			fmt.Fprintf(outF, "%s\t%s\t%s\n", rs, cols[2], cols[3])
		}
	}
	return out, nil
}

// Score runs PLINK2 to compute PGS scores given a genotype prefix and score file.
// 1. Prepares an RSID-based score file if needed
// 2. Constructs arguments (pfile, allele frequencies, score, header, extract)
// 3. Executes PLINK2 and returns the path to the .sscore output
func Score(pfilePrefix, kitType, scorePath, pvarDir string) (string, error) {
	// prepare file with RSIDs
	scPath, err := prepareScoreFile(scorePath, pfilePrefix, pvarDir)
	if err != nil {
		return "", err
	}

	// output prefix under scores dir
	outPrefix := filepath.Join(filepath.Dir(pfilePrefix), config.ScoreOutputDirName,
		trimID(scorePath))

	// build plink2 args
	args := []string{
		"--pfile", pfilePrefix,
		"--read-freq", refFreqFor(kitType),
		"--score", scPath, "cols=+scoresums", "header", "list-variants",
		"--out", outPrefix,
	}

	// if a matching snplist exists, extract
	base := filepath.Base(scPath)
	pgsID := strings.SplitN(base, ".", 2)[0]
	snplist := filepath.Join(filepath.Dir(scPath), pgsID+".snplist")
	if _, err := os.Stat(snplist); err == nil {
		args = append(args, "--extract", snplist)
	}

	// run plink2
	cmd := exec.Command(config.Plink2Cmd, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %w", config.Plink2Cmd, err)
	}
	return outPrefix + ".sscore", nil
}

// trimID removes the file extension from a path and returns the base name.
func trimID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// mustReadDir reads the directory and panics on error, returning all entries.
// Used when failure is unrecoverable (kit files must exist).
func mustReadDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Errorf("read dir %s: %w", dir, err))
	}
	return entries
}
