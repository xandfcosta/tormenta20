package api

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"t20engine/catalog"
)

// O BESTIÁRIO como dado (ALE-257).
//
// O catálogo deixa de ir ao navegador, e aqui isso pesa mais que na cena de
// personagens: são 62 KB de criaturas que a SPA baixa para PODER filtrar. No
// servidor ele já está em memória por `go:embed`, e o que atravessa a rede é a
// lista filtrada.
//
// Some junto a LISTA VIRTUALIZADA, e a razão é medida e não gosto: o bestiário
// tem 80 criaturas. Virtualização é maquinaria para lista que não cabe, e a 80
// ela cobra o custo sem cobrir problema nenhum — inclusive o de precisar de e2e
// porque jsdom mede zero.

// verbete é uma entrada do bestiário do livro.
//
// O nome vem do GLOSSARIO.md, seção D: `verbete` é a entrada IMUTÁVEL do livro,
// `bloco de criatura` é o que o mestre escreve, e `criatura` é o guarda-chuva.
// A primeira versão disto se chamava `monstro`, que é uma quarta palavra para
// um conceito que já tem a sua.
//
// A primeira versão disto EMBUTIA o `CreatureBlock` do homebrew, com a
// justificativa de que "a ficha de uma criatura do livro e a de uma inventada
// são a mesma coisa". O dado desmente, e o `encoding/json` teria aceitado a
// mentira em SILÊNCIO — três perdas de uma vez:
//
//   - `bookPage` não existe no `CreatureBlock`, e é o que a linha mostra
//     ("p289"). Sumiria.
//   - o bloco do mestre chama os dois campos de `equipment` e `treasure`; o
//     livro grava `equipamento` e `tesouro`. Nomes diferentes não casam, e os
//     dois viriam VAZIOS — exatamente a perda que a ALE-151 consertou, com o
//     equipamento faltando nos 80 verbetes.
//   - e a pior: os atributos são `int` lá e ANULÁVEIS aqui. Nove criaturas têm
//     `inteligencia: null` e uma tem `forca: null`, porque o livro escreve
//     TRAVESSÃO — o Zumbi não tem Inteligência (p297). Num `int` isso vira 0, e
//     "+0" afirma que ele tem a média de um humano. É o defeito que a ALE-151
//     nomeia palavra por palavra, e o porte o reintroduziria.
//
// Por isso a estrutura é própria e os seis atributos são ponteiros. O
// `CreatureAttack` e o `CreatureSkill` são reusados porque esses SIM têm o
// mesmo formato nos dois lados — conferido campo a campo contra o JSON.
type verbete struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ND           float64 `json:"nd"`
	Tipo         string  `json:"tipo"`
	Size         string  `json:"size"`
	HP           int     `json:"hp"`
	Defesa       int     `json:"defesa"`
	Iniciativa   int     `json:"iniciativa"`
	Percepcao    int     `json:"percepcao"`
	Fortitude    int     `json:"fortitude"`
	Reflexos     int     `json:"reflexos"`
	Vontade      int     `json:"vontade"`
	Deslocamento string  `json:"deslocamento"`
	// Os seis que podem ser TRAVESSÃO. Ver o comentário do tipo.
	Forca        *int `json:"forca"`
	Destreza     *int `json:"destreza"`
	Constituicao *int `json:"constituicao"`
	Inteligencia *int `json:"inteligencia"`
	Sabedoria    *int `json:"sabedoria"`
	Carisma      *int `json:"carisma"`
	// PM só existe em conjurador: um zero diria "tem mana e está sem".
	PM               *int             `json:"pm,omitempty"`
	Attacks          []CreatureAttack `json:"attacks"`
	Skills           []CreatureSkill  `json:"skills"`
	SpecialAbilities []string         `json:"specialAbilities"`
	Equipamento      string           `json:"equipamento"`
	Tesouro          string           `json:"tesouro"`
	BookPage         int              `json:"bookPage"`
}

// comSinal escreve o modificador como o livro, e o TRAVESSÃO quando ele não
// existe. Ver o comentário de `verbete`: ausência não é zero (ALE-151).
func comSinal(n *int) string {
	if n == nil {
		return "—"
	}
	if *n >= 0 {
		return "+" + strconv.Itoa(*n)
	}
	return strconv.Itoa(*n)
}

var (
	bestiarioUmaVez sync.Once
	bestiario       []verbete
)

func criaturasDoLivro() []verbete {
	bestiarioUmaVez.Do(func() {
		bruto, ok := catalog.Resource("bestiary")
		if !ok {
			return
		}
		// Catálogo ausente é degradação NORMAL: a ferramenta abre vazia em vez
		// de derrubar a Mesa inteira. É a mesma decisão do `racasDaTela`.
		_ = json.Unmarshal(bruto, &bestiario)
	})
	return bestiario
}

// filtroDeCriaturas são os quatro critérios da tela.
type filtroDeCriaturas struct {
	Busca string
	// Tipos VAZIO significa TODOS, e não nenhum. É a convenção da tela: sem
	// crachá aceso, o filtro não filtra por tipo — tratar vazio como "nenhum"
	// mostraria bestiário vazio a quem não escolheu nada.
	Tipos []string
	NDMin float64
	NDMax float64
}

const (
	ndMinimo = 0.0
	ndMaximo = 20.0
)

// filtraCriaturas aplica os critérios e ORDENA por ND e depois por nome.
//
// A ordem é regra e não apresentação: o mestre procura por desafio, e uma lista
// alfabética o faria ler 80 linhas para achar as de ND 3.
//
// O desempate por nome usa COLLATION pt-BR, como o `sortInitiative`, e não
// `strings.Compare`. Em bytes, "Á" (0xC3 0x81) vem depois de "Z" — então
// "Águia" cairia no fim da faixa em vez de no começo. Medi contra as 80
// criaturas de hoje e as duas ordens coincidem, mas isso é acidente do dado:
// nenhum nome do livro começa com acento AINDA, e a linha que consertaria isso
// depois seria escrita por quem visse a lista errada sem saber por quê.
// O collator nasce por chamada porque não é seguro para concorrência.
func filtraCriaturas(todas []verbete, f filtroDeCriaturas) []verbete {
	fora := make([]verbete, 0, len(todas))
	for _, m := range todas {
		if !casaBusca([]string{m.Name}, f.Busca) {
			continue
		}
		if len(f.Tipos) > 0 && !slices.Contains(f.Tipos, m.Tipo) {
			continue
		}
		if m.ND < f.NDMin || m.ND > f.NDMax {
			continue
		}
		fora = append(fora, m)
	}
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(fora, func(a, b verbete) int {
		if c := cmp.Compare(a.ND, b.ND); c != 0 {
			return c
		}
		return col.CompareString(a.Name, b.Name)
	})
	return fora
}

// faixaDeND aperta o que veio da URL para dentro dos limites do livro.
//
// Um 999 digitado ou um texto que não é número esconderia TODAS as criaturas, e
// a tela leria como "bestiário vazio" em vez de "filtro absurdo". É o mesmo
// `clampToRange` que a SPA aplica na entrada dos dois campos — só que aqui a
// entrada é a URL, que qualquer um edita à mão.
//
// A faixa INVERTIDA (min 10, max 2) devolve lista vazia, e isso é PORTE e não
// descuido: é o que a SPA faz hoje, e a tela já diz "Nenhuma criatura casa com
// os filtros", que é resposta honesta. Consertar para "faixa inteira" faria o
// filtro MENTIR — pedir 10..2 e receber tudo é pior que receber nada.
func faixaDeND(minBruto, maxBruto string) (float64, float64) {
	return numeroOuPadrao(minBruto, ndMinimo), numeroOuPadrao(maxBruto, ndMaximo)
}

func numeroOuPadrao(bruto string, padrao float64) float64 {
	if bruto == "" {
		return padrao
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(bruto), 64)
	if err != nil || n < ndMinimo || n > ndMaximo {
		return padrao
	}
	return n
}

// ── como o livro escreve ─────────────────────────────────────────────────────

// ndEscrito: abaixo de 1 o livro usa FRAÇÃO, não decimal. "ND 0.25" não existe
// em lugar nenhum de Tormenta 20 — a mesa diz "ND 1/4".
func ndEscrito(nd float64) string {
	switch {
	case aproximadamente(nd, 0.25):
		return "1/4"
	case aproximadamente(nd, 0.5):
		return "1/2"
	case nd == float64(int(nd)):
		return strconv.Itoa(int(nd))
	default:
		return strconv.FormatFloat(nd, 'g', -1, 64)
	}
}

// A comparação é por PROXIMIDADE porque 0.25 e 0.5 vêm de JSON como float, e
// igualdade exata de ponto flutuante é a armadilha clássica desse caminho.
func aproximadamente(a, b float64) bool {
	d := a - b
	return d < 0.001 && d > -0.001
}

// Os rótulos dos tipos. O dado vem do catálogo sem acento e em caixa baixa; a
// tela mostra como se escreve.
var rotuloDoTipo = map[string]string{
	"humanoide":  "Humanoide",
	"animal":     "Animal",
	"monstro":    "Monstro",
	"morto-vivo": "Morto-vivo",
	"construto":  "Construto",
	"espirito":   "Espírito",
	"planar":     "Planar",
}

// A ordem do trilho de tipos é a do catálogo da SPA, e não alfabética: ela vai
// do mais comum na mesa para o mais raro.
var tiposDeCriatura = []string{"humanoide", "animal", "monstro", "morto-vivo", "construto", "espirito", "planar"}

func nomeDoTipo(tipo string) string {
	if r, ok := rotuloDoTipo[tipo]; ok {
		return r
	}
	return tipo
}

var rotuloDoTamanho = map[string]string{
	"minusculo": "Minúsculo",
	"pequeno":   "Pequeno",
	"medio":     "Médio",
	"grande":    "Grande",
	"enorme":    "Enorme",
	"colossal":  "Colossal",
}

func nomeDoTamanho(t string) string {
	if r, ok := rotuloDoTamanho[t]; ok {
		return r
	}
	return t
}

// xpDoND é o XP de tesouro derivado do ND.
//
// Portado do `xpForNd` da SPA, que cita Cap 8 p326 — a página é herdada dali e
// eu NÃO a reconferi contra o livro nesta fatia, o que fica dito porque o guia
// do `engine-go` pede citação conferida e uma repetida sem conferir parece uma
// conferida.
func xpDoND(nd float64) int {
	return int(math.Round(nd * 1000))
}

// ── a cena ───────────────────────────────────────────────────────────────────

// bestiarioView é o que a cena precisa para se desenhar inteira, numa resposta.
//
// A SPA baixa o catálogo (62 KB) para PODER filtrar; aqui a filtragem já
// aconteceu e o que atravessa é o resultado. `Total` viaja junto porque a linha
// "12 de 80" precisa dos dois números, e sem ele a tela não sabe se o filtro
// apertou muito ou se o bestiário é pequeno.
type bestiarioView struct {
	// Base é o prefixo das rotas que ESTA cena chama, e existe porque o mesmo
	// desenho serve dois lugares: a cena do mestre em `/piloto/mestre/bestiario`
	// e o painel da Mesa em `/piloto/mesa/{c}/{s}/bestiario`. O que muda entre
	// as duas é o ENDEREÇO, não a lista nem o bloco — e um segundo desenho seria
	// a mesma criatura mantida em dois lugares.
	//
	// Sem valor não há rota: o `bestiarioBase` recusa a string vazia em vez de
	// deixar o botão apontar para a página atual, que é o defeito silencioso
	// desta forma — o clique "funciona" e recarrega a cena.
	Base      string
	Verbetes  []verbete
	Total     int
	Escolhido *verbete
	Busca     string
	Tipos     []string
	NDMin     float64
	NDMax     float64
	// Abrir diz que ESTE pedido veio de um clique numa linha, e por isso a ficha
	// tem de nascer aberta. Vem da URL e não de um sinal: a MESMA rota serve a
	// busca e os filtros de tipo, e os dois mandam os sinais TODOS — inclusive o
	// `criatura` já escolhido. Um sinal não separaria "escolhi esta criatura" de
	// "digitei uma letra com uma criatura já escolhida", e a busca passaria a
	// abrir a ficha sozinha a cada tecla.
	Abrir bool
}

// escolhidoOuPrimeiro: a cena SEMPRE mostra um bloco quando há lista.
//
// A SPA faz `shown().find(...) ?? shown()[0]`, e a razão de portar isso é que o
// painel vazio ao lado de uma lista cheia parece defeito. Quando o filtro muda
// e a criatura escolhida sai da lista, cai na primeira em vez de esvaziar.
func escolhidoOuPrimeiro(lista []verbete, id string) *verbete {
	if len(lista) == 0 {
		return nil
	}
	for i := range lista {
		if lista[i].ID == id {
			return &lista[i]
		}
	}
	return &lista[0]
}

// carregaBestiario monta a cena a partir do que veio na URL ou nos sinais.
// carregaBestiarioDe exige a BASE como primeiro parâmetro, e isso é a lição de
// um guarda que acusou na hora: a primeira versão deixava o campo de fora e um
// teste de outra pasta montou a cena sem ele. Construtor que consegue produzir
// valor inválido é o próprio defeito — pedir aqui torna o esquecimento
// impossível em vez de detectável.
func carregaBestiarioDe(base, busca string, tipos []string, ndMin, ndMax float64, escolhido string) bestiarioView {
	todos := criaturasDoLivro()
	lista := filtraCriaturas(todos, filtroDeCriaturas{Busca: busca, Tipos: tipos, NDMin: ndMin, NDMax: ndMax})
	return bestiarioView{
		Base:      base,
		Verbetes:  lista,
		Total:     len(todos),
		Escolhido: escolhidoOuPrimeiro(lista, escolhido),
		Busca:     busca,
		Tipos:     tipos,
		NDMin:     ndMin,
		NDMax:     ndMax,
	}
}

// tipoAceso diz se o crachá está ligado, para a cena não precisar de `slices`.
func (v bestiarioView) tipoAceso(tipo string) bool {
	return slices.Contains(v.Tipos, tipo)
}

// ndNaCaixa escreve o número do campo sem o `.0` que o float traria: a caixa
// diz "3", não "3.0", e o passo do livro é de um quarto.
func ndNaCaixa(nd float64) string {
	return strconv.FormatFloat(nd, 'g', -1, 64)
}

// comSinalInt é o irmão do `comSinal` para os campos que NÃO podem ser
// travessão: iniciativa, percepção e as três resistências existem em todas as
// criaturas do livro.
func comSinalInt(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// sinaisDoBestiario: o estado da tela que o Datastar mantém no cliente.
//
// São só os quatro CRITÉRIOS mais a criatura aberta — nada de lista, nada de
// bloco. O que se vê chega desenhado; o que viaja de volta é o que o mestre
// escolheu.
func sinaisDoBestiario(v bestiarioView) string {
	tipos, _ := json.Marshal(v.Tipos)
	if v.Tipos == nil {
		tipos = []byte("[]")
	}
	escolhida := ""
	if v.Escolhido != nil {
		escolhida = v.Escolhido.ID
	}
	busca, _ := json.Marshal(v.Busca)
	criatura, _ := json.Marshal(escolhida)
	// `fichaAberta` sai DAQUI e não de um evento de sinal separado, e a razão é o
	// que a medição mostrou: este `data-signals` mora no `#bestiario`, que É o
	// elemento remendado, então ele REDECLARA os sinais a cada remendo. Um
	// evento de sinal mandado depois do conteúdo era desfeito por esta linha —
	// o fio levava `{"fichaAberta":true}` e o diálogo continuava `display:none`.
	//
	// É o mesmo perigo que o `piloto_layout.templ` já tinha escrito ao pôr os
	// sinais da página no `<body>`, que nunca é remendado. Aqui a saída não é
	// mover: é o servidor redeclarar com o valor CERTO, e aí o conteúdo e o
	// estado de aberto chegam no MESMO remendo — atômicos, sem janela em que um
	// esteja aplicado e o outro não.
	return fmt.Sprintf(`{busca: %s, ndMin: %s, ndMax: %s, tipos: %s, criatura: %s, fichaAberta: %t}`,
		busca, ndNaCaixa(v.NDMin), ndNaCaixa(v.NDMax), tipos, criatura, v.Abrir)
}

// bestiarioBase é o prefixo de rota da cena, e ele NÃO tem padrão.
//
// Uma base vazia produziria `@get(”)`, que o navegador resolve para a página
// ATUAL: o filtro pareceria funcionar (a página recarrega) e não filtraria nada.
// Um pânico no render é barulhento e acontece na primeira vez que alguém monta a
// cena sem dizer de onde ela fala.
func (v bestiarioView) bestiarioBase() string {
	if v.Base == "" {
		panic("bestiarioView sem Base: a cena não sabe para que rota falar")
	}
	return v.Base
}

// alternaOTipo liga ou desliga UM crachá de tipo no conjunto.
//
// Extraída porque tem dois chamadores desde que o painel da Mesa nasceu — a cena
// do mestre e ele —, e "alternar" é regra pequena o bastante para alguém
// reescrever sem notar que já existia, e grande o bastante para as duas cópias
// discordarem sobre o que fazer com um tipo repetido.
//
// Tipo que o catálogo não conhece é RECUSADO e não descartado: a URL é editável
// à mão, e um tipo inventado no conjunto filtraria tudo fora — a tela leria
// "Nenhuma criatura casa com os filtros" sem explicar por quê. É diferente do
// `tiposConhecidos`, que descarta de propósito porque lá o conjunto inteiro vem
// da URL e uma vírgula sobrando não deve esvaziar a tela.
func alternaOTipo(tipos []string, tipo string) ([]string, error) {
	if !slices.Contains(tiposDeCriatura, tipo) {
		return nil, fmt.Errorf("tipo de criatura desconhecido: %s", tipo)
	}
	if i := slices.Index(tipos, tipo); i >= 0 {
		return slices.Delete(slices.Clone(tipos), i, i+1), nil
	}
	return append(slices.Clone(tipos), tipo), nil
}

// verbetePorID acha a criatura do livro, ou nil.
//
// Nil e não erro: quem chama decide o que dizer. O painel da Mesa recusa a
// entrada com o id na frase, porque ali um id desconhecido só chega por adulteração.
func verbetePorID(id string) *verbete {
	if id == "" {
		return nil
	}
	todas := criaturasDoLivro()
	for i := range todas {
		if todas[i].ID == id {
			return &todas[i]
		}
	}
	return nil
}

// ── ABRIR a ficha na hora certa (ALE-264) ────────────────────────────────────
//
// O clique NÃO abre mais a ficha; quem abre é o SERVIDOR, depois de o conteúdo
// estar remendado. O defeito que isso conserta foi visto pelo dono e medido
// depois: clicar numa linha NÃO selecionada fazia a ficha abrir na hora com a
// criatura ANTERIOR e trocar um quadro depois. Amostrado no navegador — a 0ms a
// ficha dizia "Bandido", a 16ms dizia "Lobo".
//
// Clicar na linha JÁ selecionada não piscava, e foi essa diferença que o dono
// isolou sozinho: lá o conteúdo já estava certo, então não havia troca para ver.
// É a assinatura de conteúdo obsoleto exibido antes do novo, e não de
// renderização lenta — a lentidão apareceria nas duas linhas.
//
// Custa uma ida ao servidor antes de a ficha aparecer, e é o preço certo: 16ms
// medidos contra um quadro mostrando a criatura errada. Mostrar o errado rápido
// é pior que mostrar o certo um quadro depois.

// abrirAFicha marca o pedido que deve ABRIR a ficha ao terminar.
//
// A marca vai na URL e não num sinal porque a MESMA rota serve a busca e os
// filtros de tipo, e os dois mandam os sinais todos — inclusive o `criatura`.
// Um sinal não distinguiria "escolhi esta criatura" de "digitei uma letra na
// busca com uma criatura já escolhida", e a busca passaria a abrir a ficha
// sozinha a cada tecla.
func abrirAFicha(base string) string {
	if strings.Contains(base, "?") {
		return base + "&abrir=1"
	}
	return base + "?abrir=1"
}
