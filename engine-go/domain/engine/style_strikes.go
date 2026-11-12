package engine

// OS ESTILOS QUE DÃO UM ATAQUE A MAIS NA AÇÃO AGREDIR (p125). A flag ligada é a declaração, e quem a liga
// é o Usar do poder (`book.DeclaredFlag`): o -2 vale nos dois ataques, e por isso o primeiro golpe só abre o
// segundo com ela já ligada.
//
// A AMBIDESTRIA TIRA O -2 E NÃO O SEGUNDO ATAQUE (p125). O catálogo escreve o -2 com `Unless: ambidestria` e,
// ao lado, uma linha de valor zero com `Requires: ambidestria`: sem ela a declaração sumia junto com o -2,
// e o Usar não teria o que ligar. A flag `tem-estilo-de-duas-armas` cala a declaração da Ambidestria de
// classe: com os dois poderes há um Usar só.

// SharpAimPowerID é a Mira Apurada: "quando usa a ação mirar" (p128). O interruptor dela é ligado pelo gesto,
// e não é oferecido: oferecido, dava o +2 a quem não mirou.
const SharpAimPowerID = "mira-apurada"

// As fontes do ataque adicional, que são os poderes e as flags que o Usar deles liga.
const (
	TwoWeaponStylePowerID = "estilo-de-duas-armas"
	// AmbidextrousSwitch é a flag que o Usar da Ambidestria de classe liga, e dá os mesmos dois ataques (p50,
	// p65). A flag `ambidestria` já é o "possui Ambidestria" da p125, e não pode ser também a declaração.
	AmbidextrousSwitch = "ataque-ambidestro"
	RapidShotPowerID   = "disparo-rapido"
	DoubleTrait        = "dupla"
	// WildAgileBothFlag é a declaração da Forma Ágil: "pode atacar com ambas" as armas naturais (p63).
	WildAgileBothFlag = "ataque-forma-agil"
)

// StyleStrike é o que um estilo ligado dá a esta carta.
type StyleStrike struct {
	SourceID string
	// Name é o nome do poder, que a linha do ataque escreve.
	Name string
	// OtherWeapon é o "um com cada arma" das Duas Armas. Sem ele, o adicional é "com ela", a
	// mesma arma do Disparo Rápido.
	OtherWeapon bool
	// FullAction é o "gastar uma ação completa para agredir" do Disparo Rápido.
	FullAction bool
	// NaturalPair é o "atacar com ambas" da Forma Ágil (p63): as duas armas naturais são iguais, e o
	// adicional sai da mesma carta, com a outra.
	NaturalPair bool
}

// offerStyleStrikes marca as cartas das armas empunhadas com o estilo declarado.
//
// O DESARMADO E A ARMA NATURAL NÃO ENTRAM, porque o livro pede duas armas EMPUNHADAS. A falta da
// arma leve não tira o ataque, nem a arma de disparo que não é arco: o Usar avisa, e quem
// decide é o mestre.
func (r *Ruleset) offerStyleStrikes(ch Character, active map[string]bool, effects ItemEffects, wielded []WeaponCard) {
	owned := parseChoiceSet(ch.ClassPowers).has
	switchedOn := func(power string) bool { return owned[power] && active[FlagGroupID(power)] }
	two := &StyleStrike{SourceID: TwoWeaponStylePowerID, Name: "Estilo de Duas Armas", OtherWeapon: true}
	if !switchedOn(TwoWeaponStylePowerID) {
		two = nil
		if effects.Flags["ambidestria"] && active[FlagGroupID(AmbidextrousSwitch)] {
			two = &StyleStrike{SourceID: AmbidextrousSwitch, Name: "Ambidestria", OtherWeapon: true}
		}
	}
	if two != nil && len(wielded) == 1 && r.isDoubleWeapon(wielded[0]) {
		// Cada ponta conta como uma arma (p143), e o adicional é com a mesma carta.
		wielded[0].Style = &StyleStrike{SourceID: two.SourceID, Name: two.Name}
		return
	}
	if two != nil && len(wielded) == 2 {
		for i := range wielded {
			wielded[i].Style = two
			if wielded[i].OtherUse != nil {
				wielded[i].OtherUse.Style = two
			}
		}
		return
	}
	if !switchedOn(RapidShotPowerID) {
		return
	}
	for i := range wielded {
		if catalog := r.getCatalogItem(wielded[i].CatalogID); catalog != nil && catalog.Weapon != nil && catalog.Weapon.Purpose == "ranged" {
			wielded[i].Style = &StyleStrike{SourceID: RapidShotPowerID, Name: "Disparo Rápido", FullAction: true}
		}
	}
}

// isDoubleWeapon é a arma que "pode ser usada com Estilo de Duas Armas (e poderes similares) [...] como se
// fosse uma arma de uma mão e uma arma leve" (p143).
func (r *Ruleset) isDoubleWeapon(card WeaponCard) bool {
	catalog := r.getCatalogItem(card.CatalogID)
	return catalog != nil && catalog.Weapon != nil && hasTrait(catalog.Weapon.Traits, DoubleTrait)
}

// agileShapeStrike é o adicional da Forma Ágil declarada, na carta da arma da forma (p63). As duas armas
// naturais dela são iguais, então elas são UMA carta, e o adicional sai dela de novo, como a outra
// ponta da arma dupla.
func agileShapeStrike(ch Character, active map[string]bool) *StyleStrike {
	if ch.WildShape == nil || ch.WildShape.Form != WildAgile || !active[FlagGroupID(WildAgileBothFlag)] {
		return nil
	}
	return &StyleStrike{SourceID: WildAgileBothFlag, Name: "Forma Ágil", NaturalPair: true}
}
