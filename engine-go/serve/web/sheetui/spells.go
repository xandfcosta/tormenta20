package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// A aba MAGIAS como dado.
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
// Não há pré-validação de PM no cliente: o `@post` responde em milissegundos e a
// recusa vem com a frase certa. O que a tela mostra é uma PRÉVIA do total,
// somada por expressão do Datastar sobre números que o servidor já mandou — e
// ela nunca decide nada.

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
	// Exclusive não aceita companhia na mesma conjuração (p171). Truque diz, além
	// disso, que a conjuração inteira custa zero. Ver GLOSSARY.
	Exclusive bool
	Truque    bool
	// Toggle, More e Less são os gestos MONTADOS, e não expressões escritas no
	// `.templ`, porque cada um precisa ver a VIZINHANÇA: ligar o exclusivo apaga
	// os outros e ligar qualquer outro apaga o exclusivo. É a mesma regra do
	// `thatOpensCastGesture` — quem troca limpa —, e é ela que impede a tela de
	// oferecer uma combinação que o servidor vai recusar.
	Toggle string
	More   string
	Less   string
}

// spellbookPanelOf monta a aba.
func (s Scene) spellbookPanelOf(dto sheet.CharacterDTO, busca, circulo, escola string) spellbookPanel {
	panel := spellbookPanel{
		IsCaster:       len(casterClassesOf(dto)) > 0,
		RequiresPrep:   sheet.RequiresPreparation(dto.Classes, dto.ClassChoices),
		CastableCircle: sheet.HighestCastableCircle(dto.Classes, 0),
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
		if sheet.IsCasterClass(c.ClassName) {
			nomes = append(nomes, c.ClassName)
		}
	}
	return nomes
}

// learnedSpellRowsOf é o grimório, por círculo e depois por nome.
func learnedSpellRowsOf(s Scene, dto sheet.CharacterDTO, castable int) []learnedSpellRow {
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
			BasePm: sheet.SpellBasePmCost[doLivro.Circle], Prepared: aprendida.Prepared,
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
func (s Scene) spellCdOf(dto sheet.CharacterDTO, magia catalog.Spell) string {
	sheet, _, ok := s.sheetForPanels(dto)
	if !ok {
		return "—"
	}
	melhor := 0
	for _, classe := range dto.Classes {
		prog, conjura := book.SpellProgressions()[classe.ClassName]
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

// augmentRowsOf traduz os aprimoramentos, trancando os fora de alcance e
// montando os gestos que se apagam entre si.
func augmentRowsOf(spellID string, magia catalog.Spell, castable int) []augmentRow {
	var exclusivos []int
	for i, a := range magia.Augments {
		if a.Exclusive {
			exclusivos = append(exclusivos, i)
		}
	}

	linhas := make([]augmentRow, 0, len(magia.Augments))
	for i, a := range magia.Augments {
		linha := augmentRow{
			Index: i, PM: a.PmCost, Stacks: a.Kind != "muda",
			Description: augmentDescription(spellID, i),
			Exclusive:   a.Exclusive, Truque: a.Truque,
		}
		if a.RequiresCircle != nil {
			linha.RequiredCircle = *a.RequiresCircle
			linha.Locked = *a.RequiresCircle > castable
		}
		// Quem o gesto APAGA: o exclusivo apaga todos os outros, e os outros
		// apagam os exclusivos. Sem isto a tela deixa montar truque + companhia,
		// mostra um custo e o servidor recusa na hora do clique — a recusa é a
		// fronteira, mas oferecer o que não vale é a tela mentindo.
		apaga := exclusivos
		if a.Exclusive {
			apaga = outrosIndices(len(magia.Augments), i)
		}
		linha.Toggle = clearSignals(apaga, i) + augmentSignal(i) + " = " + augmentSignal(i) + " ? 0 : 1"
		linha.More = clearSignals(apaga, i) + augmentSignal(i) + "++"
		// Diminuir não apaga nada: tirar uma pilha não monta combinação nova.
		linha.Less = augmentSignal(i) + " = Math.max(0, " + augmentSignal(i) + " - 1)"
		linhas = append(linhas, linha)
	}
	return linhas
}

// outrosIndices são todos os índices menos o próprio.
func outrosIndices(quantos, proprio int) []int {
	var fora []int
	for i := 0; i < quantos; i++ {
		if i != proprio {
			fora = append(fora, i)
		}
	}
	return fora
}

// clearSignals monta "$augmentA = 0; " para cada índice, pulando o próprio.
func clearSignals(indices []int, proprio int) string {
	expr := ""
	for _, i := range indices {
		if i == proprio {
			continue
		}
		expr += augmentSignal(i) + " = 0; "
	}
	return expr
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
		if sabidas[m.ID] || !passesFilter(m, busca, circulo, escola) {
			continue
		}
		linhas = append(linhas, catalogSpellRow{
			ID: m.ID, Name: m.Name, Circle: circleName(m.Circle), School: schoolName(m.School),
			Effect: m.BaseEffect, Page: m.BookPage, Command: m.ID,
		})
	}
	return linhas
}

func passesFilter(m book.Spell, busca, circulo, escola string) bool {
	if circulo != "" && strconv.Itoa(m.Circle) != circulo {
		return false
	}
	if escola != "" && m.School != escola {
		return false
	}
	if strings.TrimSpace(busca) == "" {
		return true
	}
	return strings.Contains(search.Fold(m.Name), search.Fold(busca))
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
	for _, poder := range book.PowersThatTeachSpells() {
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

// ── o que a TELA precisa escrever ────────────────────────────────────────────

// aprendidasEscrito é "3 aprendidas", com o singular certo.
func aprendidasEscrito(n int) string {
	if n == 1 {
		return "1 aprendida"
	}
	return strconv.Itoa(n) + " aprendidas"
}

// augmentSignal é o sinal que guarda a pilha de um aprimoramento.
//
// São SEIS, reaproveitados por todas as magias, e não um por magia: só um
// diálogo abre por vez, e seis é o máximo do catálogo (Conjurar Monstro). Um
// sinal por magia daria 198 × 6 declarações no `<body>` para guardar seis
// números.
func augmentSignal(indice int) string {
	return "$augment" + strconv.Itoa(indice)
}

// augmentChosenClasses é o `data-class` do botão "Trocar": a tinta do ligado e a
// do desligado, ambas presas ao MESMO sinal que o `aria-checked` usa.
//
// As duas listadas, e não só a do ligado: o `data-class` do Datastar ACRESCENTA
// e REMOVE conforme a expressão, então deixar a do desligado na classe estática
// faria as duas conviverem no instante ligado — borda dourada com texto cinza.
//
// O dourado é o mesmo do "Preparada" da magia, e isso é deliberado: as duas
// dizem "isto está ligado" na mesma tela.
func augmentChosenClasses(indice int) string {
	sinal := augmentSignal(indice)
	return "{'border-grimorio-gold/60 text-grimorio-gold': " + sinal + " > 0," +
		" 'border-grimorio-iron text-muted-foreground': " + sinal + " === 0}"
}

// thatOpensCastGesture ZERA as pilhas antes de abrir.
//
// Quem TROCA de item limpa, e não quem gera — a regra do "remendo em nó
// compartilhado" do guia do Go. Sem isto, a pilha escolhida numa magia
// reapareceria na próxima que fosse aberta, e o custo mostrado seria o de outra
// conjuração.
func thatOpensCastGesture(magia learnedSpellRow) string {
	limpeza := ""
	for i := 0; i < 6; i++ {
		limpeza += augmentSignal(i) + " = 0; "
	}
	return limpeza + "$detail = 'conjura-" + magia.Command + "'"
}

// costPreview é a expressão que soma o custo na tela.
//
// Ela é PRÉVIA e não decisão: quem recusa é o servidor, com a regra inteira — o
// teto da p224, a redução de custo por item e o PM disponível. Escrever a regra
// aqui daria uma segunda conta do mesmo número.
//
// Aprimoramento TRANCADO fica de fora da soma: ele não tem contador, então o
// sinal dele nunca sobe — mas somá-lo mostraria um custo que o servidor não
// cobraria.
func costPreview(magia learnedSpellRow) string {
	expr := strconv.Itoa(magia.BasePm)
	for _, a := range magia.Augments {
		if a.Locked {
			continue
		}
		expr += " + " + strconv.Itoa(a.PM) + " * " + augmentSignal(a.Index)
	}
	// O TRUQUE zera a conjuração INTEIRA, e não a parcela dele: "reduz seu custo
	// em PM para zero" (p171). Somado como um `+0 PM`, o custo base ficava de pé
	// e a tela anunciava 1 PM sobre uma conjuração que o servidor não cobra
	// (ALE-339). A prévia continua sem decidir nada — ela só deixou de discordar.
	for _, a := range magia.Augments {
		if a.Truque {
			return "(" + augmentSignal(a.Index) + " ? 0 : (" + expr + ")) + ' PM'"
		}
	}
	return "(" + expr + ") + ' PM'"
}

// augmentPriceWritten é a linha de baixo do aprimoramento: o que ele custa e o
// que ele impede.
//
// O TRUQUE não escreve "+0 PM", que é a mesma coisa que um aprimoramento de
// custo zero — e a Luz tem um desses (p197), então as duas linhas sairiam
// idênticas dizendo coisas diferentes. Ele escreve que ZERA a conjuração, que é
// o que a p171 diz.
//
// E a linha diz que ele não combina porque o gesto APAGA os outros: sem a frase,
// ligar o truque faz as escolhas anteriores sumirem sem explicação.
func augmentPriceWritten(a augmentRow) string {
	if a.Locked {
		return "+" + strconv.Itoa(a.PM) + " PM · exige o " + strconv.Itoa(a.RequiredCircle) + "º círculo"
	}
	if a.Truque {
		return "a conjuração custa 0 PM · não combina com os outros"
	}
	if a.Exclusive {
		return "+" + strconv.Itoa(a.PM) + " PM · não combina com os outros"
	}
	return "+" + strconv.Itoa(a.PM) + " PM"
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
