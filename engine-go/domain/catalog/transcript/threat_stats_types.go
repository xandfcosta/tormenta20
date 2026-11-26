package transcript

// ThreatStatsBook é a Tabela 7-2 (p323) e os pacotes de atributos de cada patamar (p322): o que o livro dá para
// montar a ficha de um NPC pelo ND, com os valores finais e sem a origem de cada bônus.
type ThreatStatsBook struct {
	Rows  []ThreatStatsRow `json:"rows"`
	Tiers []ThreatTier     `json:"tiers"`
}

// ThreatStatsRow é uma linha da Tabela 7-2.
type ThreatStatsRow struct {
	ND float64 `json:"nd"`
	// Attack é o bônus do teste de ataque, e Attacks quantos ataques o NPC faz por rodada — o "x2" ao lado do
	// ataque; 1 onde a tabela não escreve multiplicador.
	Attack  int    `json:"attack"`
	Attacks int    `json:"attacks"`
	Damage  string `json:"damage"`
	Defense int    `json:"defense"`
	HP      int    `json:"hp"`
	// FocusedSkills é o valor das perícias em que o NPC é focado, pelo conceito dele, e OtherSkills o das outras.
	FocusedSkills int `json:"focusedSkills"`
	OtherSkills   int `json:"otherSkills"`
	// DC é a dificuldade para resistir às habilidades do NPC.
	DC int `json:"dc"`
}

// ThreatTier é um patamar (p35) com o pacote de atributos que o mestre distribui entre os seis (p322).
type ThreatTier struct {
	Name       string  `json:"name"`
	FirstND    float64 `json:"firstNd"`
	LastND     float64 `json:"lastNd"`
	Attributes []int   `json:"attributes"`
}
