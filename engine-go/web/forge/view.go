package forge

import (
	"fmt"
	"sort"
	"strings"

	"t20engine/book"
	"t20engine/engine"
	"t20engine/plataforma"
)

// A FOLHA EM BRANCO DA FORJA (ALE-272, fatia 9) — o que a cena desenha.
//
// A cena inteira sai daqui montada: as cartas com o que cada escolha COMPRA (os
// atributos da raça, o PV e o PM da classe), as origens, e o equipamento de
// p140 já estreitado pela classe escolhida. A tela não decide nada — ela não
// sabe que arcanista não veste armadura nem que bardo leva arma marcial; ela
// desenha as opções que este arquivo lhe entrega, e o servidor recusa o que não
// couber (`forgeRefusals`).

// forgeView é a folha inteira.
type forgeView struct {
	Name    string
	Races   []raceCard
	Classes []classCard
	Origins []originOption
	// Equipamento é nil enquanto não há classe escolhida: o kit de p140 só se
	// conhece pela classe, e oferecer uma armadura antes disso seria oferecer a
	// escolha errada para o arcanista.
	Gear   *startingGear
	Errors plataforma.FieldErrorMap
	// OrphanRefusals são as recusas cujo campo NÃO está na tela.
	//
	// Elas existem porque a folha só desenha o que o kit oferece: mandar uma
	// arma marcial para um ladino é recusado, e o seletor de arma marcial não
	// existe na folha dele — a frase não teria onde aparecer, e a pessoa veria
	// a folha voltar sem uma palavra. Isso só acontece com um POST feito na mão
	// ou com uma folha respondida antes de trocar a classe, e as duas merecem
	// resposta.
	OrphanRefusals []string
}

// raceCard é uma das 17 cartas da linhagem.
type raceCard struct {
	Name         string
	Attributes   string
	Size         string
	Displacement int
	Abilities    string
	Chosen       bool
}

// classCard é uma das 14 cartas do ofício.
type classCard struct {
	Name       string
	PV         int
	PM         int
	Expertises string
	Chosen     bool
}

// originOption é uma linha da lista de origens.
type originOption struct {
	Name    string
	Benefit string
	Chosen  bool
}

// startingGear é o kit de p140 já reduzido ao que ESTA classe recebe.
type startingGear struct {
	SimpleWeapons  []itemOption
	MartialWeapons []itemOption
	Armors         []itemOption
	// Escudo é o rótulo do escudo do kit, ou "" quando a classe não usa escudos.
	Shield       string
	ShieldChosen bool
	// AResolver são as concessões de origem que são ESCOLHA e não item — elas
	// não nascem na mochila, e a folha diz isso em vez de deixar o jogador
	// descobrir que faltou algo.
	ToResolve []string
}

// itemOption é uma entrada de um dos seletores do equipamento.
type itemOption struct {
	ID     string
	Label  string
	Chosen bool
}

// blankForgeSheet monta a cena a partir das respostas que já existem.
//
// Ela é chamada nas três situações e devolve a mesma coisa nas três: a folha
// vazia do primeiro GET, a folha redesenhada quando a classe muda, e a folha
// recusada com os erros por campo.
func blankForgeSheet(folha forgeAnswers, erros plataforma.FieldErrorMap) forgeView {
	racas, classes, _ := book.CharacterCatalogs()
	v := forgeView{Name: folha.Name, Errors: erros}
	for _, raca := range racas {
		v.Races = append(v.Races, raceCardOf(raca, folha.Race))
	}
	for _, classe := range classes {
		v.Classes = append(v.Classes, classCardOf(classe, folha.Class))
	}
	v.Origins = originOptions(folha.Origin)
	if classe := classByName(folha.Class); classe != nil {
		v.Gear = startingGearFor(folha, *classe)
	}
	v.OrphanRefusals = orphanRefusals(erros, v.Gear)
	return v
}

// orphanRefusals junta as recusas dos campos que esta folha não desenha.
func orphanRefusals(erros plataforma.FieldErrorMap, gear *startingGear) []string {
	naTela := map[string]bool{"name": true, "race": true, "class": true, "origin": true}
	if gear != nil {
		naTela["weaponSimple"] = true
		naTela["weaponMartial"] = len(gear.MartialWeapons) > 0
		naTela["armor"] = len(gear.Armors) > 0
		naTela["shield"] = gear.Shield != ""
	}
	var orfas []string
	for campo, mensagens := range erros {
		if naTela[campo] {
			continue
		}
		orfas = append(orfas, mensagens...)
	}
	sort.Strings(orfas)
	return orfas
}

func raceCardOf(raca book.Race, escolhida string) raceCard {
	nomes := make([]string, 0, 2)
	for _, habilidade := range raca.Abilities {
		if len(nomes) == 2 {
			break
		}
		nomes = append(nomes, habilidade.Name)
	}
	return raceCard{
		Name: raca.Name, Attributes: raca.AttributeMod.Escrito(),
		Size: raca.Tamanho, Displacement: raca.Deslocamento,
		Abilities: strings.Join(nomes, ", "), Chosen: raca.Name == escolhida,
	}
}

func classCardOf(classe book.Class, escolhida string) classCard {
	pv, pm, _ := engine.ClassStartingVitals(classe.Name)
	return classCard{
		Name: classe.Name, PV: pv, PM: pm,
		Expertises: cardExpertisesLine(classe), Chosen: classe.Name == escolhida,
	}
}

// cardExpertisesLine escreve a linha "Perícias" do bloco da classe: as que vêm
// treinadas de saída e quantas ainda se escolhem.
//
//	"Fortitude · mais 2 a escolher"
func cardExpertisesLine(classe book.Class) string {
	fixas := strings.Join(classe.Pericias, ", ")
	if classe.Escolhe == 0 {
		return fixas
	}
	escolha := fmt.Sprintf("mais %d a escolher", classe.Escolhe)
	if fixas == "" {
		return escolha
	}
	return fixas + " · " + escolha
}

// originOptions são as 35 origens com uma linha do que elas dão.
func originOptions(escolhida string) []originOption {
	origens := book.Origins()
	nomes := make([]string, 0, len(origens))
	for nome := range origens {
		nomes = append(nomes, nome)
	}
	book.SortByName(nomes, func(n string) string { return n })

	lista := make([]originOption, 0, len(nomes))
	for _, nome := range nomes {
		lista = append(lista, originOption{
			Name: nome, Benefit: benefitsLine(origens[nome]),
			Chosen: nome == escolhida,
		})
	}
	return lista
}

// benefitsLine resume a lista de benefícios da origem. São dois a
// escolher (p85), e a escolha é da ficha — aqui é só o que a origem oferece.
func benefitsLine(origem book.Origin) string {
	nomes := make([]string, 0, len(origem.Benefits))
	for _, beneficio := range origem.Benefits {
		nomes = append(nomes, beneficio.Name)
	}
	return strings.Join(nomes, ", ")
}

// startingGearFor monta os seletores do kit desta classe.
func startingGearFor(folha forgeAnswers, classe book.Class) *startingGear {
	kit := engine.StartingKitFor(classe.Name, classe.Proficiencias)
	eq := &startingGear{
		SimpleWeapons: itemOptionsInCategory("weapon-simple", folha.SimpleWeapon),
		Armors:        itemOptionsByID(kit.Armors, folha.Armor),
		ToResolve:     grantsToResolve(folha.Origin),
	}
	if kit.MartialWeapon {
		eq.MartialWeapons = itemOptionsInCategory("weapon-martial", folha.MartialWeapon)
	}
	if item := book.ItemByID(kit.Shield); item != nil {
		eq.Shield, eq.ShieldChosen = item.Name, folha.Shield
	}
	return eq
}

// itemOptionsInCategory são todos os itens de uma categoria do catálogo, na ordem
// em que o acervo já os guarda.
func itemOptionsInCategory(categoria, escolhido string) []itemOption {
	var opcoes []itemOption
	for _, item := range book.Catalogs().Itens {
		if item.Category == categoria {
			opcoes = append(opcoes, itemOptionOf(item, escolhido))
		}
	}
	return opcoes
}

// itemOptionsByID são os itens que o kit nomeia, na ordem do kit — as armaduras
// leves antes da brunea, como o livro as escreve.
func itemOptionsByID(ids []string, escolhido string) []itemOption {
	opcoes := make([]itemOption, 0, len(ids))
	for _, id := range ids {
		if item := book.ItemByID(id); item != nil {
			opcoes = append(opcoes, itemOptionOf(*item, escolhido))
		}
	}
	return opcoes
}

func itemOptionOf(item book.Item, escolhido string) itemOption {
	return itemOption{ID: item.ID, Label: item.Name, Chosen: item.ID == escolhido}
}

// grantsToResolve são as linhas "Itens" da origem que pedem uma decisão —
// "Estojo de disfarces OU gazua", "Arma marcial", "Um item estrangeiro (até T$
// 100)". Elas não nascem na mochila; a folha as anuncia e a Mochila as resolve.
func grantsToResolve(origem string) []string {
	var rotulos []string
	for _, concessao := range originGrants(origem) {
		switch concessao.Kind {
		case engine.OriginItemFixed, engine.OriginItemMoney:
			continue
		default:
			rotulos = append(rotulos, concessao.Label)
		}
	}
	return rotulos
}
