package api

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"t20engine/book"
	"t20engine/catalog"
	"t20engine/sheet"
)

// A aba MAGIAS como dado (ALE-272, fatia 6).
//
// O grimório do personagem: só o que ele APRENDEU, cada magia com preparar,
// esquecer e conjurar. "Aprender" abre o catálogo inteiro do Capítulo 4 com
// filtros de círculo, escola e busca.
//
// # As CONCEDIDAS aparecem mesmo para quem não conjura
//
// Um poder pode ensinar uma magia (Totem Espiritual, p42), e um bárbaro com
// Totem tem de ver a magia dele. Elas não estão no grimório — não se aprendem
// nem se esquecem —, então são um bloco próprio, acima.
//
// # O SERVIDOR é a autoridade do custo, e a tela só antecipa
//
// A SPA calcula o PM no cliente com `shared/rules` para poupar uma ida ao
// servidor nas recusas óbvias. Aqui não há pré-validação nenhuma: o `@post`
// responde em ~2ms e a recusa vem com a frase certa. O que a tela mostra é uma
// PRÉVIA do total, somada por expressão do Datastar sobre números que o servidor
// já mandou — e ela nunca decide nada.

// spellbookPanel é a aba Magias pronta para desenhar.
type spellbookPanel struct {
	Learned []learnedSpellRow
	Granted []grantedSpellRow
	// Catalogo são as magias que o personagem AINDA não sabe, já filtradas.
	Catalogo []catalogSpellRow
	// IsCaster diz se há classe conjuradora. Sem ela não há o que aprender — mas
	// as concedidas continuam aparecendo.
	IsCaster bool
	// RequiresPrep é Clérigo, Druida e o Arcanista do caminho `mago`: para eles
	// a magia precisa estar PREPARADA para ser conjurada.
	RequiresPrep bool
	// CastableCircle é o maior círculo que este personagem alcança, e é o que
	// TRANCA os aprimoramentos de círculo alto.
	CastableCircle int
	PmCurrent      int64
	Search         string
	Circulo        string
	Escola         string
	Escolas        []filterOption
}

type filterOption struct {
	Valor  string
	Rotulo string
	Ativo  bool
}

type learnedSpellRow struct {
	ID       string
	Name     string
	Circle   string
	School   string
	BasePm   int
	Prepared bool
	// CD é a Classe de Dificuldade dos testes contra esta magia, pelo
	// atributo-chave da classe que a concede.
	CD        string
	Augments  []augmentRow
	Execution string
	Range     string
	Duration  string
	Effect    string
	Page      int
	Command   string
}

// grantedSpellRow é a magia que um PODER ensinou.
type grantedSpellRow struct {
	Name   string
	Circle string
	Source string
	Effect string
	Page   int
}

type catalogSpellRow struct {
	ID      string
	Name    string
	Circle  string
	School  string
	Effect  string
	Page    int
	Command string
}

// augmentRow é um aprimoramento, com o que ele custa e se está trancado.
type augmentRow struct {
	Index int
	PM    int
	// Stacks diz se ele EMPILHA. `muda` não empilha — trocar o tipo de dano duas
	// vezes não é trocar duas vezes mais.
	Stacks      bool
	Description string
	// Locked é o aprimoramento que exige um círculo acima do alcançável. Ele
	// aparece com cadeado em vez de sumir: a pessoa merece saber que existe e
	// que ela ainda não chega lá.
	Locked         bool
	RequiredCircle int
}

// spellbookPanelOf monta a aba.
func (s *Server) spellbookPanelOf(dto sheet.CharacterDTO, busca, circulo, escola string) spellbookPanel {
	panel := spellbookPanel{
		IsCaster:       len(casterClassesOf(dto)) > 0,
		RequiresPrep:   requiresPreparation(dto.Classes, dto.ClassChoices),
		CastableCircle: highestCastableCircle(dto.Classes, 0),
		PmCurrent:      dto.MpCurrent,
		Search:         busca,
		Circulo:        circulo,
		Escola:         escola,
		Escolas:        schoolOptions(escola),
		Granted:        grantedSpellRowsOf(dto),
	}
	panel.Learned = learnedSpellRowsOf(s, dto, panel.CastableCircle)
	// Quem não conjura não aprende, e por isso não paga o catálogo: a lista
	// inteira do Capítulo 4 viajaria em toda cena da ficha de um guerreiro.
	if panel.IsCaster {
		panel.Catalogo = catalogSpellRowsOf(dto, busca, circulo, escola)
	}
	return panel
}

// casterClassesOf são as classes do personagem que conjuram.
func casterClassesOf(dto sheet.CharacterDTO) []string {
	nomes := []string{}
	for _, c := range dto.Classes {
		if _, conjura := spellProgressions()[c.ClassName]; conjura {
			nomes = append(nomes, c.ClassName)
		}
	}
	return nomes
}

// learnedSpellRowsOf é o grimório, por círculo e depois por nome.
func learnedSpellRowsOf(s *Server, dto sheet.CharacterDTO, castable int) []learnedSpellRow {
	linhas := []learnedSpellRow{}
	for _, aprendida := range dto.Spells {
		magia, conhecida := catalog.LookupSpell(aprendida.CatalogSpellID)
		if !conhecida {
			continue
		}
		doLivro := spellOfBook(aprendida.CatalogSpellID)
		linhas = append(linhas, learnedSpellRow{
			ID: aprendida.CatalogSpellID, Name: doLivro.Name,
			Circle: circleName(doLivro.Circle), School: schoolName(doLivro.School),
			BasePm: spellBasePmCost[doLivro.Circle], Prepared: aprendida.Prepared,
			CD:        s.spellCdOf(dto, magia),
			Augments:  augmentRowsOf(aprendida.CatalogSpellID, magia, castable),
			Execution: doLivro.Execution, Range: doLivro.Range, Duration: doLivro.Duration,
			Effect: doLivro.BaseEffect, Page: doLivro.BookPage,
			Command: aprendida.CatalogSpellID,
		})
	}
	sort.SliceStable(linhas, func(a, b int) bool {
		if linhas[a].Circle != linhas[b].Circle {
			return linhas[a].Circle < linhas[b].Circle
		}
		return linhas[a].Name < linhas[b].Name
	})
	return linhas
}

// spellCdOf é a CD dos testes contra a magia, pelo atributo-chave da classe.
//
// Ela vem do MOTOR, pelo mapa por atributo que a caixa "CD Magia" do Combate já
// usa: uma segunda conta aqui daria dois números para a mesma pergunta.
func (s *Server) spellCdOf(dto sheet.CharacterDTO, magia catalog.Spell) string {
	sheet, _, ok := s.sheetForPanels(dto)
	if !ok {
		return "—"
	}
	melhor := 0
	for _, classe := range dto.Classes {
		prog, conjura := spellProgressions()[classe.ClassName]
		if !conjura || !aceitaAClasse(magia.Classes, classe.ClassName) {
			continue
		}
		if cd := sheet.SpellCdByAttribute[prog.Attribute]; cd > melhor {
			melhor = cd
		}
	}
	if melhor == 0 {
		return "—"
	}
	return strconv.Itoa(melhor)
}

func aceitaAClasse(lista []string, nome string) bool {
	for _, c := range lista {
		if c == nome {
			return true
		}
	}
	return false
}

// augmentRowsOf traduz os aprimoramentos, trancando os fora de alcance.
func augmentRowsOf(spellID string, magia catalog.Spell, castable int) []augmentRow {
	linhas := make([]augmentRow, 0, len(magia.Augments))
	for i, a := range magia.Augments {
		linha := augmentRow{
			Index: i, PM: a.PmCost, Stacks: a.Kind != "muda",
			Description: augmentDescription(spellID, i),
		}
		if a.RequiresCircle != nil {
			linha.RequiredCircle = *a.RequiresCircle
			linha.Locked = *a.RequiresCircle > castable
		}
		linhas = append(linhas, linha)
	}
	return linhas
}

// augmentDescription lê o texto do aprimoramento do catálogo cru.
//
// O `catalog.Augment` do Go é um subconjunto deliberado e não carrega a
// descrição — mas a tela precisa dela, senão o jogador escolhe entre "1 PM" e
// "1 PM" sem saber o que cada um faz.
func augmentDescription(spellID string, index int) string {
	doLivro := spellOfBook(spellID)
	if index < len(doLivro.Augments) {
		return doLivro.Augments[index].Description
	}
	return ""
}

// spellOfBook acha a magia no acervo já ordenado.
func spellOfBook(id string) book.Spell {
	for _, m := range book.Catalogs().Magias {
		if m.ID == id {
			return m
		}
	}
	return book.Spell{ID: id, Name: id}
}

// catalogSpellRowsOf são as magias que ainda dá para aprender, filtradas.
func catalogSpellRowsOf(dto sheet.CharacterDTO, busca, circulo, escola string) []catalogSpellRow {
	sabidas := map[string]bool{}
	for _, s := range dto.Spells {
		sabidas[s.CatalogSpellID] = true
	}
	linhas := []catalogSpellRow{}
	for _, m := range book.Catalogs().Magias {
		if sabidas[m.ID] || !passaNoFiltro(m, busca, circulo, escola) {
			continue
		}
		linhas = append(linhas, catalogSpellRow{
			ID: m.ID, Name: m.Name, Circle: circleName(m.Circle), School: schoolName(m.School),
			Effect: m.BaseEffect, Page: m.BookPage, Command: m.ID,
		})
	}
	return linhas
}

func passaNoFiltro(m book.Spell, busca, circulo, escola string) bool {
	if circulo != "" && strconv.Itoa(m.Circle) != circulo {
		return false
	}
	if escola != "" && m.School != escola {
		return false
	}
	if strings.TrimSpace(busca) == "" {
		return true
	}
	return strings.Contains(foldAccents(m.Name), foldAccents(busca))
}

// grantedSpellRowsOf são as magias que um PODER ensinou.
//
// Elas não moram no grimório: não se aprendem nem se esquecem, e some com o
// poder. Aparecem mesmo para quem não tem classe conjuradora — um bárbaro com
// Totem Espiritual (p42) tem de ver a magia dele.
func grantedSpellRowsOf(dto sheet.CharacterDTO) []grantedSpellRow {
	escolhas := map[string][]string{}
	if err := json.Unmarshal([]byte(dto.PowerChoices), &escolhas); err != nil {
		return nil
	}
	porNome := map[string]book.Spell{}
	for _, m := range book.Catalogs().Magias {
		porNome[m.Name] = m
	}
	linhas := []grantedSpellRow{}
	for _, poder := range powersThatTeachSpells() {
		picks, escolheu := escolhas[poder.ID]
		if !escolheu {
			continue
		}
		for _, pick := range picks {
			magia, existe := porNome[poder.Options[pick]]
			if !existe {
				continue
			}
			linhas = append(linhas, grantedSpellRow{
				Name: magia.Name, Circle: circleName(magia.Circle),
				Source: poder.Name, Effect: magia.BaseEffect, Page: magia.BookPage,
			})
		}
	}
	return linhas
}

// powerTeachingSpells é um poder que ENSINA magia, com o nome de cada opção.
type powerTeachingSpells struct {
	ID   string
	Name string
	// Options mapeia o id da escolha para o NOME da magia, que é o que a `note`
	// do catálogo guarda.
	Options map[string]string
}

func powersThatTeachSpells() []powerTeachingSpells {
	bruto, ok := catalog.Resource("class-powers")
	if !ok {
		return nil
	}
	var poderes []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Choice *struct {
			GrantsSpellAttribute string `json:"grantsSpellAttribute"`
			Options              []struct {
				ID   string `json:"id"`
				Note string `json:"note"`
			} `json:"options"`
		} `json:"choice"`
	}
	if err := json.Unmarshal(bruto, &poderes); err != nil {
		return nil
	}
	fora := []powerTeachingSpells{}
	for _, p := range poderes {
		if p.Choice == nil || p.Choice.GrantsSpellAttribute == "" {
			continue
		}
		opcoes := map[string]string{}
		for _, o := range p.Choice.Options {
			opcoes[o.ID] = o.Note
		}
		fora = append(fora, powerTeachingSpells{ID: p.ID, Name: p.Name, Options: opcoes})
	}
	return fora
}

// ── o que a TELA precisa escrever ────────────────────────────────────────────

// aprendidasEscrito é "3 aprendidas", com o singular certo.
func aprendidasEscrito(n int) string {
	if n == 1 {
		return "1 aprendida"
	}
	return strconv.Itoa(n) + " aprendidas"
}

// oSinalDoAprimoramento é o sinal que guarda a pilha de um aprimoramento.
//
// São SEIS, reaproveitados por todas as magias, e não um por magia: só um
// diálogo abre por vez, e seis é o máximo do catálogo (Conjurar Monstro). Um
// sinal por magia daria 198 × 6 declarações no `<body>` para guardar seis
// números.
func oSinalDoAprimoramento(indice int) string {
	return "$aug" + strconv.Itoa(indice)
}

// oGestoQueAbreOConjurar ZERA as pilhas antes de abrir.
//
// Quem TROCA de item limpa, e não quem gera — a regra do "remendo em nó
// compartilhado" do guia do Go. Sem isto, a pilha escolhida numa magia
// reapareceria na próxima que fosse aberta, e o custo mostrado seria o de outra
// conjuração.
func oGestoQueAbreOConjurar(magia learnedSpellRow) string {
	limpeza := ""
	for i := 0; i < 6; i++ {
		limpeza += oSinalDoAprimoramento(i) + " = 0; "
	}
	return limpeza + "$detalhe = 'conjura-" + magia.Command + "'"
}

// aPreviaDoCusto é a expressão que soma o custo na tela.
//
// Ela é PRÉVIA e não decisão: quem recusa é o servidor, com a regra inteira — o
// teto da p224, a redução de custo por item e o PM disponível. Escrever a regra
// aqui daria uma segunda conta do mesmo número, que é o defeito que a ALE-110
// registrou.
//
// Aprimoramento TRANCADO fica de fora da soma: ele não tem contador, então o
// sinal dele nunca sobe — mas somá-lo mostraria um custo que o servidor não
// cobraria.
func aPreviaDoCusto(magia learnedSpellRow) string {
	expr := strconv.Itoa(magia.BasePm)
	for _, a := range magia.Augments {
		if a.Locked {
			continue
		}
		expr += " + " + strconv.Itoa(a.PM) + " * " + oSinalDoAprimoramento(a.Index)
	}
	return "(" + expr + ") + ' PM'"
}

// circleName é "Truque" ou "3º", como a mesa fala.
func circleName(circle int) string {
	if circle == 0 {
		return "Truque"
	}
	return strconv.Itoa(circle) + "º"
}

var schoolNames = map[string]string{
	"abjuracao": "Abjuração", "adivinhacao": "Adivinhação", "convocacao": "Convocação",
	"encantamento": "Encantamento", "evocacao": "Evocação", "ilusao": "Ilusão",
	"necromancia": "Necromancia", "transmutacao": "Transmutação",
}

func schoolName(id string) string {
	if nome, tem := schoolNames[id]; tem {
		return nome
	}
	return id
}

func schoolOptions(ativa string) []filterOption {
	ids := make([]string, 0, len(schoolNames))
	for id := range schoolNames {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	opcoes := []filterOption{{Valor: "", Rotulo: "Todas as escolas", Ativo: ativa == ""}}
	for _, id := range ids {
		opcoes = append(opcoes, filterOption{Valor: id, Rotulo: schoolNames[id], Ativo: ativa == id})
	}
	return opcoes
}
