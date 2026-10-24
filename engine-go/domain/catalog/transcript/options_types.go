package transcript

type CreationOptionLists struct {
	Classes    []string          `json:"classes"`
	Gods       []string          `json:"gods"`
	Sizes      []string          `json:"sizes"`
	Expertises []ExpertiseOption `json:"expertises"`
}

type ExpertiseOption struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
}
