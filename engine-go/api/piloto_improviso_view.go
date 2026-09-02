package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"t20engine/catalog"
	"t20engine/engine"
)

// O IMPROVISO (ALE-261), quarta e última ferramenta da Mesa do Mestre — as
// tabelas do Cap 6 e o esqueleto de masmorra na hora.
//
// A SPA tinha isto em DUAS telas, e o comentário de lá conta por que viraram
// uma: a ideia de masmorra em d20 é uma tabela do Cap 6 que aparecia nas duas, e
// um lugar só é um lugar só para procurar.
//
// O dado rola no SERVIDOR (ver `engine/improvisation.go`), e o HISTÓRICO viaja nos
// sinais — mesma forma do rascunho do encontro (ALE-259), pela mesma razão: o
// mestre rola várias vezes seguidas e cada rolagem não pode virar uma entrada no
// histórico do navegador.

// ── as tabelas, lidas uma vez ────────────────────────────────────────────────

type linhaDeRuina struct {
	RollMin int    `json:"rollMin"`
	RollMax int    `json:"rollMax"`
	Outcome string `json:"outcome"`
	Label   string `json:"label"`
}

func (l linhaDeRuina) Cobre(r int) bool { return r >= l.RollMin && r <= l.RollMax }

type linhaDePerseguicao struct {
	RollMin int     `json:"rollMin"`
	RollMax int     `json:"rollMax"`
	Kind    string  `json:"kind"`
	Test    *string `json:"test"`
	CD      *int    `json:"cd"`
	Example string  `json:"example"`
}

func (l linhaDePerseguicao) Cobre(r int) bool { return r >= l.RollMin && r <= l.RollMax }

// linhaDeRecompensa é a única das três que casa por valor EXATO e não por
// faixa: a tabela de recompensa/castigo tem uma linha por face do d6.
type linhaDeRecompensa struct {
	Roll    int    `json:"roll"`
	Reward  string `json:"reward"`
	Castigo string `json:"castigo"`
}

func (l linhaDeRecompensa) Cobre(r int) bool { return l.Roll == r }

type ideiaDeMasmorra struct {
	Roll  int    `json:"roll"`
	Label string `json:"label"`
}

func (i ideiaDeMasmorra) Cobre(r int) bool { return i.Roll == r }

type tamanhoDeMasmorra struct {
	Size                   string `json:"size"`
	Label                  string `json:"label"`
	MinRooms               int    `json:"minRooms"`
	MaxRooms               int    `json:"maxRooms"`
	Pacing                 string `json:"pacing"`
	MaxSecondaryObjectives int    `json:"maxSecondaryObjectives"`
}

type tabelasDoMestre struct {
	Ruina         []linhaDeRuina       `json:"ruina"`
	ChaseEvents   []linhaDePerseguicao `json:"chaseEvents"`
	RewardCastigo []linhaDeRecompensa  `json:"rewardCastigo"`
	RewardLabels  map[string]string    `json:"rewardLabels"`
	CastigoLabels map[string]string    `json:"castigoLabels"`
}

type desenhoDeMasmorra struct {
	Sizes          []string            `json:"sizes"`
	SizeTable      []tamanhoDeMasmorra `json:"sizeTable"`
	RoomsPerThreat int                 `json:"roomsPerThreat"`
	Ideas          []ideiaDeMasmorra   `json:"ideas"`
}

var (
	improvisoUmaVez sync.Once
	tabelasCap6     tabelasDoMestre
	masmorras       desenhoDeMasmorra
)

func tabelasDoImproviso() (tabelasDoMestre, desenhoDeMasmorra) {
	improvisoUmaVez.Do(func() {
		if bruto, ok := catalog.Resource("gm-tables"); ok {
			_ = json.Unmarshal(bruto, &tabelasCap6)
		}
		if bruto, ok := catalog.Resource("dungeon-design"); ok {
			_ = json.Unmarshal(bruto, &masmorras)
		}
	})
	return tabelasCap6, masmorras
}

// ── o que a tela guarda ──────────────────────────────────────────────────────

// sorteio é uma rolagem já resolvida na linha dela. O NÚMERO viaja junto porque
// o mestre quer vê-lo: "saiu 4" é parte da resposta, e sem ele a tabela vira um
// oráculo que ninguém confere.
type sorteio struct {
	Rolagem int    `json:"r"`
	Texto   string `json:"t"`
	// Detalhe é a segunda linha, quando a tabela tem uma — o teste e a CD do
	// evento de perseguição, ou o castigo que acompanha a recompensa.
	Detalhe string `json:"d,omitempty"`
}

// A profundidade do histórico é a da SPA, e a razão dela sobrevive ao porte: o
// mestre que rola na mesma tabela duas vezes na cena quer comparar, e guardar só
// "o último" joga fora a comparação no instante da segunda rolagem.
const fundoDoHistorico = 5

// improvisoView é a cena inteira. Os quatro históricos são separados porque as
// tabelas são independentes — rolar ruína não pode empurrar o evento de
// perseguição para fora da tela.
type improvisoView struct {
	Ruina       []sorteio
	Perseguicao []sorteio
	Recompensa  []sorteio
	Ideias      []sorteio
	// A masmorra não é sorteio: é uma conta sobre o número de salas.
	Salas   int
	Tamanho *tamanhoDeMasmorra
	Ameacas int
	// AcimaDoTeto diz que o número de salas passou do maior tamanho do livro.
	// Não é erro: é o livro recomendando parar, e a tela diz isso em vez de
	// esconder o campo.
	AcimaDoTeto bool
}

const (
	salasMinimo = 1
	salasMaximo = 200
	salasPadrao = 6
)

// carregaImproviso monta a cena a partir dos históricos que vieram nos sinais.
func carregaImproviso(v improvisoView) improvisoView {
	_, masmorra := tabelasDoImproviso()
	v.Salas = aperta(v.Salas, salasMinimo, salasMaximo, salasPadrao)

	for i := range masmorra.SizeTable {
		t := masmorra.SizeTable[i]
		if v.Salas >= t.MinRooms && v.Salas <= t.MaxRooms {
			v.Tamanho = &t
			break
		}
	}
	v.AcimaDoTeto = v.Tamanho == nil
	if n, err := engine.PlannedThreats(v.Salas, masmorra.RoomsPerThreat); err == nil {
		v.Ameacas = n
	}
	return v
}

// empilha põe o sorteio novo na frente e corta o excesso.
func empilha(historico []sorteio, novo sorteio) []sorteio {
	fora := append([]sorteio{novo}, historico...)
	if len(fora) > fundoDoHistorico {
		fora = fora[:fundoDoHistorico]
	}
	return fora
}

// ── as quatro rolagens ───────────────────────────────────────────────────────

// rolaRuina: Tabela 6-4, d6, p272.
func rolaRuina() (sorteio, error) {
	t, _ := tabelasDoImproviso()
	d, err := engine.RollDie(6)
	if err != nil {
		return sorteio{}, err
	}
	linha, err := engine.RowForRoll(t.Ruina, d.Valor, "ruina")
	if err != nil {
		return sorteio{}, err
	}
	return sorteio{Rolagem: d.Valor, Texto: linha.Label}, nil
}

// rolaPerseguicao: Tabela 6-5, d20, p274.
//
// O TIPO é a manchete, e não o exemplo. A primeira pergunta numa perseguição é
// "isto atrapalha ou ajuda?", e é o `kind` que responde — a SPA acerta nisso e a
// minha primeira versão trocou os dois, pondo o exemplo em cima e perdendo o
// tipo inteiro. Só apareceu ao olhar a captura: a rolagem 4 saía como "4 —",
// porque na faixa "nenhum evento" o exemplo do livro é um travessão.
//
// O que a SPA descarta e aqui fica: a CD e o exemplo. Ela mostra
// "obstaculo · teste: Força" e para aí; o mestre no meio da cena quer o número
// contra o qual rolar e uma frase para narrar, e os dois estão no dado.
//
// O `nil` do teste é significativo: na faixa 1-6 não há o que rolar, e um
// "CD 0" diria que existe um teste trivial em vez de nenhum.
func rolaPerseguicao() (sorteio, error) {
	t, _ := tabelasDoImproviso()
	d, err := engine.RollDie(20)
	if err != nil {
		return sorteio{}, err
	}
	linha, err := engine.RowForRoll(t.ChaseEvents, d.Valor, "chaseEvents")
	if err != nil {
		return sorteio{}, err
	}
	s := sorteio{Rolagem: d.Valor, Texto: nomeDoEvento(linha.Kind)}
	var partes []string
	if linha.Test != nil && linha.CD != nil {
		partes = append(partes, fmt.Sprintf("%s (CD %d)", *linha.Test, *linha.CD))
	}
	// O travessão é como o livro escreve "não há exemplo", e repeti-lo na tela
	// só ocuparia a linha com um traço.
	if linha.Example != "" && linha.Example != "—" {
		partes = append(partes, linha.Example)
	}
	s.Detalhe = strings.Join(partes, " · ")
	return s, nil
}

// Os tipos de evento de perseguição, como se lê. O dado vem em caixa baixa e
// sem acento; a SPA mostra o valor CRU ("obstaculo"), e resolver é olhar a
// tabela ao lado — mesma divergência deliberada do `nomeDaCondicao`.
var rotuloDoEvento = map[string]string{
	"nenhum":    "Nenhum evento",
	"obstaculo": "Obstáculo",
	"atalho":    "Atalho",
}

func nomeDoEvento(k string) string { return rotuloOuCru(rotuloDoEvento, k) }

// rolaRecompensa devolve as DUAS pontas: a tabela dá recompensa e castigo na
// mesma linha, e separá-las esconderia que elas são um par.
func rolaRecompensa() (sorteio, error) {
	t, _ := tabelasDoImproviso()
	d, err := engine.RollDie(6)
	if err != nil {
		return sorteio{}, err
	}
	linha, err := engine.RowForRoll(t.RewardCastigo, d.Valor, "rewardCastigo")
	if err != nil {
		return sorteio{}, err
	}
	return sorteio{
		Rolagem: d.Valor,
		Texto:   rotuloOuCru(t.RewardLabels, linha.Reward),
		Detalhe: "Castigo: " + rotuloOuCru(t.CastigoLabels, linha.Castigo),
	}, nil
}

// rolaIdeia: Tabela 6-2, d20, p263.
func rolaIdeia() (sorteio, error) {
	_, m := tabelasDoImproviso()
	d, err := engine.RollDie(20)
	if err != nil {
		return sorteio{}, err
	}
	linha, err := engine.RowForRoll(m.Ideas, d.Valor, "ideias de masmorra")
	if err != nil {
		return sorteio{}, err
	}
	return sorteio{Rolagem: d.Valor, Texto: linha.Label}, nil
}

func rotuloOuCru(mapa map[string]string, chave string) string {
	if r, ok := mapa[chave]; ok {
		return r
	}
	return chave
}

// ── a escrita da masmorra ────────────────────────────────────────────────────

var rotuloDoRitmo = map[string]string{
	"parte-de-sessao":  "Parte de uma sessão",
	"sessao-inteira":   "Sessão inteira",
	"aventura-inteira": "Aventura inteira",
}

func nomeDoRitmo(p string) string { return rotuloOuCru(rotuloDoRitmo, p) }

// sinaisDoImproviso: os quatro históricos e o número de salas. Nada mais — o
// que se vê chega desenhado.
func sinaisDoImproviso(v improvisoView) string {
	j := func(s []sorteio) string {
		if s == nil {
			return "[]"
		}
		b, _ := json.Marshal(s)
		return string(b)
	}
	return fmt.Sprintf(`{ruina: %s, perseguicao: %s, recompensa: %s, ideias: %s, salas: %d}`,
		j(v.Ruina), j(v.Perseguicao), j(v.Recompensa), j(v.Ideias), v.Salas)
}
