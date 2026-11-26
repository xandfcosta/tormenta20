package live

// O PERIGO DISPARADO (ver GLOSSARY, **perigo**): a armadilha que o mestre dispara, ou o ácido e a lava a que ele
// expõe alguém, da página Perigos. É uma rolagem informada como a magia — o dano, a resistência e a CD; quem passou
// e o que se aplica é do mestre —, com uma diferença de QUEM VÊ: a faixa é só dele, porque a CD e o dano são da
// ficha do perigo, que o jogador não conhece. Por isso o `redactForViewer` o tira do estado do jogador, e não a
// tela: não mandar é segurança, não desenhar seria só UX.

// HazardRoll é o perigo disparado: o nome e o tipo, o dano rolado e a resistência, e a marca que ele deixa em quem
// fica nele.
type HazardRoll struct {
	// Hazard é o nome do perigo, e Kind o tipo ("armadilha", "ambiente"), na grafia do catálogo.
	Hazard   string `json:"hazard"`
	Kind     string `json:"kind"`
	BookPage int    `json:"bookPage"`
	// Exposure é o nome da exposição do ambiente ("exposição", "imersão total"), vazio na armadilha.
	Exposure string `json:"exposure,omitempty"`
	// Parts é o dano rolado, uma parcela por dado da ficha; vazio na armadilha sem dano e na exposição, que só
	// rola na vez de quem está nela.
	Parts []SpellCastPart `json:"parts,omitempty"`
	// Save é a resistência escrita ("Reflexos evita") e DC a CD dela, vazios no perigo sem resistência.
	Save string `json:"save,omitempty"`
	DC   int    `json:"dc,omitempty"`
	// Rounds é quantas rodadas o perigo dura, rolado no disparo com o dado da ficha (RoundsDice, RoundsRolls):
	// os 2d4 do Gás Venenoso. Zero no que não dura.
	Rounds      int    `json:"rounds,omitempty"`
	RoundsDice  string `json:"roundsDice,omitempty"`
	RoundsRolls []int  `json:"roundsRolls,omitempty"`
	// Lingers diz que o dano persiste uma rodada depois de sair (a imersão total, p317 e p319).
	Lingers bool `json:"lingers,omitempty"`
	// Description é a paráfrase do catálogo, para o que a ficha não diz em campo (a queda do fosso).
	Description string `json:"description,omitempty"`
	// Lingering é a marca que o perigo deixa em quem fica nele, nil no que acaba no disparo: vira a `Markable` da
	// mesa, como a da magia.
	Lingering *QueueMark `json:"lingering,omitempty"`
}

// recordIn guarda o perigo no lugar do anterior, e a marca dele, se deixa uma, no lugar da última que o mestre
// podia pôr. O perigo que não deixa marca não toca a `Markable`: a magia de antes continua marcável.
func (h HazardRoll) recordIn(st *SessionRuntimeState) error {
	st.LastHazard = &h
	if h.Lingering != nil {
		mark := *h.Lingering
		st.Markable = &mark
	}
	return nil
}
