package api

import (
	"context"
	"strconv"
	"strings"
)

// A cena de PERSONAGENS como dado (ALE-239) — a segunda cena de seleção, e a
// primeira em que o SERVIDOR faz trabalho que a SPA pedia por requisição.
//
// A forma é a de campanhas (ALE-234): o cursor é sinal, todos os palcos são
// desenhados, e a busca vai ao servidor. O que muda são três coisas, e as três
// são ganho de servidor:
//
//  1. A DEFESA sai da `ComputeSheetV2`, a mesma da ficha. Na SPA é uma consulta
//     por personagem, disparada quando o cursor pousa; aqui todas saem juntas.
//  2. Os TEXTOS das habilidades de raça vêm do catálogo embutido, não baixado.
//  3. A vaga de CRIAR é posição de cursor, não link — ver `piloto_personagens.templ`.

type personagensView struct {
	Busca string
	// Herois já vem na ordem do trilho. A vaga de criar é a posição seguinte, e
	// não entra nesta lista: ela não é um herói e tratá-la como um faria toda
	// contagem da tela ficar um a mais.
	Herois   []heroiCartao
	CursorID int64
	// Total é o elenco INTEIRO, e a contagem da barra diz "3 de 10" com filtro
	// — dizer "3 de 3" esconderia que há sete escondidos pela busca.
	Total       int
	TemAlgum    bool
	FiltrouTudo bool
}

type heroiCartao struct {
	ID       int64
	Nome     string
	Iniciais string
	// Gradiente é o retrato derivado do nome, como a capa da campanha.
	Gradiente string
	// Papel é "GUERREIRO 10" — classe primária em caixa alta, ou a origem
	// quando o personagem ainda não tem classe.
	Papel string
	// Resumo é a linha de sabor montada dos campos estruturados, porque o app
	// não tem campo de biografia: raças • origem • devoto • tamanho • nível.
	Resumo string
	Nivel  int64
	PV     string
	PM     string
	// Defesa é TEXTO e não número porque ela pode ser desconhecida, e aí é um
	// travessão. A SPA faz igual, e o motivo dela vale aqui: nunca um zero, que
	// é um valor de Defesa plausível e errado. Travessão também mantém a fileira
	// do mesmo tamanho — uma coluna que some faz o palco dançar ao trocar de
	// herói, que é o defeito da ALE-99.
	Defesa  string
	SemMana bool
	Raca    string
	Origem  string
	Classes string
	Dossie  []habilidadeDeRaca
}

func (s *Server) carregaPersonagens(ctx context.Context, eu AuthUser, busca string) (personagensView, error) {
	elenco, err := s.characterList(ctx, eu.ID)
	if err != nil {
		return personagensView{}, err
	}

	v := personagensView{Busca: busca, Total: len(elenco), TemAlgum: len(elenco) > 0}
	for _, c := range elenco {
		if !casaBusca(camposDeBusca(c), busca) {
			continue
		}
		v.Herois = append(v.Herois, s.cartaoDoHeroi(c))
	}
	v.FiltrouTudo = v.TemAlgum && len(v.Herois) == 0
	if len(v.Herois) > 0 {
		v.CursorID = v.Herois[0].ID
	}
	return v, nil
}

// camposDeBusca são os quatro que a SPA indexa: nome, classe primária, origem e
// raças. Portados como estão — buscar por raça é o que faz "anao" achar o anão,
// e esse é o caso que a regra de acento existe para servir.
func camposDeBusca(c CharacterDTO) []string {
	return []string{c.Name, classePrimaria(c), c.Origin, racasEmLinha(c)}
}

func (s *Server) cartaoDoHeroi(c CharacterDTO) heroiCartao {
	cartao := heroiCartao{
		ID:        c.ID,
		Nome:      c.Name,
		Iniciais:  iniciais(c.Name),
		Gradiente: gradienteDaCampanha(c.Name),
		Papel:     placaDoHeroi(c),
		Resumo:    linhaDoPalco(c),
		Nivel:     c.Level,
		PV:        vital(c.HpCurrent, c.HpMax),
		PM:        vital(c.MpCurrent, c.MpMax),
		SemMana:   c.MpMax == 0,
		Raca:      racaPrincipal(c),
		Origem:    c.Origin,
		Classes:   classesDoHeroi(c),
	}
	// A DEFESA vem da mesma `ComputeSheetV2` que a ficha usa, e do agregado JÁ
	// carregado — ver `sheetFromDTO`. Sem motor (catálogo não primado) o cartão
	// simplesmente não mostra Defesa; a cena inteira não pode cair por causa de
	// um número.
	cartao.Defesa = "—"
	if s.catalogs != nil {
		if ficha, err := s.sheetFromDTO(c); err == nil {
			cartao.Defesa = strconv.Itoa(ficha.Defense.Total)
		}
	}
	cartao.Dossie = habilidadesDaRaca(cartao.Raca, 8)
	return cartao
}

// linhaDoPalco é o resumo curto sob os vitais: "Devoto de X · origem · tamanho".
//
// Portei o `characterFlavor` primeiro e estava ERRADO — comparando com a tela
// da SPA lado a lado, aquele é o resumo do DOSSIÊ, e o palco usa outro, mais
// curto e com separador diferente (` · ` e não ` • `). São duas funções
// parecidas no mesmo arquivo de origem, e escolher pela semelhança do nome é
// como se troca uma pela outra.
//
// `god` é opcional e some quando ausente, em vez de virar "Devoto de ".
func linhaDoPalco(c CharacterDTO) string {
	partes := []string{}
	if c.God != nil && *c.God != "" {
		partes = append(partes, "Devoto de "+*c.God)
	}
	partes = append(partes, c.Origin, c.Size)
	// Fatia nova em vez do filtro no lugar (`partes[:0]`): aquele é correto e é
	// idioma conhecido, mas escreve no mesmo array que lê, e a lista aqui tem
	// cinco itens. Não vale um segundo de leitura a mais para quem passar.
	presentes := make([]string, 0, len(partes))
	for _, p := range partes {
		if strings.TrimSpace(p) != "" {
			presentes = append(presentes, p)
		}
	}
	return strings.Join(presentes, " · ")
}

// placaDoHeroi é o subtítulo da placa: "GUERREIRO 10 · ANÃO". A raça entra
// junto, e não é enfeite — num elenco de dez, classe sozinha repete.
func placaDoHeroi(c CharacterDTO) string {
	placa := strings.ToUpper(classeOuOrigem(c))
	if raca := racaPrincipal(c); raca != "" {
		placa += " · " + strings.ToUpper(raca)
	}
	return placa
}

// classeOuOrigem: a classe primária com o nível, ou a origem para quem ainda
// não tem classe. É o "cargo" do herói na lista.
func classeOuOrigem(c CharacterDTO) string {
	if len(c.Classes) == 0 {
		return c.Origin
	}
	return c.Classes[0].ClassName + " " + strconv.FormatInt(c.Classes[0].Level, 10)
}

func classePrimaria(c CharacterDTO) string {
	if len(c.Classes) == 0 {
		return ""
	}
	return c.Classes[0].ClassName
}

func classesDoHeroi(c CharacterDTO) string {
	partes := make([]string, 0, len(c.Classes))
	for _, cl := range c.Classes {
		partes = append(partes, cl.ClassName+" "+strconv.FormatInt(cl.Level, 10))
	}
	return strings.Join(partes, " / ")
}

func racasEmLinha(c CharacterDTO) string {
	partes := make([]string, 0, len(c.Races))
	for _, r := range c.Races {
		partes = append(partes, r.Race)
	}
	return strings.Join(partes, ", ")
}

func racaPrincipal(c CharacterDTO) string {
	if len(c.Races) == 0 {
		return ""
	}
	return c.Races[0].Race
}

func vital(atual, max int64) string {
	return strconv.FormatInt(atual, 10) + "/" + strconv.FormatInt(max, 10)
}

// lavagemDoHeroi é o facho de luz no matiz do próprio herói — o que faz o palco
// parecer iluminado em vez de listado. Decorativo, e por isso o elemento que o
// usa é `aria-hidden`.
func lavagemDoHeroi(h heroiCartao) string {
	m := matizDoNome(h.Nome)
	return "radial-gradient(ellipse 60% 50% at 50% 42%, oklch(0.55 0.15 " +
		strconv.Itoa(m) + " / 0.14), transparent 70%)"
}
