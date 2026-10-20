package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A TABELA 3-2 INTEIRA, transcrita do livro (p143).
//
// Uma linha por "Normal", e as seis colunas na ordem que a página imprime:
// −2, −1, Normal, +1, +2, +3. Escrita à mão e conferida contra a página — nada
// aqui é derivado do código sob teste, que é o jeito de um oráculo de tabela
// mentir.
var tabela32 = []struct {
	normal             string
	menosDois, menosUm string
	maisUm, maisDois   string
	maisTres           string
}{
	{normal: "1d3", menosDois: "1", menosUm: "1d2", maisUm: "1d4", maisDois: "1d6", maisTres: "1d8"},
	{normal: "1d4", menosDois: "1d2", menosUm: "1d3", maisUm: "1d6", maisDois: "1d8", maisTres: "1d10"},
	{normal: "1d6", menosDois: "1d3", menosUm: "1d4", maisUm: "1d8", maisDois: "1d10", maisTres: "1d12"},
	{normal: "1d8", menosDois: "1d4", menosUm: "1d6", maisUm: "1d10", maisDois: "1d12", maisTres: "3d6"},
	{normal: "1d10", menosDois: "1d6", menosUm: "1d8", maisUm: "1d12", maisDois: "3d6", maisTres: "4d6"},
	{normal: "1d12", menosDois: "1d8", menosUm: "1d10", maisUm: "3d6", maisDois: "4d6", maisTres: "4d8"},
	{normal: "2d8", menosDois: "1d10", menosUm: "2d6", maisUm: "3d8", maisDois: "4d8", maisTres: "4d10"},
	{normal: "2d10", menosDois: "2d6", menosUm: "2d8", maisUm: "3d10", maisDois: "4d10", maisTres: "4d12"},
}

// CADA CÉLULA DA TABELA 3-2 É UMA ASSERÇÃO (p143).
func TestEveryCellOfTheDamageStepTableIsWalked(t *testing.T) {
	medidas := 0
	for _, linha := range tabela32 {
		for passos, quer := range map[int]string{
			-2: linha.menosDois, -1: linha.menosUm, 0: linha.normal,
			1: linha.maisUm, 2: linha.maisDois, 3: linha.maisTres,
		} {
			if deu := DamageStep(linha.normal, passos); deu != quer {
				t.Errorf("%s em %+d passos deu %q, e a Tabela 3-2 (p143) diz %q",
					linha.normal, passos, deu, quer)
			}
			medidas++
		}
	}
	if medidas != 48 {
		t.Fatalf("o guarda mediu %d células e a Tabela 3-2 tem 48: a transcrição encolheu", medidas)
	}
}

// OS APELIDOS ENTRAM PELA LINHA EM QUE O LIVRO OS IMPRIME NA COLUNA NORMAL.
//
// Duas células da coluna Normal têm mais de uma notação — "1d8 ou 2d4" e
// "1d12, 2d6 ou 3d4" —, e uma arma com qualquer uma delas anda pela linha
// daquela célula. É o que faz da coluna Normal a PORTA DE ENTRADA da tabela.
func TestTheAliasesOfTheNormalColumnWalkTheirOwnRow(t *testing.T) {
	for _, caso := range []struct {
		notacao, igualA, maisUm string
	}{
		{notacao: "2d4", igualA: "1d8", maisUm: "1d10"},
		{notacao: "2d6", igualA: "1d12", maisUm: "3d6"},
		{notacao: "3d4", igualA: "1d12", maisUm: "3d6"},
	} {
		if deu := DamageStep(caso.notacao, 1); deu != caso.maisUm {
			t.Errorf("%s +1 passo deu %q, e a p143 imprime %s na mesma célula Normal que %s, cujo +1 é %q",
				caso.notacao, deu, caso.notacao, caso.igualA, caso.maisUm)
		}
	}
}

// A TABELA NÃO É ASSOCIATIVA, e é por isso que o passo se aplica DE UMA VEZ.
//
// `2d8 +1` é `3d8` e `2d10 −1` é `2d8` — então somar dois passos de um em um
// passaria por 2d10 e daria 3d10, que não é o que a linha do 2d8 imprime. O
// degrau total entra numa chamada só, e esta é a célula que denuncia quem o
// compuser.
func TestTheStepIsReadInOneGoBecauseTheTableIsNotAssociative(t *testing.T) {
	if deu := DamageStep("2d8", 2); deu != "4d8" {
		t.Errorf("2d8 +2 deu %q, e a linha do 2d8 na p143 diz 4d8 — somar +1 duas vezes daria 4d10", deu)
	}
	if deu := DamageStep("2d10", -1); deu != "2d8" {
		t.Errorf("2d10 −1 deu %q, e a p143 diz 2d8", deu)
	}
}

// O PISO E O TETO são da própria tabela: ela abre no `1` e fecha em
// "4d12 (máximo)".
func TestTheLadderStopsAtTheEndsTheTablePrints(t *testing.T) {
	if deu := DamageStep("1d3", -3); deu != "1" {
		t.Errorf("1d3 −3 deu %q: a coluna mais baixa da p143 é o 1, e não há degrau abaixo dela", deu)
	}
	if deu := DamageStep("2d10", 4); deu != "4d12" {
		t.Errorf("2d10 +4 deu %q: a p143 escreve \"4d12 (máximo)\" na ponta de cima", deu)
	}
}

// ARMA DUPLA ANDA AS DUAS PONTAS: "cada «ponta» conta como uma arma separada
// para efeitos de melhorias e encantos" (p143).
func TestADoubleWeaponWalksBothEnds(t *testing.T) {
	if deu := DamageStep("1d6/1d6", 1); deu != "1d8/1d8" {
		t.Errorf("1d6/1d6 +1 deu %q, quer 1d8/1d8", deu)
	}
	if deu := DamageStep("1d10/1d12", 1); deu != "1d12/3d6" {
		t.Errorf("1d10/1d12 +1 deu %q, quer 1d12/3d6 — as pontas andam cada uma pela linha dela", deu)
	}
}

// O QUE NÃO TEM DADO NÃO ANDA, e passo zero não mexe em nada.
//
// A rede do catálogo causa dano "0"; inventar um degrau para ela seria inventar
// regra. Notação que a tabela não imprime volta intacta pela mesma razão.
func TestWhatTheTableDoesNotPrintComesBackUntouched(t *testing.T) {
	for _, caso := range []struct {
		notacao string
		passos  int
	}{
		{"0", 1}, {"1d8", 0}, {"5d6", 1}, {"", 2},
	} {
		if deu := DamageStep(caso.notacao, caso.passos); deu != caso.notacao {
			t.Errorf("%q em %+d passos deu %q, e devia voltar intacto",
				caso.notacao, caso.passos, deu)
		}
	}
}

// TODA ARMA DO CATÁLOGO SABE ANDAR, e este é o denominador da forma.
//
// Um passo que não acha a notação devolve o dado original — silêncio com cara
// de "não tem modificador". Este guarda pergunta o contrário: existe arma no
// catálogo que a Tabela 3-2 não alcança? A rede (dano "0") é a exceção, e ela
// está nomeada.
func TestEveryWeaponInTheCatalogCanWalkTheTable(t *testing.T) {
	bruto, err := os.ReadFile(filepath.Join("..", "catalog", "data", "items.json"))
	if err != nil {
		t.Fatalf("ler o catálogo de itens: %v", err)
	}
	var itens []struct {
		ID     string `json:"id"`
		Weapon *struct {
			Damage string `json:"damage"`
		} `json:"weapon"`
	}
	if err := json.Unmarshal(bruto, &itens); err != nil {
		t.Fatalf("itens.json: %v", err)
	}
	semDado := map[string]bool{"0": true}
	medidas := 0
	for _, item := range itens {
		if item.Weapon == nil || semDado[item.Weapon.Damage] {
			continue
		}
		medidas++
		for _, ponta := range strings.Split(item.Weapon.Damage, "/") {
			if DamageStep(ponta, 1) == ponta {
				t.Errorf("a arma %s causa %s e a Tabela 3-2 não a alcança: um passo nela "+
					"devolveria o mesmo dado, em silêncio", item.ID, ponta)
			}
		}
	}
	if medidas < 30 {
		t.Fatalf("o guarda viu só %d armas: ele parou de achar o que veio medir", medidas)
	}
}

// O CAMPEÃO ANDA O DADO DE TODA ARMA (p66), e o 20º nível é a fronteira.
//
// "No 20º nível, o dano de todos os seus ataques aumenta em um passo" — o
// escopo `all` do guerreiro contra o `this` do material: lá o passo é da arma
// que tem o material, aqui é de quem ataca.
//
// O caso é de INTEGRAÇÃO de propósito: o modificador só vale se a coleta de
// poder de classe o alcançar, e um teste sobre o `DamageStep` sozinho passaria
// verde com o verbete do catálogo inerte — que é o defeito que o guarda de
// alvos existe para não deixar acontecer.
func TestTheChampionWalksTheDieOfEveryWeapon(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	cardAtLevel := func(level int) WeaponCard {
		cards := BookRuleset(catalogs).ComputeWeaponCards(Character{
			Classes: []CharacterClass{{ClassName: "Guerreiro", Level: level}},
			Items: []CharacterItem{{
				CatalogID: ptr("espada-longa"), Name: "Espada longa",
				Equipped: ptr("wielded"), Improvements: "[]",
			}},
		}, map[string]bool{})
		if len(cards) == 0 {
			t.Fatalf("o guerreiro de nível %d não virou carta", level)
		}
		return cards[0]
	}

	// O CONTROLE É O 19º: um nível abaixo do capstone, a espada é a do
	// catálogo. Sem ele, um passo que valesse para todo guerreiro passaria
	// verde no caso de cima.
	if got := cardAtLevel(19).Damage; got != "1d8" {
		t.Fatalf("o guerreiro de 19º causa %q com a espada longa, e o Campeão é do 20º (p66)", got)
	}
	if got := cardAtLevel(20).Damage; got != "1d10" {
		t.Errorf("o guerreiro de 20º causa %q: o Campeão anda um passo (p66) e a Tabela 3-2 "+
			"leva 1d8 a 1d10", got)
	}
}
