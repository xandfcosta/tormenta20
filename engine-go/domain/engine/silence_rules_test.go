package engine

import (
	"path/filepath"
	"testing"
)

// TODO TERMO DO CATÁLOGO TEM ENDEREÇO ÚNICO DENTRO DA FONTE DELE (ALE-387).
//
// É o que faz o silêncio do mestre ser cirúrgico. Duas contribuições da mesma
// fonte no mesmo endereço não podem ser distinguidas, e calar uma mataria as
// duas — sem aviso, com a ficha apenas mostrando um número menor.
//
// MEDIDO quando a regra nasceu: 246 termos, zero colisões. Os discriminadores
// que fazem isso valer são a ESCALA e a CONDIÇÃO — o anão tem `maxPv +2` e
// `maxPv +1 por nível`, e a Força da Natureza do druida tem `pmCost -2` e o
// mesmo `-2` em terreno natural. Sem esses dois campos no endereço, esses são
// justamente os pares que colidiriam.
//
// O DENOMINADOR é afirmado junto: uma varredura que deixasse de achar as fontes
// teria zero colisões, que no terminal é a mesma cor de "está tudo único".
func TestEveryCatalogTermHasAUniqueAddress(t *testing.T) {
	const wantAtLeast = 200

	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	medidos := 0
	visto := map[string]string{} // endereço -> a descrição do primeiro que o ocupou
	// UMA FONTE SE MEDE UMA VEZ. O mesmo benefício de origem (`poder-atraente`,
	// `poder-sentidos-agucados`) é oferecido por VÁRIAS origens, e o coletor o lê
	// uma vez só — o `getOriginBenefit` devolve o primeiro que casa o id. Medir
	// cada cópia acusaria colisão de uma fonte consigo mesma.
	jaMedida := map[string]bool{}
	conferir := func(sourceID, what string, mods []Modifier) {
		if jaMedida[sourceID] {
			return
		}
		jaMedida[sourceID] = true
		for _, m := range mods {
			medidos++
			address := TermID(sourceID, m)
			quem := what + " (amount " + itoaForTest(m.Amount) + ")"
			if antes, taken := visto[address]; taken {
				t.Errorf("dois termos dividem o endereço %q:\n  %s\n  %s\n"+
					"Calar um mataria os dois. Distinga-os pela ESCALA ou pela CONDIÇÃO,\n"+
					"que são os campos que o endereço carrega.", address, antes, quem)
				continue
			}
			visto[address] = quem
		}
	}

	// conferirContra mede uma ALTERNATIVA contra a base da fonte, sem deixá-la no
	// mapa: duas variantes da mesma habilidade nunca convivem numa ficha.
	conferirContra := func(sourceID string, base []Modifier, what string, mods []Modifier) {
		ocupados := map[string]bool{}
		for _, m := range base {
			ocupados[TermID(sourceID, m)] = true
		}
		for _, m := range mods {
			medidos++
			if address := TermID(sourceID, m); ocupados[address] {
				t.Errorf("a variante ocupa um endereço que a BASE da fonte já usa: %q\n  %s",
					address, what)
			}
		}
	}

	for id, item := range book.itemsByID {
		conferir(id, "item "+id, item.Modifiers)
	}
	// A RAÇA É UMA FONTE SÓ: o `raceActiveItems` junta os modificadores de TODAS
	// as habilidades dela num `ActiveItem` com o `race.ID`. Medir habilidade por
	// habilidade deixaria passar duas que disputassem o mesmo endereço.
	//
	// As VARIANTES são a exceção, e por serem alternativas: só uma é escolhida,
	// então duas variantes da mesma habilidade podem ocupar o mesmo endereço sem
	// nunca se encontrarem numa ficha. Cada uma é conferida contra a BASE, e
	// nenhuma contra a outra.
	for id, race := range book.racesByID {
		base := []Modifier{}
		for _, ability := range race.Abilities {
			base = append(base, ability.Modifiers...)
		}
		conferir(id, "raça "+id, base)
		for _, ability := range race.Abilities {
			for _, variant := range ability.Variants {
				conferirContra(id, base, "raça "+id+" variante "+variant.ID, variant.Modifiers)
			}
		}
	}
	for _, origin := range book.origins {
		for _, benefit := range origin.Benefits {
			conferir(benefit.ID, "benefício "+benefit.ID, benefit.Modifiers)
		}
		conferir(origin.UniquePower.ID, "poder único "+origin.UniquePower.ID, origin.UniquePower.Modifiers)
	}
	for _, power := range book.classPowers {
		conferir(power.ID, "poder de classe "+power.ID, power.Modifiers)
	}
	for id, power := range book.generalByID {
		conferir(id, "poder geral "+id, power.Modifiers)
	}
	for name, power := range book.grantedByName {
		conferir(name, "poder concedido "+name, power.Modifiers)
	}

	t.Logf("endereçados %d termos em %d fontes", medidos, len(jaMedida))
	if medidos < wantAtLeast {
		t.Fatalf("a varredura endereçou %d termos e o piso é %d — ela parou de achar as fontes do catálogo",
			medidos, wantAtLeast)
	}
}

// itoaForTest existe só para a mensagem acima não puxar `strconv` para um
// arquivo de teste que não faz mais nada com números.
func itoaForTest(n int) string {
	if n == 0 {
		return "0"
	}
	sign, digits := "", ""
	if n < 0 {
		sign, n = "-", -n
	}
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return sign + digits
}

// O SILÊNCIO TIRA UM TERMO E DEIXA OS IRMÃOS DA MESMA FONTE (ALE-387).
//
// É a granularidade que a mesa pediu: o medalhão de prata continua concedendo o
// limite de PM da p160 depois de o mestre calar o bônus de Luta que a mesa tinha
// acrescentado.
//
// O caso usa uma fonte com DOIS termos de propósito. Com um só, calar o termo e
// calar a fonte inteira dão o mesmo resultado, e o guarda mediria a metade em
// que o defeito é invisível.
func TestSilencingOneTermLeavesTheOthersOfTheSameSource(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	wielded := "wielded"
	id := "medalhao-de-prata"
	ch := Character{
		ID:         7,
		Expertises: []CharacterExpertise{{Name: "Luta", Attribute: "strength"}},
		Items:      []CharacterItem{{Name: "Medalhão de prata", CatalogID: &id, Equipped: &wielded}},
	}
	daMesa := Modifier{Target: ModifierTarget{K: "expertise", Name: "Luta"}, Amount: 1, BonusType: "untyped"}
	emendado := Amendments{Entries: map[string][]Modifier{id: {daMesa}}}

	termosDe := func(world *Ruleset) []string {
		out := []string{}
		for _, item := range world.ActiveItemsFor(ch) {
			if item.SourceID != id {
				continue
			}
			for _, m := range item.Modifiers {
				out = append(out, targetKey(m.Target))
			}
		}
		return out
	}

	antes := termosDe(RulesetOf(book, emendado))
	if len(antes) != 2 {
		t.Fatalf("o medalhão emendado devia ter 2 termos e tem %d: %v", len(antes), antes)
	}

	calado := emendado
	calado.Silences = []Silence{{Applies: EveryoneIn(), Term: TermID(id, daMesa)}}
	depois := termosDe(RulesetOf(book, calado))

	if len(depois) != 1 {
		t.Fatalf("depois do silêncio sobraram %d termos e esperava 1: %v", len(depois), depois)
	}
	if depois[0] != "pmLimit" {
		t.Errorf("sobrou %q e o que tinha de sobrar é o `pmLimit` do LIVRO — o silêncio comeu o irmão errado",
			depois[0])
	}
}

// O SILÊNCIO TAMBÉM TEM ESCOPO, e o vizinho de mesa continua com o termo.
func TestASilenceReachesOnlyWhoTheSelectorNames(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	umaLuta := Modifier{Target: ModifierTarget{K: "expertise", Name: "Luta"}, Amount: 1, BonusType: "untyped"}
	mundo := RulesetOf(book, Amendments{
		Grants: []CampaignGrant{
			{ID: "g-mesa", Applies: EveryoneIn(), Label: "Bênção da mesa", Modifiers: []Modifier{umaLuta}},
		},
		Silences: []Silence{
			{Applies: OnlyCharacter(7), Term: TermID(CampaignGrantSource+"g-mesa", umaLuta)},
		},
	})

	lutaDe := func(id int) int {
		ch := Character{ID: id, Expertises: []CharacterExpertise{{Name: "Luta", Attribute: "strength"}}}
		for _, ex := range mundo.ComputeSheet(ch, nil).Expertises {
			if ex.Name == "Luta" {
				return ex.Total
			}
		}
		t.Fatal("a perícia Luta não apareceu na ficha")
		return 0
	}

	if calado, ouvindo := lutaDe(7), lutaDe(9); calado != ouvindo-1 {
		t.Errorf("o herói 7 (calado) tirou %d e o 9 (não calado) tirou %d — esperava exatamente 1 de diferença.\n"+
			"Iguais quer dizer que o escopo do silêncio não foi respeitado.", calado, ouvindo)
	}
}

// UMA ERRATA DO LIVRO NÃO EVAPORA O SILÊNCIO DO MESTRE (ALE-387).
//
// É a razão de o endereço NÃO carregar o valor nem a prosa, e ela não aparece em
// nenhum dos outros casos: os dois lados deles calculam o endereço com a mesma
// função, então incluir o valor os deixaria verdes. O defeito é de TEMPO — o
// mestre cala um termo hoje, o catálogo corrige o número amanhã, e o silêncio
// deixa de casar sem uma palavra em lugar nenhum.
//
// O endereço que os condicionais usavam antes da convergência tinha exatamente
// essa fragilidade, e é por isso que ele não sobreviveu: hoje o opt-in do
// jogador e o silêncio do mestre são a MESMA string.
func TestTheTermAddressSurvivesAnErrataToTheAmount(t *testing.T) {
	doLivro := Modifier{
		Target:    ModifierTarget{K: "expertise", Name: "Luta"},
		Amount:    1,
		BonusType: "untyped",
		Note:      "a frase que a tela mostra",
	}
	errata := doLivro
	errata.Amount = 2
	errata.BonusType = "circumstance"
	errata.Note = "a frase reescrita"

	if antes, depois := TermID("x", doLivro), TermID("x", errata); antes != depois {
		t.Errorf("a errata mudou o endereço do termo:\n  antes:  %q\n  depois: %q\n"+
			"O silêncio que o mestre escreveu deixaria de casar, em silêncio.", antes, depois)
	}

	// E o CONTROLE: o endereço ainda distingue o que PRECISA distinguir, senão
	// ele passaria neste caso sendo constante.
	outroAlvo := doLivro
	outroAlvo.Target = ModifierTarget{K: "expertise", Name: "Pontaria"}
	if TermID("x", doLivro) == TermID("x", outroAlvo) {
		t.Error("dois alvos diferentes deram o mesmo endereço — ele virou constante")
	}
	comEscala := doLivro
	comEscala.Scale = &VitalScale{Per: "level"}
	if TermID("x", doLivro) == TermID("x", comEscala) {
		t.Error("a escala deixou de distinguir — é ela que separa os dois `maxPv` do anão")
	}
	comCondicao := doLivro
	comCondicao.Condition = &ModifierCondition{C: "terrain", Type: "natural"}
	if TermID("x", doLivro) == TermID("x", comCondicao) {
		t.Error("a condição deixou de distinguir — é ela que separa os dois `pmCost` do druida")
	}
}
