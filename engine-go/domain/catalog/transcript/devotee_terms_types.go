package transcript

type DevoteeTermTable struct {
	OpenTerms   []string             `json:"openTerms"`
	TermToNames OrderedMap[[]string] `json:"termToNames"`
}
