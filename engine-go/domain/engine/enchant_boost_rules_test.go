package engine

import (
	"slices"
	"strings"
	"testing"
)

// O GASTO DE 2 PM DOS ENCANTOS (p335-336). Cinco encantos abrem com a mesma frase: "Uma vez por rodada,
// quando ataca, você pode gastar 2 PM. Se fizer isso…". É o impulso do Golpe Divino com outra origem: a
// ARMA que tem o encanto, e não um poder de classe. O motor oferece o gasto só na carta dessa arma, e a
// carta paga leva o que o livro dá: o dado a mais da Tumular, ou o que a mesa aplica no acerto — a
// condição (Venenosa, Congelante) ou o efeito com a fórmula escrita, sem rolar (Corrosiva, Elétrica),
// por decisão do dono.

// enchanted é um guerreiro de 5º nível com uma espada longa (item 7) que tem o encanto, e uma adaga sem
// encanto na outra mão (item 8).
func enchanted(enchant string) Character {
	ptr := func(s string) *string { return &s }
	ch := attributeSumHero("Guerreiro", 5, `[]`)
	ch.Items = []CharacterItem{
		{ID: 7, CatalogID: ptr("espada-longa"), Name: "Espada longa", Equipped: ptr("wielded"), Improvements: "[]", Enchants: []string{enchant}},
		{ID: 8, CatalogID: ptr("adaga"), Name: "Adaga", Equipped: ptr("wielded"), Improvements: "[]"},
	}
	return ch
}

// enchantOffer é a oferta do encanto na ficha, e se ela existe.
func enchantOffer(world *Ruleset, ch Character, enchant string) (BoostOffer, bool) {
	for _, offer := range world.BoostOffers(ch) {
		if offer.PowerID == EnchantBoostID(7, enchant) {
			return offer, true
		}
	}
	return BoostOffer{}, false
}

func TestTheEnchantOffersTwoPMOnlyOnItsWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := enchanted("encanto-tumular")
	offer, found := enchantOffer(world, ch, "encanto-tumular")
	if !found {
		t.Fatalf("a espada tumular não oferece o gasto de 2 PM; as ofertas são %+v", world.BoostOffers(ch))
	}
	if offer.Name != "Tumular" || offer.BasePM != 2 || offer.MaxSteps != 0 || offer.ItemID != 7 {
		t.Errorf("a oferta da Tumular é %+v; quer o nome do encanto, 2 PM, sem degraus, e o item 7", offer)
	}
	// A ADAGA não tem o encanto: o gasto não vale nela.
	cards, cost, err := world.BoostedWeaponCards(ch, nil, BoostChoice{PowerID: offer.PowerID})
	if err != nil {
		t.Fatalf("gastar os 2 PM da Tumular: %v", err)
	}
	if cost != 2 {
		t.Errorf("o gasto custa %d PM, e o livro diz 2", cost)
	}
	plain := world.ComputeWeaponCards(ch, nil)
	if len(cards[1].ExtraDamage) != len(plain[1].ExtraDamage) {
		t.Errorf("a adaga ganhou o dado da Tumular: %+v", cards[1].ExtraDamage)
	}
	// Sem a espada tumular na mão, a oferta some.
	ch.Items[0].Enchants = nil
	if _, found := enchantOffer(world, ch, "encanto-tumular"); found {
		t.Error("a oferta da Tumular ficou numa espada sem o encanto")
	}
}

// p336 "A arma causa +1d8 de dano de trevas. […] Se fizer isso, o bônus de dano aumenta para +2d8".
func TestTheTomblikeEnchantRaisesItsDieToTwoD8(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := enchanted("encanto-tumular")
	countTrevas := func(card WeaponCard) int {
		n := 0
		for _, extra := range card.ExtraDamage {
			if extra.Type == "trevas" {
				count, _, _ := strings.Cut(extra.Dice, "d")
				n += map[string]int{"1": 1, "2": 2}[count]
			}
		}
		return n
	}
	plain := world.ComputeWeaponCards(ch, nil)[0]
	cards, _, err := world.BoostedWeaponCards(ch, nil, BoostChoice{PowerID: EnchantBoostID(7, "encanto-tumular")})
	if err != nil {
		t.Fatal(err)
	}
	if countTrevas(plain) != 1 || countTrevas(cards[0]) != 2 {
		t.Errorf("os d8 de trevas são %d sem o gasto e %d com ele; o livro diz 1 e 2 (%+v)",
			countTrevas(plain), countTrevas(cards[0]), cards[0].ExtraDamage)
	}
}

// Venenosa e Congelante põem uma CONDIÇÃO na vítima (p336, p335); Corrosiva e Elétrica, um EFEITO que a
// mesa aplica noutra hora ou noutro alvo (p335). O anúncio só existe na carta paga.
func TestTheEnchantAnnouncesWhatTheBookForeseesOnlyWhenPaid(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		enchant, condition, effect string
	}{
		{"encanto-venenosa", "envenenado", ""},
		{"encanto-congelante", "enredado", ""},
		{"encanto-corrosiva", "", "4d4"},
		{"encanto-eletrica", "", "3d8"},
	} {
		t.Run(c.enchant, func(t *testing.T) {
			ch := enchanted(c.enchant)
			foresees := func(card WeaponCard) bool {
				for _, a := range card.Announces {
					if c.condition != "" && a.Condition == c.condition {
						return true
					}
					if c.effect != "" && a.Condition == "" && strings.Contains(a.Effect, c.effect) {
						return true
					}
				}
				return false
			}
			if foresees(world.ComputeWeaponCards(ch, nil)[0]) {
				t.Error("a carta sem o gasto já anuncia o efeito dele")
			}
			cards, _, err := world.BoostedWeaponCards(ch, nil, BoostChoice{PowerID: EnchantBoostID(7, c.enchant)})
			if err != nil {
				t.Fatal(err)
			}
			if !foresees(cards[0]) {
				t.Errorf("a carta paga não anuncia %q%q: %+v", c.condition, c.effect, cards[0].Announces)
			}
		})
	}
}

// p335 "A arma é letal contra um tipo de criatura (ou uma raça de humanoides). Uma vez por rodada, quando ataca
// uma criatura desse tipo, você pode gastar 2 PM. Se fizer isso e acertar o ataque, causa +4d8 de dano." O tipo
// é escolhido no encanto, e a oferta o leva para a tela dizer contra quem o gasto vale. Quem confere o alvo é o
// mestre (decisão do dono): o gasto é oferecido contra qualquer alvo.
func TestTheAnticreatureEnchantBuysFourD8AgainstTheChosenKind(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := enchanted("encanto-anticriatura")
	ch.Items[0].EnchantChoices = map[string]string{"encanto-anticriatura": "morto-vivo"}
	offer, found := enchantOffer(world, ch, "encanto-anticriatura")
	if !found {
		t.Fatalf("a espada anticriatura não oferece o gasto de 2 PM; as ofertas são %+v", world.BoostOffers(ch))
	}
	if offer.BasePM != 2 || offer.Against != "morto-vivo" || len(offer.Buys) != 1 || offer.Buys[0] != "+4d8" {
		t.Errorf("a oferta da Anticriatura é %+v; quer 2 PM, contra morto-vivo, comprando +4d8", offer)
	}
	d8s := func(card WeaponCard) []string {
		out := []string{}
		for _, extra := range card.ExtraDamage {
			out = append(out, extra.Dice)
		}
		return out
	}
	plain := world.ComputeWeaponCards(ch, nil)
	cards, _, err := world.BoostedWeaponCards(ch, nil, BoostChoice{PowerID: offer.PowerID})
	if err != nil {
		t.Fatalf("gastar os 2 PM da Anticriatura: %v", err)
	}
	if got := d8s(cards[0]); len(got) != len(d8s(plain[0]))+1 || got[len(got)-1] != "4d8" {
		t.Errorf("os dados extras da espada paga são %v, e sem o gasto %v; o livro dá +4d8", got, d8s(plain[0]))
	}
	if len(cards[1].ExtraDamage) != len(plain[1].ExtraDamage) {
		t.Errorf("a adaga ganhou os 4d8 da espada: %+v", cards[1].ExtraDamage)
	}
}

// p335 "A arma causa +1d6 de dano de fogo. Uma vez por rodada, quando ataca, você pode gastar 2 PM. Se fizer
// isso, em vez do ataque normal você dispara uma bola de fogo contra um alvo em alcance médio. O alvo sofre
// 6d6 pontos de dano. Um teste de Reflexos (CD For ou Des, à sua escolha) reduz à metade." O gasto TROCA o
// golpe, e por isso não é oferecido na linha de ataque: é um gesto próprio, da arma que tem o encanto (decisão
// do dono).
func TestTheFlamingEnchantFiresABlastInsteadOfTheStrike(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := enchanted("encanto-flamejante")
	if _, found := enchantOffer(world, ch, "encanto-flamejante"); found {
		t.Error("a bola de fogo da Flamejante foi oferecida como gasto do golpe; ela é em vez do golpe")
	}
	blasts := world.EnchantBlasts(ch)
	if len(blasts) != 1 {
		t.Fatalf("a espada flamejante dispara %d bolas de fogo; quer uma, só dela: %+v", len(blasts), blasts)
	}
	b := blasts[0]
	want := BoostInstead{Name: "Bola de fogo", Damage: "6d6", DamageType: "fogo", Range: "médio", Save: "Reflexos",
		SaveAttributes: []string{"strength", "dexterity"}, OnSave: "reduz à metade"}
	if b.ID != EnchantBoostID(7, "encanto-flamejante") || b.ItemID != 7 || b.Weapon != "Espada longa" || b.PM != 2 ||
		b.Name != want.Name || b.Damage != want.Damage || b.DamageType != want.DamageType || b.Range != want.Range ||
		b.Save != want.Save || !slices.Equal(b.SaveAttributes, want.SaveAttributes) || b.OnSave != want.OnSave {
		t.Errorf("a bola de fogo é %+v; quer a da espada (item 7) por 2 PM, com %+v", b, want)
	}
	// Guardada, a espada não dispara: a bola de fogo é da arma em punho, como o gasto do golpe.
	ch.Items[0].Equipped = nil
	if blasts := world.EnchantBlasts(ch); len(blasts) != 0 {
		t.Errorf("a espada guardada dispara %+v", blasts)
	}
}
