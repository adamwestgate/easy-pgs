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

/*──────────────────────── 1.  Helper: pick the .afreq ───────────────────────*/
func refFreqFor(kitType string) string {
	switch strings.ToLower(kitType) {
	case "ancestry":
		return config.ReferenceFreqAncestry
	case "23andme":
		return config.ReferenceFreq23andme
	default:
		// fallback to union panel
		return config.ReferenceFreqFiltered
	}
}

/*──────────────────────── 2.  BatchResult struct ────────────────────────────*/
type BatchResult struct {
	ScorePath string
	Err       error
}

/*──────────────────────── 3.  BatchScore (chip-aware) ───────────────────────*/
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

/*──────────────────────── 4.  prepareScoreFile (RSID remap) ─────────────────*/
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

/*──────────────────────── 5.  Score (runs plink2) ───────────────────────────*/
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

/*──────────────────────── 6.  tiny helpers ──────────────────────────────────*/
func trimID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func mustReadDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Errorf("read dir %s: %w", dir, err))
	}
	return entries
}
