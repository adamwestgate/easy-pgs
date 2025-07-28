// Package pgsconvert normalises any PGS‑Catalog scoring file stream to a
// canonical four‑column layout expected by the PLINK scoring step:
//
//     chr_name   chr_position   effect_allele   effect_weight
//
// Additionally, if a score file has exactly three columns in the form:
//
//     rsID	effect_allele	effect_weight
//
// then it will be passed through (header stripped of metadata) and output
// unchanged with only those three columns.
//
// Robust to:
// • Optional metadata lines beginning with "#".
// • Header rows appearing anywhere.
// • Rows missing *rsID* and/or trailing optional columns.
//
// The caller handles decompression; `r` must already be plain text.

package pgsconvert

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "strconv"
    "strings"
)

// Options lets a caller override defaults.
// WeightCol – override the column name that contains the weights.  If empty,
//             the first recognised default wins.
// ----------------------------------------------------------------------------

type Options struct {
    WeightCol string
}

var defaultWeightNames = []string{"effect_weight", "beta", "weight", "or"}

// Normalize converts a PGS score file on `r` to canonical TSV on `w`.
// It emits **one** header (4-col layout) unless the file is already
// a simple 3-col rsID score, in which case it passes through only those columns.
func Normalize(r io.Reader, w io.Writer, opt Options) error {
    out := bufio.NewWriter(w)
    defer out.Flush()

    scan := bufio.NewScanner(r)
    scan.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

    // 1) Find first non-comment line (header or data).
    var (
        header       []string
        isHeaderLine bool
    )
    for scan.Scan() {
        line := strings.TrimSpace(scan.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        header = strings.Split(line, "\t")
        isHeaderLine = looksLikeHeader(header)
        break
    }
    if len(header) == 0 {
        return errors.New("empty score file – no header or data row found")
    }

    // 2) Special-case: exactly 3 columns rsID,effect_allele,effect_weight
    if isThreeColHeader(header) {
        // strip other header metadata, output only the three columns
        fmt.Fprintln(out, strings.Join(header, "\t"))
        // stream rest of the file
        for scan.Scan() {
            line := strings.TrimSpace(scan.Text())
            if line == "" || strings.HasPrefix(line, "#") {
                continue
            }
            fields := strings.Split(line, "\t")
            if len(fields) < 3 {
                return fmt.Errorf("malformed row – expected >=3 cols: %q", line)
            }
            fmt.Fprintf(out, "%s\t%s\t%s\n", fields[0], fields[1], fields[2])
        }
        return scan.Err()
    }

    // 3) Build column map for general 4+ column files.
    var colMap map[string]int
    if isHeaderLine {
        colMap = buildColumnMap(header)
    } else {
        colMap = buildSyntheticMap(len(header))
    }

    // 4) Determine weight column key.
    weightKey := strings.ToLower(opt.WeightCol)
    if weightKey == "" {
        for _, cand := range defaultWeightNames {
            if _, ok := colMap[cand]; ok {
                weightKey = cand
                break
            }
        }
        if weightKey == "" {
            return fmt.Errorf("no recognised weight column (looked for %v)", defaultWeightNames)
        }
    } else if _, ok := colMap[weightKey]; !ok {
        return fmt.Errorf("requested weight column %q not found in header", weightKey)
    }

    // 5) Locate required indices
    chrIdx, posIdx, a1Idx := colMap["chr_name"], colMap["chr_position"], colMap["effect_allele"]
    betaIdx := colMap[weightKey]
    if chrIdx < 0 || posIdx < 0 || a1Idx < 0 || betaIdx < 0 {
        return fmt.Errorf("missing required columns – header map: %v", colMap)
    }

    // 6) Emit canonical 4-column header
    fmt.Fprintln(out, "chr_name\tchr_position\teffect_allele\teffect_weight")

    // 7) Helper to write a general row
    write := func(fields []string) error {
        // pad missing rsID at front
        if len(fields)+1 == len(header) {
            if _, err := strconv.Atoi(fields[0]); err == nil || fields[0] == "X" || fields[0] == "Y" || fields[0] == "MT" {
                fields = append([]string{""}, fields...)
            }
        }
        // pad trailing missing fields
        if len(fields) < len(header) {
            pad := make([]string, len(header)-len(fields))
            fields = append(fields, pad...)
        }
        if betaIdx >= len(fields) {
            return fmt.Errorf("malformed row – weight column absent: %q", strings.Join(fields, "\t"))
        }
        _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", fields[chrIdx], fields[posIdx], fields[a1Idx], fields[betaIdx])
        return err
    }

    // 8) If first non-comment line was data, write it
    if !isHeaderLine {
        if err := write(header); err != nil {
            return err
        }
    }

    // 9) Stream remaining lines
    for scan.Scan() {
        line := strings.TrimSpace(scan.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        if err := write(strings.Split(line, "\t")); err != nil {
            return err
        }
    }
    return scan.Err()
}

// isThreeColHeader returns true if header is exactly the three columns we pass through.
func isThreeColHeader(cols []string) bool {
    if len(cols) != 3 {
        return false
    }
    return strings.EqualFold(cols[0], "rsID") &&
        strings.EqualFold(cols[1], "effect_allele") &&
        strings.EqualFold(cols[2], "effect_weight")
}

// looksLikeHeader returns true if cols contain minimum four-column names.
func looksLikeHeader(cols []string) bool {
    lower := make(map[string]struct{}, len(cols))
    for _, c := range cols {
        lower[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
    }
    must := []string{"chr_name", "chr_position", "effect_allele"}
    for _, m := range must {
        if _, ok := lower[m]; !ok {
            return false
        }
    }
    for _, w := range defaultWeightNames {
        if _, ok := lower[w]; ok {
            return true
        }
    }
    return false
}

func buildColumnMap(cols []string) map[string]int {
    m := make(map[string]int, len(cols))
    for i, c := range cols {
        m[strings.ToLower(strings.TrimSpace(c))] = i
    }
    return m
}

func buildSyntheticMap(n int) map[string]int {
    idx := func(i int) int { if i < n { return i } ; return -1 }
    return map[string]int{
        "chr_name":      idx(0),
        "chr_position":  idx(1),
        "effect_allele": idx(2),
        "effect_weight": idx(3),
    }
}
