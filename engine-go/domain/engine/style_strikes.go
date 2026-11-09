package engine

// OS ESTILOS QUE DÃO UM ATAQUE A MAIS NA AÇÃO AGREDIR (p125). O interruptor ligado é a declaração:
// o -2 vale nos dois ataques, e por isso o primeiro golpe só abre o segundo com ele já ligado.

// As fontes do ataque adicional, que são os poderes e os interruptores deles.
const (
	TwoWeaponStylePowerID = "estilo-de-duas-armas"
	RapidShotPowerID      = "disparo-rapido"
)

// StyleStrike é o que um estilo ligado dá a esta carta.
type StyleStrike struct {
	SourceID string
	// OtherWeapon é o "um com cada arma" das Duas Armas. Sem ele, o adicional é "com ela", a
	// mesma arma do Disparo Rápido.
	OtherWeapon bool
	// FullAction é o "gastar uma ação completa para agredir" do Disparo Rápido.
	FullAction bool
}

// offerStyleStrikes marca as cartas das armas empunhadas com o estilo ligado.
//
// O DESARMADO E A ARMA NATURAL NÃO ENTRAM, porque o livro pede duas armas EMPUNHADAS. A falta da
// arma leve não tira o ataque, nem a arma de disparo que não é arco: o interruptor avisa, e quem
// decide é o mestre.
func (r *Ruleset) offerStyleStrikes(ch Character, active map[string]bool, wielded []WeaponCard) {
	owned := parseChoiceSet(ch.ClassPowers).has
	switchedOn := func(power string) bool { return owned[power] && active[FlagGroupID(power)] }
	if switchedOn(TwoWeaponStylePowerID) && len(wielded) == 2 {
		style := &StyleStrike{SourceID: TwoWeaponStylePowerID, OtherWeapon: true}
		for i := range wielded {
			wielded[i].Style = style
			if wielded[i].OtherUse != nil {
				wielded[i].OtherUse.Style = style
			}
		}
		return
	}
	if !switchedOn(RapidShotPowerID) {
		return
	}
	for i := range wielded {
		if catalog := r.getCatalogItem(wielded[i].CatalogID); catalog != nil && catalog.Weapon != nil && catalog.Weapon.Purpose == "ranged" {
			wielded[i].Style = &StyleStrike{SourceID: RapidShotPowerID, FullAction: true}
		}
	}
}
