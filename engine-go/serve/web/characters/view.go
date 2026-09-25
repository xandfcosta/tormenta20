package characters

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// A cena de PERSONAGENS como dado. A forma é a de campanhas: o cursor é sinal,
// todos os palcos são desenhados, e a busca vai ao servidor.
//
// A DEFESA sai da `ComputeSheet`, a mesma da ficha, e todas saem juntas; os
// TEXTOS das habilidades de raça vêm do catálogo embutido; e a vaga de CRIAR é
// posição de cursor e não link — ver `scene.templ`.

type View struct {
	Search string
	// Herois já vem na ordem do trilho. A vaga de criar é a posição seguinte, e
	// não entra nesta lista: ela não é um herói e tratá-la como um faria toda
	// contagem da tela ficar um a mais.
	Heroes   []HeroCard
	CursorID int64
	// Total é o elenco INTEIRO, e a contagem da barra diz "3 de 10" com filtro
	// — dizer "3 de 3" esconderia que há sete escondidos pela busca.
	Total       int
	HasAny      bool
	FilteredAll bool
	// Neighbors espelha `Heroes` na ordem do trilho, e existe porque o vizinho é
	// COMPARTILHADO com a cena de campanhas: o `ui.NeighborAt` só indexa, e quem
	// sabe traduzir um cartão de herói em vizinho é quem tem o cartão. Montar aqui,
	// uma vez, também evita reconstruir dois vizinhos por palco desenhado.
	Neighbors []ui.Neighbor
}

type HeroCard struct {
	ID       int64
	Name     string
	Monogram string
	// Gradiente é o retrato derivado do nome, como a capa da campanha.
	Gradient string
	// Papel é "GUERREIRO 10" — classe primária em caixa alta, ou a origem
	// quando o personagem ainda não tem classe.
	Role string
	// Resumo é a linha de sabor montada dos campos estruturados, porque o app
	// não tem campo de biografia: raças • origem • devoto • tamanho • nível.
	Summary string
	Level   int64
	PV      string
	// PVInk é a TINTA do PV, e não a cor de preencher uma barra: aqui o vital é um
	// número e não uma faixa, então ele segue a escada de ESCREVER. Ela vive no
	// cartão e não no `templ` porque quem decide é a view — o componente só pinta o
	// que recebe.
	PVInk string
	// PVDown é a palavra de quem caiu — morrendo, estável, morto (p236) —, que
	// segue o número; vazia de pé.
	PVDown string
	PM     string
	// Defesa é TEXTO e não número porque ela pode ser desconhecida, e aí é um
	// travessão: nunca um zero, que é um valor de Defesa plausível e errado. O
	// travessão também mantém a fileira do mesmo tamanho — uma coluna que some faz
	// o palco dançar ao trocar de herói.
	Defense string
	// DefenseVs é a MESMA Defesa dita para quem vai resolver um ataque: um número
	// quando nada é direcional, e os dois quando algo é — hoje só o Caído (p394).
	//
	// DOIS campos e não um, porque os dois consumidores fazem perguntas diferentes.
	// A LISTA de heróis mostra o `Defense`: ela é catálogo fora da sessão, ninguém
	// está resolvendo ataque ali, e um par de números onde se comparam heróis é
	// ruído. O crachá da FICHA mostra o `DefenseVs`, porque ele existe justamente
	// para responder "acerta?", e essa resposta nunca é o total enquanto o alvo
	// está caído.
	//
	// A frase dos dois sai da MESMA função (`book.DefenseLabel`), então elas não
	// podem divergir sobre o que os números significam.
	DefenseVs string
	NoMana    bool
	Race      string
	Origin    string
	Classes   string
	Dossier   []book.RaceAbility
}

// Load monta a cena para um dono. Ela é EXPORTADA porque o consumidor hoje é
// uma bancada e não uma tela.
//
// O que os nove casos do `api/characters_scene_test.go` prendem é o caminho
// BANCO → PALCO: personagens gravados de verdade saem na lista, com a contagem,
// os vizinhos e a Defesa que a ficha mostra. Este pacote não pode provar isso —
// ele não tem banco, e importar o `db/testdb` junto com um `*api.Server` seria o
// ciclo que a divisão inteira existe para evitar.
//
// A fronteira fica assim: a cena diz COMO montar a si mesma, o hospedeiro prova
// que o que está no banco chega aqui.
func (s Scene) Load(ctx context.Context, ownerID int64, query string) (View, error) {
	cast, err := s.deps.CharacterList(ctx, ownerID)
	if err != nil {
		return View{}, err
	}

	v := View{Search: query, Total: len(cast), HasAny: len(cast) > 0}
	for _, c := range cast {
		if !search.Matches(searchFields(c), query) {
			continue
		}
		v.Heroes = append(v.Heroes, HeroCardOf(s.deps.Catalogs(), c))
	}
	v.FilteredAll = v.HasAny && len(v.Heroes) == 0
	if len(v.Heroes) > 0 {
		v.CursorID = v.Heroes[0].ID
	}
	for i, h := range v.Heroes {
		v.Neighbors = append(v.Neighbors, ui.Neighbor{
			ID: h.ID, Name: h.Name, Monogram: h.Monogram, Gradient: h.Gradient, Index: i,
		})
	}
	return v, nil
}

// searchFields são os quatro campos que a busca indexa: nome, classe primária,
// origem e raças. Buscar por RAÇA é o que faz "anao" achar o anão, e é o caso
// que a regra de acento existe para servir.
func searchFields(c sheet.CharacterDTO) []string {
	return []string{c.Name, primaryClass(c), c.Origin, racesInLine(c)}
}

// HeroCardOf é função LIVRE e não método da cena, e a razão é a regra da menor
// pergunta: de tudo que a `Deps` oferece, o cartão usa só o motor. Deixá-lo
// método obrigaria quem o chama de fora — a ficha, que reaproveita quatro campos
// dele — a montar uma `Scene` inteira para pedir um cartão.
func HeroCardOf(catalogs *engine.Catalogs, c sheet.CharacterDTO) HeroCard {
	card := HeroCard{
		ID:       c.ID,
		Name:     c.Name,
		Monogram: ui.Monogram(c.Name),
		Gradient: ui.NameGradient(c.Name),
		Role:     heroPlate(c),
		Summary:  stageLine(c),
		Level:    c.Level,
		PV:       vital(c.HpCurrent, c.HpMax),
		PVInk:    ui.HpInkTone(ui.VitalPercent(c.HpCurrent, c.HpMax)),
		PVDown: ui.DownedWord(c.HpCurrent, c.HpMax,
			slices.Contains(sheet.UnmarshalStrings(c.ActiveConditions), engine.ConditionBleeding)),
		PM:      vital(c.MpCurrent, c.MpMax),
		NoMana:  c.MpMax == 0,
		Race:    mainRace(c),
		Origin:  c.Origin,
		Classes: ClassesOf(c),
	}
	// A DEFESA vem da mesma `ComputeSheet` que a ficha usa, e do agregado JÁ
	// carregado — ver `sheet.Compute`. Sem motor (catálogo não primado) o cartão
	// simplesmente não mostra Defesa; a cena inteira não pode cair por causa de
	// um número.
	card.Defense = "—"
	card.DefenseVs = "—"
	if c.Ruleset != nil {
		if character, err := sheet.Compute(c); err == nil {
			card.Defense = strconv.Itoa(character.Defense.Total)
			card.DefenseVs = book.DefenseLabel(character.Defense)
		}
	}
	card.Dossier = book.RaceAbilities(card.Race, 8)
	return card
}

// stageLine é o resumo curto sob os vitais: "Devoto de X · origem · tamanho".
//
// Não confundir com o resumo do DOSSIÊ, que é mais longo e separa com ` • `.
//
// `god` é opcional e some quando ausente, em vez de virar "Devoto de ".
func stageLine(c sheet.CharacterDTO) string {
	parts := []string{}
	if c.God != nil && *c.God != "" {
		parts = append(parts, "Devoto de "+*c.God)
	}
	parts = append(parts, c.Origin, c.Size)
	// Fatia nova em vez do filtro no lugar (`partes[:0]`): aquele é correto e é
	// idioma conhecido, mas escreve no mesmo array que lê, e a lista aqui tem
	// cinco itens. Não vale um segundo de leitura a mais para quem passar.
	present := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			present = append(present, p)
		}
	}
	return strings.Join(present, " · ")
}

// heroPlate é o subtítulo da placa: "GUERREIRO 10 · ANÃO". A raça entra
// junto, e não é enfeite — num elenco de dez, classe sozinha repete.
func heroPlate(c sheet.CharacterDTO) string {
	plate := strings.ToUpper(classOrOrigin(c))
	if race := mainRace(c); race != "" {
		plate += " · " + strings.ToUpper(race)
	}
	return plate
}

// classOrOrigin: a classe primária com o nível, ou a origem para quem ainda
// não tem classe. É o "cargo" do herói na lista.
func classOrOrigin(c sheet.CharacterDTO) string {
	if len(c.Classes) == 0 {
		return c.Origin
	}
	return c.Classes[0].ClassName + " " + strconv.FormatInt(c.Classes[0].Level, 10)
}

func primaryClass(c sheet.CharacterDTO) string {
	if len(c.Classes) == 0 {
		return ""
	}
	return c.Classes[0].ClassName
}

func ClassesOf(c sheet.CharacterDTO) string {
	parts := make([]string, 0, len(c.Classes))
	for _, cl := range c.Classes {
		parts = append(parts, cl.ClassName+" "+strconv.FormatInt(cl.Level, 10))
	}
	return strings.Join(parts, " / ")
}

func racesInLine(c sheet.CharacterDTO) string {
	parts := make([]string, 0, len(c.Races))
	for _, r := range c.Races {
		parts = append(parts, r.Race)
	}
	return strings.Join(parts, ", ")
}

func mainRace(c sheet.CharacterDTO) string {
	if len(c.Races) == 0 {
		return ""
	}
	return c.Races[0].Race
}

func vital(current, max int64) string {
	return ui.HitPoints(current) + "/" + strconv.FormatInt(max, 10)
}
