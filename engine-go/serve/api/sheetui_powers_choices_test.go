package api

import (
	"context"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestAtTheFirstLevelThereIsNoPowerSlot(t *testing.T) {
	f, id := barbaro(t, 1)

	refusal := powerCommand(t, f, id, "escolhe/class.barbaro.golpe-poderoso", "")
	if !strings.Contains(refusal, "0 vagas") {
		t.Errorf("o nível 1 aceitou um poder: %q", refusal)
	}
}

// UM PODER DE OUTRA CLASSE não entra.
func TestAPowerFromAnotherClassDoesNotEnter(t *testing.T) {
	f, id := barbaro(t, 5)

	refusal := powerCommand(t, f, id, "escolhe/class.bardo.lendas-e-historias", "")
	if !strings.Contains(refusal, "Bardo") {
		t.Errorf("a recusa não diz de qual classe é o poder: %q", refusal)
	}
	// O PODER GERAL, ao contrário, entra em qualquer classe: "você sempre pode
	// substituir um poder de classe por um poder geral" (p33).
	if refused := powerCommand(t, f, id, "escolhe/ataque-poderoso", ""); refused != "" {
		t.Errorf("um poder geral foi recusado: %q", refused)
	}
}

// UM PODER AUTOMÁTICO não ocupa vaga — nem se alguém tentar escolhê-lo.
func TestAnAutomaticPowerTakesNoSlot(t *testing.T) {
	f, id := barbaro(t, 5)

	refusal := powerCommand(t, f, id, "escolhe/class.barbaro.furia", "")
	if !strings.Contains(refusal, "automático") {
		t.Errorf("a Fúria, que o nível concede, foi aceita como escolha: %q", refusal)
	}
}

// A ORIGEM DÁ DOIS BENEFÍCIOS, e o terceiro é recusado.
func TestTheOriginGivesTwoBenefits(t *testing.T) {
	f, id := barbaro(t, 3)

	for _, b := range []string{"pericia-furtividade", "pericia-percepcao"} {
		if refusal := powerCommand(t, f, id, "origem/origin-batedor-"+b, ""); refusal != "" {
			t.Fatalf("o benefício %q foi recusado: %q", b, refusal)
		}
	}
	refused := powerCommand(t, f, id, "origem/origin-batedor-pericia-sobrevivencia", "")
	if !strings.Contains(refused, "2 benefícios") {
		t.Errorf("o terceiro benefício foi aceito: %q", refused)
	}
}

// O PODER ÚNICO DA ORIGEM é escolhível — ele não está na lista de benefícios.
//
// O catálogo guarda o `poderUnico` num campo à parte, e a ficha o trata como um
// dos dois que a pessoa leva (p85): sem juntá-lo, ele não aparece em lugar
// nenhum e o servidor recusa quem tentar escolhê-lo.
func TestTheOnlyOriginPowerIsStillAChoice(t *testing.T) {
	f, id := barbaro(t, 3)

	if refusal := powerCommand(t, f, id, "origem/origin-batedor-unique", ""); refusal != "" {
		// O id do poder único do Batedor sai do catálogo; se ele mudar, o teste
		// falha dizendo o que procurar.
		t.Fatalf("o poder único da origem foi recusado: %q", refusal)
	}
	if !strings.Contains(powerScreen(t, f, id), "Estilo de Disparo") {
		t.Error("o poder único escolhido não aparece na lista")
	}
}

// UMA ORIGEM SEM BENEFÍCIOS não cobra o que não oferece.
//
// O Amnésico é a única assim: "em vez de dois benefícios, recebe uma perícia e
// um poder escolhidos pelo mestre" (p88). Cobrar dois dele daria uma pendência
// que a pessoa não tem como resolver, para sempre.
func TestAnOriginWithoutBenefitsDoesNotDemandTwo(t *testing.T) {
	f, id := barbaro(t, 1)
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET origin = ? WHERE id = ?", "amnesico", id,
	); err != nil {
		t.Fatalf("trocar a origem: %v", err)
	}

	if step := choiceStepTitled(t, powerScreen(t, f, id), "Origem · Amnésico"); step.Tag != "0 de 1" {
		t.Errorf("o passo da origem do Amnésico diz %q; ele oferece um só benefício, o poder único, e cobra \"0 de 1\"", step.Tag)
	}
}

// UM BENEFÍCIO DE OUTRA ORIGEM não entra.
func TestABenefitFromAnotherOriginDoesNotEnter(t *testing.T) {
	f, id := barbaro(t, 3)

	refusal := powerCommand(t, f, id, "origem/origin-acolito-pericia-cura", "")
	if !strings.Contains(refusal, "Batedor") {
		t.Errorf("a recusa não diz qual é a origem da ficha: %q", refusal)
	}
}

// O CAMINHO e o DEVOTO só aceitam o que a classe oferece.
func TestThePathAndTheDevotoOnlyAcceptWhatTheClassOffers(t *testing.T) {
	f, id := arcanista(t)

	if refusal := powerCommand(t, f, id, "classe/Arcanista/caminho/mago", ""); refusal != "" {
		t.Fatalf("um caminho do arcanista foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, "classe/Arcanista/caminho/bastiao", ""); refusal == "" {
		t.Error("o caminho do cavaleiro foi aceito num arcanista")
	}
	// E O ARCANISTA NÃO ESCOLHE DEVOTO: são três classes que escolhem (p57,
	// p61, p82), e ele não é nenhuma delas.
	if refusal := powerCommand(t, f, id, "classe/Arcanista/devoto/khalmyr", ""); refusal == "" {
		t.Error("um arcanista escolheu devoto")
	}
}

// AS PENDÊNCIAS dizem o que ainda falta — na ponta de cada passo do Escolher
// —, e a contagem sai no crachá.
func TestThePendenciesSayWhatIsMissing(t *testing.T) {
	f, id := barbaro(t, 3)

	screen := powerScreen(t, f, id)
	for _, want := range []choiceStepOnScreen{
		{Title: "Bárbaro 3", Tag: "0 de 2"}, // duas vagas, nenhuma usada
		{Title: "Origem · Batedor", Tag: "0 de 2"},
	} {
		if got := choiceStepTitled(t, screen, want.Title); got.Tag != want.Tag {
			t.Errorf("o passo %q diz %q, quer %q", want.Title, got.Tag, want.Tag)
		}
	}
	if !strings.Contains(screen, "escolhas pendentes") {
		t.Error("o crachá do Escolher não anuncia as escolhas pendentes")
	}

	// E ELAS DESCEM quando a escolha é feita.
	powerCommand(t, f, id, "escolhe/class.barbaro.golpe-poderoso", "")
	if got := choiceStepTitled(t, powerScreen(t, f, id), "Bárbaro 3"); got.Tag != "1 de 2" {
		t.Errorf("a pendência não desceu para uma vaga depois da escolha: o passo diz %q", got.Tag)
	}
}

// O DIÁLOGO oferece o que cabe em cada aba.
func TestTheDialogOffersWhatFitsInEachTab(t *testing.T) {
	f, id := arcanista(t)
	screen := powerScreen(t, f, id)

	if !strings.Contains(screen, `id="power-choices"`) {
		t.Fatal("a aba não tem o diálogo de escolher")
	}
	// O CAMINHO do arcanista, que ele escolhe desde o 1º nível — no passo
	// dele, e não em qualquer canto da tela.
	path := choiceStepBody(t, screen, "Arcanista · caminho")
	for _, want := range []string{"Bruxo", "Feiticeiro", "Mago"} {
		if !strings.Contains(path, want) {
			t.Errorf("o diálogo não oferece o caminho %q: %q", want, path)
		}
	}
	// E NÃO oferece o que é de outra classe.
	if strings.Contains(screen, "Égide Sagrada") {
		t.Error("o diálogo ofereceu um caminho de paladino a um arcanista")
	}
}
func chosen(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return row.Classpowers
}

// AS VAGAS DE PODER são uma por nível a partir do 2º (p33).
func TestPowerSlotsAreOnePerLevelFromTheSecondOn(t *testing.T) {
	f, id := barbaro(t, 3)

	// Duas vagas no 3º nível: a do 2º e a do 3º.
	if refusal := powerCommand(t, f, id, "escolhe/class.barbaro.golpe-poderoso", ""); refusal != "" {
		t.Fatalf("a primeira escolha foi recusada: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, "escolhe/class.barbaro.brado-assustador", ""); refusal != "" {
		t.Fatalf("a segunda escolha foi recusada: %q", refusal)
	}
	// A TERCEIRA não cabe.
	refused := powerCommand(t, f, id, "escolhe/class.barbaro.frenesi", "")
	if !strings.Contains(refused, "2 vagas") {
		t.Errorf("a recusa não diz quantas vagas existem: %q", refused)
	}
	if saved := chosen(t, f, id); strings.Contains(saved, "frenesi") {
		t.Errorf("a recusa gravou assim mesmo: %s", saved)
	}
}

// A PENDÊNCIA DO ATRIBUTO DE RAÇA é a promessa da forja ("dá para criar assim e
// terminar na ficha", ALE-169).
func TestTheRaceAttributePendencyAppearsAndCloses(t *testing.T) {
	f, id := barbaro(t, 1)
	seedRaca(t, f.s, id, "humano")

	// O humano só pede a distribuição, então o "falta 1" do passo da raça é
	// ela — e o "feito" depois é ela fechando.
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Raça · Humano"); step.Tag != "falta 1" {
		t.Fatalf("o humano sem distribuição não mostra a pendência: o passo diz %q", step.Tag)
	}

	body := `{"race_attributes":["strength","dexterity","constitution"]}`
	if refusal := powerCommand(t, f, id, "atributos", body); refusal != "" {
		t.Fatalf("a distribuição foi recusada: %q", refusal)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Raça · Humano"); step.Tag != "feito" {
		t.Errorf("a pendência ficou depois de a distribuição fechar: o passo diz %q", step.Tag)
	}
}

// A DISTRIBUIÇÃO REPETIDA é recusada — o livro pede atributos DIFERENTES.
func TestARepeatedDistributionIsRefused(t *testing.T) {
	f, id := barbaro(t, 1)
	seedRaca(t, f.s, id, "humano")

	body := `{"race_attributes":["strength","strength","strength"]}`
	if refusal := powerCommand(t, f, id, "atributos", body); !strings.Contains(refusal, "distintos") {
		t.Errorf("três vezes o mesmo atributo foi aceito: %q", refusal)
	}
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if strings.Contains(row.Raceattributechoices, "strength") {
		t.Errorf("a recusa gravou assim mesmo: %s", row.Raceattributechoices)
	}
}

// Não há caso de API JSON aqui de propósito: a regra é `sheet.WithChoicesValid`,
// e quem a prende é o comando da cena da ficha, que a chama pelo mesmo caminho.
// Uma regra, uma camada — e a camada que sobrou é a que a mesa usa.

// GRAVAR UMA ESCOLHA NÃO APAGA AS OUTRAS.
//
// As cinco escolhas moram em cinco COLUNAS da mesma linha, e cada gesto mexe só
// nas que mudaram. O caso precisa de DOIS gestos e de duas colunas: com um só,
// gravar o que veio e gravar as cinco dão o mesmo resultado, e a metade em que
// o defeito é visível fica de fora.
//
// A distribuição de atributo é o segundo gesto de propósito — ela é a única
// escolha que se grava SOZINHA, sem passar pelo funil que confere a ficha
// inteira, e por isso é a que teria mais chance de levar as vizinhas junto.
func TestWritingOneChoiceDoesNotEraseTheOthers(t *testing.T) {
	f, id := barbaro(t, 5)
	seedRaca(t, f.s, id, "humano")

	if refusal := powerCommand(t, f, id, "escolhe/ataque-poderoso", ""); refusal != "" {
		t.Fatalf("escolher o poder geral foi recusado: %q", refusal)
	}
	body := `{"race_attributes":["strength","dexterity","constitution"]}`
	if refusal := powerCommand(t, f, id, "atributos", body); refusal != "" {
		t.Fatalf("a distribuição foi recusada: %q", refusal)
	}

	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if !strings.Contains(row.Classpowers, "ataque-poderoso") {
		t.Errorf("o poder escolhido sumiu quando a distribuição foi gravada: %s", row.Classpowers)
	}
	if !strings.Contains(row.Raceattributechoices, "strength") {
		t.Errorf("a distribuição não foi gravada: %s", row.Raceattributechoices)
	}
}

// O PODER REPETÍVEL OCUPA UMA VAGA POR ESCOLHA (p38, ALE-491).
//
// "Você pode escolher este poder várias vezes" — e cada vez é um atributo. O
// gesto grava a escolha junto do poder: "Aumento: Força" e "Aumento:
// Sabedoria" são duas vagas, o poder aparece UMA vez em `classPowers` e as
// duas escolhas em `powerChoices`. A terceira escolha num bárbaro de nível 3,
// que tem duas vagas, é recusada.
func TestARepeatablePowerTakesOneSlotPerChoice(t *testing.T) {
	f, id := barbaro(t, 3)
	const poder = "class.barbaro.aumento-de-atributo"

	for _, opcao := range []string{"strength", "wisdom"} {
		if refusal := powerCommand(t, f, id, "escolha/"+poder+"/"+opcao, ""); refusal != "" {
			t.Fatalf("escolher Aumento de Atributo em %s foi recusado: %q", opcao, refusal)
		}
	}
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	if row.Classpowers != `["`+poder+`"]` {
		t.Errorf("o poder repetível devia entrar UMA vez em classPowers, e ficou %s", row.Classpowers)
	}
	if row.Powerchoices != `{"`+poder+`":["strength","wisdom"]}` {
		t.Errorf("as duas escolhas deviam estar em powerChoices, e ficou %s", row.Powerchoices)
	}

	// As duas escolhas ocupam as duas vagas: o terceiro poder não cabe.
	if refused := powerCommand(t, f, id, "escolhe/class.barbaro.golpe-poderoso", ""); !strings.Contains(refused, "2 vagas") {
		t.Errorf("com o Aumento tomado duas vezes, um terceiro poder coube em 2 vagas: %q", refused)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Bárbaro 3"); step.Tag != "feito" {
		t.Errorf("a pendência ainda pede poder com as duas vagas ocupadas pelo Aumento: o passo diz %q", step.Tag)
	}

	// UMA OPÇÃO QUE O LIVRO NÃO TEM é recusada, dizendo quais são.
	if refused := powerCommand(t, f, id, "escolha/"+poder+"/sorte", ""); !strings.Contains(refused, "Força") {
		t.Errorf("um atributo inventado não foi recusado com as opções: %q", refused)
	}

	// TIRAR uma escolha devolve a vaga; tirar a última tira o poder.
	powerCommand(t, f, id, "escolha/"+poder+"/strength/tira", "")
	powerCommand(t, f, id, "escolha/"+poder+"/wisdom/tira", "")
	if saved := chosen(t, f, id); strings.Contains(saved, "aumento") {
		t.Errorf("sem nenhuma escolha, o Aumento continuou em classPowers: %s", saved)
	}
}

// O DIÁLOGO OFERECE AS OPÇÕES DO PODER QUE PEDE ESCOLHA (ALE-491).
//
// O Aumento de Atributo não é um interruptor: cada atributo é um gesto, e o
// que já foi escolhido diz quantas vezes e oferece o de tirar. Sem isto a
// regra existia no motor e nenhuma tela a alcançava.
func TestTheDialogOffersTheOptionsOfAPowerThatAsksForAChoice(t *testing.T) {
	// Nível 5: duas vezes o mesmo atributo pede dois patamares (p38).
	f, id := barbaro(t, 5)
	const poder = "class.barbaro.aumento-de-atributo"

	screen := powerScreen(t, f, id)
	for _, opcao := range []string{"strength", "dexterity", "charisma"} {
		if !strings.Contains(screen, "/poderes/escolha/"+poder+"/"+opcao) {
			t.Errorf("o diálogo não oferece o Aumento de Atributo em %s", opcao)
		}
	}
	if strings.Contains(screen, "/poderes/escolha/"+poder+"/strength/tira") {
		t.Fatal("o controle já estava errado: sem escolha, o diálogo oferece tirar Força")
	}

	powerCommand(t, f, id, "escolha/"+poder+"/strength", "")
	powerCommand(t, f, id, "escolha/"+poder+"/strength", "")
	screen = powerScreen(t, f, id)
	if !strings.Contains(screen, "/poderes/escolha/"+poder+"/strength/tira") {
		t.Error("com Força escolhida, o diálogo não oferece tirá-la")
	}
	if !strings.Contains(screen, "Força ×2") {
		t.Error("Força escolhida duas vezes não diz ×2 no diálogo")
	}
}

// A ARMA DA ESPECIALIZAÇÃO É ESCOLHIDA PELO DIÁLOGO, e só arma do catálogo
// entra (p65, ALE-491).
func TestTheWeaponSpecializationIsChosenFromTheWeaponCatalog(t *testing.T) {
	f, id := fighterFixture(t)
	const poder = "class.guerreiro.especializacao-em-arma"

	screen := powerScreen(t, f, id)
	if !strings.Contains(screen, "/poderes/escolha/"+poder+"/espada-longa") {
		t.Error("o diálogo não oferece a espada longa à Especialização em Arma")
	}
	// "Escolha uma arma" é o TIPO de arma. A Avalanche é um item mágico
	// específico (p337), feito sobre um martelo de guerra, e não uma arma
	// que se escolhe.
	if strings.Contains(screen, "/poderes/escolha/"+poder+"/avalanche") {
		t.Error("o diálogo oferece a Avalanche, uma arma específica do capítulo 8, à Especialização")
	}
	if refusal := powerCommand(t, f, id, "escolha/"+poder+"/espada-longa", ""); refusal != "" {
		t.Fatalf("escolher a espada longa foi recusado: %q", refusal)
	}
	// Um item que não é arma não entra.
	if refused := powerCommand(t, f, id, "escolha/"+poder+"/mochila", ""); !strings.Contains(refused, "Adaga") {
		t.Errorf("um item que não é arma não foi recusado com as opções: %q", refused)
	}
}

// ── a leitura do Escolher ───────────────────────────────────────────────────

// choiceStepOnScreen é um passo da lista do Escolher como a pessoa o lê: o
// título ("Origem · Batedor"), a nota, e a ponta — "falta 1", "0 de 2" ou
// "feito". A pendência mora na PONTA: é ela que diz o que ainda falta.
type choiceStepOnScreen struct {
	Title, Meta, Tag string
}

// choiceStepsOnScreen são os passos da lista do Escolher, na ordem da tela.
func choiceStepsOnScreen(t *testing.T, page string) []choiceStepOnScreen {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("ler a página: %v", err)
	}
	list := findNode(doc, "class", "ds-steps-list")
	if list == nil {
		t.Fatal("a página não tem a lista dos passos do Escolher — o resto mediria o nada")
	}
	steps := []choiceStepOnScreen{}
	for _, row := range nodesWithClassToken(list, "ds-step-row") {
		step := choiceStepOnScreen{}
		if title := nodesWithClassToken(row, "ds-row-title"); len(title) > 0 {
			step.Title = spacedText(title[0])
		}
		if meta := nodesWithClassToken(row, "ds-row-meta"); len(meta) > 0 {
			step.Meta = spacedText(meta[0])
		}
		if tag := nodesWithClassToken(row, "ds-step-tag"); len(tag) > 0 {
			step.Tag = spacedText(tag[0])
		}
		steps = append(steps, step)
	}
	return steps
}

// choiceStepTitled é o passo com aquele título; a falha diz quais passos a
// lista tem, que é o que se precisa para saber se o título mudou ou se o
// passo sumiu.
func choiceStepTitled(t *testing.T, page, title string) choiceStepOnScreen {
	t.Helper()
	steps := choiceStepsOnScreen(t, page)
	for _, step := range steps {
		if step.Title == title {
			return step
		}
	}
	t.Fatalf("a lista do Escolher não tem o passo %q; tem %+v", title, steps)
	return choiceStepOnScreen{}
}

// choiceFieldOnScreen é um bloco de escolha dentro de um passo — a
// Deformidade, o Especialista: o "1 de 2" do cabeçalho e as opções, cada uma
// pelo `aria-label` e dizendo se está marcada. Marcar e desmarcar é o MESMO
// toque, então "marcada" é o que oferece tirar.
type choiceFieldOnScreen struct {
	Written string
	Options map[string]bool
	HTML    string
}

// choiceFieldIn é o bloco `label` do passo `step`, e se ele existe.
func choiceFieldIn(t *testing.T, page, step, label string) (choiceFieldOnScreen, bool) {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("ler a página: %v", err)
	}
	var section *html.Node
	for _, candidate := range nodesWithClassToken(doc, "ds-step") {
		if attr(candidate, "aria-label") == step {
			section = candidate
		}
	}
	if section == nil {
		t.Fatalf("o Escolher não tem a seção do passo %q", step)
	}
	for _, field := range nodesWithClassToken(section, "ds-field") {
		labels := nodesWithClassToken(field, "ds-field-label")
		if len(labels) == 0 || spacedText(labels[0]) != label {
			continue
		}
		read := choiceFieldOnScreen{Options: map[string]bool{}, HTML: renderNode(t, field)}
		if heads := nodesWithClassToken(field, "ds-pick-head"); len(heads) > 0 {
			if meta := nodesWithClassToken(heads[0], "ds-row-meta"); len(meta) > 0 {
				read.Written = spacedText(meta[0])
			}
		}
		for _, option := range nodesWithClassToken(field, "ds-option") {
			read.Options[attr(option, "aria-label")] = attr(option, "aria-pressed") == "true"
		}
		return read, true
	}
	return choiceFieldOnScreen{}, false
}

// choiceStepBody é o texto do passo aberto, para o que ele oferece.
func choiceStepBody(t *testing.T, page, step string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("ler a página: %v", err)
	}
	for _, candidate := range nodesWithClassToken(doc, "ds-step") {
		if attr(candidate, "aria-label") == step {
			return spacedText(candidate)
		}
	}
	t.Fatalf("o Escolher não tem a seção do passo %q", step)
	return ""
}

// nodesWithClassToken são os elementos sob `n` que têm `token` entre as
// classes — por TOKEN, porque a linha é `ds-row ds-step-row` e casar a string
// inteira deixaria de ver a variante.
func nodesWithClassToken(n *html.Node, token string) []*html.Node {
	found := []*html.Node{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && slices.Contains(strings.Fields(attr(n, "class")), token) {
			found = append(found, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return found
}

func firstElementChild(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return c
		}
	}
	return nil
}

// spacedText é o texto do nó com o espaço achatado, como a tela o mostra.
func spacedText(n *html.Node) string {
	return strings.Join(strings.Fields(textOf(n)), " ")
}
