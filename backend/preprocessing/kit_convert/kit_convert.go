package kitconv

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/adamwestgate/easy-pgs/backend/config"
)

/*──────────────────────── helpers ───────────────────────*/

func isValidGenotype(g string) bool {
    if len(g) != 2 {
        return false
    }
    for _, c := range g {
        switch c {
        case 'A', 'C', 'G', 'T', '-':
        default:
            return false
        }
    }
    return true
}

func normalizeChrom(raw string) string {
    r := strings.ToUpper(strings.TrimPrefix(raw, "CHR"))
    switch r {
    case "23":
        return "X"
    case "24", "25":
        return "Y"
    case "26":
        return "MT"
    default:
        return r
    }
}

/*─────────────────────── converters ─────────────────────*/

func ConvertAncestry(src io.Reader, dst io.Writer) error {
    sc := bufio.NewScanner(src)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        f := strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == '\t' })
        if len(f) < 5 || strings.EqualFold(f[0], "rsid") {
            continue
        }
        rsid, chrom, pos, a1, a2 := f[0], normalizeChrom(f[1]), f[2], f[3], f[4]
        gt := strings.ReplaceAll(a1+a2, "0", "-")
        if !isValidGenotype(gt) {
            continue
        }
        fmt.Fprintf(dst, "%s\t%s\t%s\t%s\n", rsid, chrom, pos, gt)
    }
    return sc.Err()
}

func Convert23andMe(src io.Reader, dst io.Writer) error {
    sc := bufio.NewScanner(src)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(strings.ToLower(line), "rsid") {
            continue
        }
        f := strings.Fields(line)
        if len(f) < 4 {
            continue
        }
        rsid, chrom, pos, gt := f[0], normalizeChrom(f[1]), f[2], f[3]
        if !isValidGenotype(gt) {
            continue
        }
        fmt.Fprintf(dst, "%s\t%s\t%s\t%s\n", rsid, chrom, pos, gt)
    }
    return sc.Err()
}

/*──────────────── ConvertFileToPgen (returns kitType) ───────────────*/

// ConvertFileToPgen converts a raw consumer‑DNA file into PLINK2 pgen.
// It returns the detected kitType ("ancestry" or "23andme").
func ConvertFileToPgen(rawPath, processedDir string) (string, error) {
    if err := os.MkdirAll(processedDir, 0755); err != nil {
        return "", fmt.Errorf("mkdir processedDir: %w", err)
    }

    base := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))
    tmp4 := filepath.Join(processedDir, base+"_4col.txt")

    in, err := os.Open(rawPath)
    if err != nil {
        return "", err
    }
    defer in.Close()
    out, err := os.Create(tmp4)
    if err != nil {
        return "", err
    }
    defer out.Close()

    // sniff first non‑comment line to decide chip
    sc := bufio.NewScanner(in)
    var first string
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line != "" && !strings.HasPrefix(line, "#") {
            first = line
            break
        }
    }
    if first == "" {
        return "", fmt.Errorf("empty kit file")
    }
    cols := strings.FieldsFunc(first, func(r rune) bool { return r == ',' || r == '\t' })

    // rewind file
    if _, err := in.Seek(0, io.SeekStart); err != nil {
        return "", err
    }

    var kitType string
    switch len(cols) {
    case 5:
        kitType = "ancestry"
        if err := ConvertAncestry(in, out); err != nil {
            return "", err
        }
    case 4:
        kitType = "23andme"
        if err := Convert23andMe(in, out); err != nil {
            return "", err
        }
    default:
        return "", fmt.Errorf("unrecognized column count: %d", len(cols))
    }

    maniDir := map[string]string{"ancestry": config.ChipManifestAncestryDir, "23andme": config.ChipManifestV5Dir}[kitType]
    snplist := filepath.Join(maniDir, kitType+".snplist")
    refallele := filepath.Join(maniDir, kitType+".refallele")

    outBase := filepath.Join(processedDir, base)

    // PLINK1: 4‑col → .bed/.bim/.fam
    cmd1 := exec.Command("plink1", "--23file", tmp4, "--allele-acgt", "--extract", snplist,
        "--make-bed", "--out", outBase)
    cmd1.Stdout, cmd1.Stderr = os.Stdout, os.Stderr
    if err := cmd1.Run(); err != nil {
        return "", fmt.Errorf("plink1: %w", err)
    }

    if err := patchBimWithRefAlleles(outBase+".bim", refallele); err != nil {
        return "", err
    }

    patched := outBase + "_patched"
    _ = os.Rename(outBase+".bim.patched", patched+".bim")
    _ = os.Rename(outBase+".bed", patched+".bed")
    _ = os.Rename(outBase+".fam", patched+".fam")

    // PLINK2: patched bed → pgen
    cmd2 := exec.Command("plink2", "--bfile", patched, "--ref-allele", refallele, "--make-pgen", "--out", outBase)
    cmd2.Stdout, cmd2.Stderr = os.Stdout, os.Stderr
    if err := cmd2.Run(); err != nil {
        return "", fmt.Errorf("plink2: %w", err)
    }

    _ = os.Remove(tmp4)
    return kitType, nil
}


func patchBimWithRefAlleles(bimPath, refPath string) error {
	refMap := make(map[string]string)

	refFile, err := os.Open(refPath)
	if err != nil {
		return err
	}
	defer refFile.Close()

	sc := bufio.NewScanner(refFile)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) >= 2 {
			refMap[parts[0]] = strings.ToUpper(parts[1])
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	bimTmp := bimPath + ".patched"
	excludePath := strings.TrimSuffix(bimPath, ".bim") + "_exclude.txt"

	in, err := os.Open(bimPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(bimTmp)
	if err != nil {
		return err
	}
	defer out.Close()

	exclude, err := os.Create(excludePath)
	if err != nil {
		return err
	}
	defer exclude.Close()

	sc = bufio.NewScanner(in)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 6 {
			continue
		}
		rsid := fields[1]
		ref := refMap[rsid]

		if fields[4] == "0" {
			fields[4] = ref
		}
		if fields[5] == "0" {
			fields[5] = ref
		}

		// if the REF allele matches A2 but not A1, swap
		if fields[5] == ref && fields[4] != ref {
			fields[4], fields[5] = fields[5], fields[4]
		}

		// If either allele is still 0 or they are identical, add to exclude
		if fields[4] == "0" || fields[5] == "0" || fields[4] == fields[5] {
			_, _ = fmt.Fprintln(exclude, rsid)
		}

        if fields[4] == fields[5] {
        fields[5] = "0" // force one to be missing to avoid PLINK2 crash
        fmt.Fprintln(exclude, rsid)
    }

		_, _ = fmt.Fprintln(out, strings.Join(fields, "\t"))
	}
	if err := sc.Err(); err != nil {
		return err
	}

	_ = os.Rename(bimTmp, bimPath)
	return nil
}
