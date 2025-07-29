// backend/server/handlers/results_handler.go
package handlers

import (
    "bufio"
    "encoding/json"
    "errors"
    "io"
    "math"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync"

    "github.com/adamwestgate/easy-pgs/backend/data"
    "github.com/adamwestgate/easy-pgs/backend/config"
    "github.com/adamwestgate/easy-pgs/backend/preprocessing/scoring"
)

type ScoringResults struct {
    User map[string]scoring.BatchResult `json:"user"`
    Pop  map[string]scoring.BatchResult `json:"pop"`
}

type flatResults struct {
    Population    map[string]float64 `json:"population"`
    User          map[string]float64 `json:"user"`
    Z             map[string]float64 `json:"z"`
    Pct           map[string]float64 `json:"pct"`
    Trait         map[string]string  `json:"trait"`
    PctSnpsScored map[string]float64 `json:"pct_snps_scored"`
}

var (
    resultsMu   sync.RWMutex
    resultsByID = make(map[string]flatResults)
)

type stats struct{ mean, sd float64 }

// parseSscoreStats reads a .sscore file at the given path and computes the mean and SD of per-variant averages.
// It looks for columns SCORE1_AVG (preferred) or SCORE1_SUM/ALLELE_CT.
func parseSscoreStats(path string) (stats, error) {
    f, err := os.Open(path)
    if err != nil {
        return stats{}, err
    }
    defer f.Close()

    sc := bufio.NewScanner(f)
    if !sc.Scan() {
        return stats{}, errors.New("empty .sscore")
    }
    header := strings.Fields(sc.Text())

    idxAvg, idxSum, idxCt := -1, -1, -1
    for i, h := range header {
        switch h {
        case "SCORE1_AVG":
            idxAvg = i
        case "SCORE1_SUM":
            idxSum = i
        case "ALLELE_CT":
            idxCt = i
        }
    }
    if idxAvg == -1 && (idxSum == -1 || idxCt == -1) {
        return stats{}, errors.New("missing columns")
    }

    var n int
    var sum, sumSq float64
    for sc.Scan() {
        fields := strings.Fields(sc.Text())
        var avg float64
        if idxAvg != -1 {
            v, err := strconv.ParseFloat(fields[idxAvg], 64)
            if err != nil {
                continue
            }
            avg = v
        } else {
            s, err1 := strconv.ParseFloat(fields[idxSum], 64)
            c, err2 := strconv.ParseFloat(fields[idxCt], 64)
            if err1 != nil || err2 != nil || c == 0 {
                continue
            }
            avg = s / c
        }
        n++
        sum += avg
        sumSq += avg * avg
    }
    if err := sc.Err(); err != nil {
        return stats{}, err
    }
    if n == 0 {
        return stats{}, errors.New("no rows")
    }

    mean := sum / float64(n)
    var sd float64
    if n > 1 {
        variance := (sumSq - sum*sum/float64(n)) / float64(n-1)
        if variance > 0 {
            sd = math.Sqrt(variance)
        }
    }
    return stats{mean, sd}, nil
}

// ScoreKitWithPGS performs batch scoring of a kit against given PGS weight files.
// 1. Lookup kit from kitStore
// 2. Score user data & get list of snps scored
// 3. Score population on list of snps scored in the user kit
// Returns ScoringResults containing user and population BatchResult maps.
func ScoreKitWithPGS(kitID string, norm []string) (*ScoringResults, error) {
    prefix, kitType, ok := kitStore.Lookup(kitID)
    if !ok {
        return nil, errors.New("kit not found")
    }

    // 1️⃣ User kit scoring
    userRaw := scoring.BatchScore(prefix, kitType, norm, "")
    user := make(map[string]scoring.BatchResult, len(userRaw))
    for k, v := range userRaw {
        user[canonicalID(k)] = v
    }

    // 2️⃣ Choose reference panel
    popRoot := config.ReferenceAncestryDir
    if strings.ToLower(kitType) == "23andme" {
        popRoot = config.Reference23andmeDir
    }

    // 3️⃣ Move .vars → .snplist and gather .rsid.score files
    varsDst := filepath.Join(popRoot, "scores")
    _ = os.MkdirAll(varsDst, 0o755)
    var popWeights []string

    for id, br := range user {
        if br.Err != nil || br.ScorePath == "" {
            continue
        }
        // .vars → .snplist
        srcVars := br.ScorePath + ".vars"
        if _, err := os.Stat(srcVars); err == nil {
            dst := filepath.Join(varsDst, id+".snplist")
            if err := os.Rename(srcVars, dst); err != nil {
                if in, e1 := os.Open(srcVars); e1 == nil {
                    defer in.Close()
                    if out, e2 := os.Create(dst); e2 == nil {
                        defer out.Close()
                        _, _ = io.Copy(out, in)
                        _ = os.Remove(srcVars)
                    }
                }
            }
        }
        // copy .norm.rsid.score
        dir := filepath.Dir(br.ScorePath)
        srcScore := filepath.Join(dir, id+".norm.rsid.score")
        if _, err := os.Stat(srcScore); err == nil {
            dst := filepath.Join(varsDst, filepath.Base(srcScore))
            if _, err2 := os.Stat(dst); err2 != nil {
                if in, e1 := os.Open(srcScore); e1 == nil {
                    defer in.Close()
                    if out, e2 := os.Create(dst); e2 == nil {
                        defer out.Close()
                        _, _ = io.Copy(out, in)
                    }
                }
            }
            popWeights = append(popWeights, dst)
        }
    }

    // 4️⃣ Population scoring on that snplist
    popRaw := scoring.BatchScore(popRoot, kitType, popWeights, "")
    pop := make(map[string]scoring.BatchResult, len(popRaw))
    for k, v := range popRaw {
        pop[canonicalID(k)] = v
    }

    return &ScoringResults{User: user, Pop: pop}, nil
}

// storeResults flattens ScoringResults and caches them in memory under the given kitID.
func storeResults(kitID string, r *ScoringResults) {
    // Determine chip panel
    _, kitType, ok := kitStore.Lookup(kitID)
    popRoot := config.ReferenceAncestryDir
    if ok && strings.ToLower(kitType) == "23andme" {
        popRoot = config.Reference23andmeDir
    }

    flat := flatResults{
        Population:    map[string]float64{},
        User:          map[string]float64{},
        Z:             map[string]float64{},
        Pct:           map[string]float64{},
        Trait:         map[string]string{},
        PctSnpsScored: map[string]float64{}, // "Coverage" on the results page
    }

    // a) population means & SDs
    popStats := map[string]stats{}
    for id, br := range r.Pop {
        if br.Err == nil && br.ScorePath != "" {
            if st, err := parseSscoreStats(br.ScorePath); err == nil {
                popStats[id] = st
                flat.Population[id] = st.mean
            }
        }
        if _, exist := flat.Trait[id]; !exist {
            flat.Trait[id] = getTraitLabel(id)
        }
    }

    // b) user + z / pct
    for id, br := range r.User {
        if br.Err == nil && br.ScorePath != "" {
            if stU, err := parseSscoreStats(br.ScorePath); err == nil {
                flat.User[id] = stU.mean
                if stP, ex := popStats[id]; ex && stP.sd > 0 {
                    z := (stU.mean - stP.mean) / stP.sd
                    flat.Z[id] = z
                    flat.Pct[id] = 0.5 * (1 + math.Erf(z/math.Sqrt2))
                }
            }
        }
        if _, exist := flat.Trait[id]; !exist {
            flat.Trait[id] = getTraitLabel(id)
        }
    }

    // c) cleanup NaN/Inf
    for _, m := range []map[string]float64{flat.Population, flat.User, flat.Z, flat.Pct} {
        for k, v := range m {
            if math.IsNaN(v) || math.IsInf(v, 0) {
                delete(m, k)
            }
        }
    }

    // d) percent SNPs scored
    for id := range flat.User {
        snpList := filepath.Join(popRoot, "scores", id+".snplist")
        tsv := filepath.Join(config.PGSFilesDir, id, id+".norm.tsv")
        flat.PctSnpsScored[id] = snpRetentionPercent(snpList, tsv)
    }

    resultsMu.Lock()
    resultsByID[kitID] = flat
    resultsMu.Unlock()
}

// fetchResults retrieves cached results
func fetchResults(kitID string) (flatResults, bool) {
    resultsMu.RLock()
    defer resultsMu.RUnlock()
    r, ok := resultsByID[kitID]
    return r, ok
}

// ResultsHandler handles HTTP GET requests to /results?kitId=<id>.
// It returns cached scoring results in JSON, or an error if not found or wrong method.
func ResultsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodOptions {
        w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    kitID := r.URL.Query().Get("kitId")
    if kitID == "" {
        http.Error(w, "kitId query param required", http.StatusBadRequest)
        return
    }
    if res, ok := fetchResults(kitID); ok {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(res)
        return
    }
    http.Error(w, "results not ready", http.StatusNotFound)
}

// canonicalID strips any suffix after the first "." in a PGS ID string.
func canonicalID(raw string) string {
    if dot := strings.IndexByte(raw, '.'); dot >= 0 {
        return raw[:dot]
    }
    return raw
}

// getTraitLabel looks up a PGS ID in loaded data to find its reported trait label.
func getTraitLabel(pgsID string) string {
    idClean := canonicalID(pgsID)
    for _, m := range data.LoadedScores {
        if id, _ := m["Polygenic Score (PGS) ID"].(string); id == idClean {
            if trait, ok := m["Reported Trait"].(string); ok {
                return trait
            }
            if trait, ok := m["Reported trait"].(string); ok {
                return trait
            }
        }
    }
    for _, tr := range data.LoadedTraits {
        for _, pid := range tr.PGSFiles {
            if pid == idClean {
                return tr.Label
            }
        }
    }
    return ""
}

// snpRetentionPercent calculates the percentage of SNPs in snpList versus lines in TSV (minus header). This is to get the Coverage stat for the results page.
func snpRetentionPercent(snplistPath, tsvPath string) float64 {
    snpF, err := os.Open(snplistPath)
    if err != nil {
        return 0
    }
    defer snpF.Close()
    tsvF, err := os.Open(tsvPath)
    if err != nil {
        return 0
    }
    defer tsvF.Close()

    count := func(sc *bufio.Scanner) int {
        n := 0
        for sc.Scan() {
            n++
        }
        return n
    }

    snpCount := count(bufio.NewScanner(snpF))
    tsvCount := count(bufio.NewScanner(tsvF))
    if tsvCount <= 1 {
        return 0
    }
    return float64(snpCount) / float64(tsvCount-1) * 100.0
}
