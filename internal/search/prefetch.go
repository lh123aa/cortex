package search

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/patrickmn/go-cache"
)

const (
	prefetchCacheTTL    = 5 * time.Minute
	prefetchCleanupIntv = 10 * time.Minute
	prefetchMaxWorkers  = 4
	prefetchMaxResults  = 5
	prefetchSearchTO    = 2 * time.Second
	prefetchMaxKeywords = 5
)

type PrefetchEngine struct {
	search  *HybridSearchEngine
	cache   *cache.Cache
	workers chan struct{}
}

func NewPrefetchEngine(se *HybridSearchEngine) *PrefetchEngine {
	return &PrefetchEngine{
		search:  se,
		cache:   cache.New(prefetchCacheTTL, prefetchCleanupIntv),
		workers: make(chan struct{}, prefetchMaxWorkers),
	}
}

func (pe *PrefetchEngine) OnFileChange(path string, content []byte) {
	if len(content) == 0 {
		return
	}

	keywords := extractKeywords(string(content), prefetchMaxKeywords)
	fileName := filepath.Base(path)
	hasFileName := false
	for _, kw := range keywords {
		if strings.EqualFold(kw, fileName) || strings.Contains(kw, strings.TrimSuffix(fileName, filepath.Ext(fileName))) {
			hasFileName = true
			break
		}
	}
	if !hasFileName && fileName != "" {
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		name = strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(name)
		keywords = append([]string{name}, keywords...)
		if len(keywords) > prefetchMaxKeywords {
			keywords = keywords[:prefetchMaxKeywords]
		}
	}

	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		select {
		case pe.workers <- struct{}{}:
			go pe.prefetchOne(path, kw)
		default:
			return
		}
	}
}

func (pe *PrefetchEngine) prefetchOne(path, keyword string) {
	defer func() { <-pe.workers }()

	ctx, cancel := context.WithTimeout(context.Background(), prefetchSearchTO)
	defer cancel()

	opts := models.SearchOptions{
		TopK:   prefetchMaxResults,
		Mode:   "hybrid",
		UserID: "",
	}
	results, err := pe.search.Search(ctx, keyword, opts)
	if err != nil || len(results) == 0 {
		return
	}
	key := prefetchCacheKey(path, keyword)
	pe.cache.Set(key, results, cache.DefaultExpiration)
}

func (pe *PrefetchEngine) Suggest(ctx context.Context, partial string, currentFile string, topK int) []*models.SearchResult {
	if partial == "" {
		return nil
	}

	key := prefetchCacheKey(currentFile, partial)
	if val, found := pe.cache.Get(key); found {
		return val.([]*models.SearchResult)
	}

	searchOpts := models.SearchOptions{
		TopK:   topK,
		Mode:   "hybrid",
		UserID: "",
	}
	results, err := pe.search.Search(ctx, partial, searchOpts)
	if err != nil {
		return nil
	}
	return results
}

func (pe *PrefetchEngine) InvalidateFile(path string) {
	prefix := path + "||"
	for key := range pe.cache.Items() {
		if strings.HasPrefix(key, prefix) {
			pe.cache.Delete(key)
		}
	}
}

func (pe *PrefetchEngine) GetCachedCount() int {
	return pe.cache.ItemCount()
}

func prefetchCacheKey(path, keyword string) string {
	return path + "||" + strings.TrimSpace(keyword)
}

var commonStopWords = map[string]bool{
	"the": true, "is": true, "at": true, "of": true, "on": true, "and": true,
	"a": true, "an": true, "in": true, "to": true, "for": true, "it": true,
	"or": true, "be": true, "by": true, "as": true, "that": true, "this": true,
	"with": true, "from": true, "are": true, "was": true, "were": true,
	"has": true, "have": true, "had": true, "not": true, "but": true,
	"what": true, "which": true, "who": true, "how": true, "where": true,
	"when": true, "do": true, "does": true, "did": true, "will": true,
}

func isStopWord(word string) bool {
	lower := strings.ToLower(word)
	if commonStopWords[lower] {
		return true
	}
	if len(word) <= 2 {
		return true
	}
	for _, r := range word {
		if unicode.IsDigit(r) {
			return false
		}
	}
	return false
}

func extractKeywords(content string, max int) []string {
	lines := strings.Split(content, "\n")

	seen := make(map[string]bool)
	var keywords []string

	titleCandidates := extractTitleCandidates(content, lines)
	for _, t := range titleCandidates {
		if len(keywords) >= max {
			break
		}
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		keywords = append(keywords, t)
	}
	if len(keywords) >= max {
		return keywords
	}

	wordFreq := make(map[string]int)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.FieldsFunc(line, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
		})
		for _, f := range fields {
			if isStopWord(f) {
				continue
			}
			wordFreq[f]++
		}
	}

	type wordEntry struct {
		word string
		freq int
	}
	var sorted []wordEntry
	for w, f := range wordFreq {
		sorted = append(sorted, wordEntry{w, f})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].freq > sorted[j].freq
	})

	for _, entry := range sorted {
		if len(keywords) >= max {
			break
		}
		if seen[entry.word] {
			continue
		}
		seen[entry.word] = true
		keywords = append(keywords, entry.word)
	}

	return keywords
}

func extractTitleCandidates(content string, lines []string) []string {
	var candidates []string

	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if strings.HasPrefix(firstLine, "# ") {
			candidates = append(candidates, strings.TrimPrefix(firstLine, "# "))
		} else if strings.HasPrefix(firstLine, "#") && !strings.HasPrefix(firstLine, "##") {
			candidates = append(candidates, strings.TrimLeft(firstLine, "# "))
		}
	}

	var firstH1, firstH2 string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && firstH1 == "" {
			firstH1 = strings.TrimPrefix(trimmed, "# ")
		}
		if strings.HasPrefix(trimmed, "## ") && firstH2 == "" {
			firstH2 = strings.TrimPrefix(trimmed, "## ")
		}
	}
	if firstH1 != "" {
		candidates = append(candidates, firstH1)
	}
	if firstH2 != "" && firstH2 != firstH1 {
		candidates = append(candidates, firstH2)
	}

	return candidates
}
