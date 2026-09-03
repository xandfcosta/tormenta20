package master

import (
	"encoding/json"
	"fmt"
	"strings"

	"t20engine/book"
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

// ── o que a tela guarda ──────────────────────────────────────────────────────

// sorteio é uma rolagem já resolvida na linha dela. O NÚMERO viaja junto porque
// o mestre quer vê-lo: "saiu 4" é parte da resposta, e sem ele a tabela vira um
// oráculo que ninguém confere.
type roll struct {
	Rolagem int    `json:"r"`
	Texto   string `json:"t"`
	// Detalhe é a segunda linha, quando a tabela tem uma — o teste e a CD do
	// evento de perseguição, ou o castigo que acompanha a recompensa.
	Detalhe string `json:"d,omitempty"`
}

// A profundidade do histórico é a da SPA, e a razão dela sobrevive ao porte: o
// mestre que rola na mesma tabela duas vezes na cena quer comparar, e guardar só
// "o último" joga fora a comparação no instante da segunda rolagem.
const historyDepth = 5

// improvView é a cena inteira. Os quatro históricos são separados porque as
// tabelas são independentes — rolar ruína não pode empurrar o evento de
// perseguição para fora da tela.
type improvView struct {
	Ruina       []roll
	Perseguicao []roll
	Recompensa  []roll
	Ideias      []roll
	// A masmorra não é sorteio: é uma conta sobre o número de salas.
	Salas   int
	Tamanho *book.DungeonSize
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

// loadImprov monta a cena a partir dos históricos que vieram nos sinais.
func loadImprov(v improvView) improvView {
	_, masmorra := book.ImprovTables()
	v.Salas = clamp(v.Salas, salasMinimo, salasMaximo, salasPadrao)

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
func push(historico []roll, novo roll) []roll {
	fora := append([]roll{novo}, historico...)
	if len(fora) > historyDepth {
		fora = fora[:historyDepth]
	}
	return fora
}

// ── as quatro rolagens ───────────────────────────────────────────────────────

// rollRuin: Tabela 6-4, d6, p272.
func rollRuin() (roll, error) {
	t, _ := book.ImprovTables()
	d, err := engine.RollDie(6)
	if err != nil {
		return roll{}, err
	}
	linha, err := engine.RowForRoll(t.Ruina, d.Valor, "ruina")
	if err != nil {
		return roll{}, err
	}
	return roll{Rolagem: d.Valor, Texto: linha.Label}, nil
}

// rollChase: Tabela 6-5, d20, p274.
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
func rollChase() (roll, error) {
	t, _ := book.ImprovTables()
	d, err := engine.RollDie(20)
	if err != nil {
		return roll{}, err
	}
	linha, err := engine.RowForRoll(t.ChaseEvents, d.Valor, "chaseEvents")
	if err != nil {
		return roll{}, err
	}
	s := roll{Rolagem: d.Valor, Texto: eventName(linha.Kind)}
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
// tabela ao lado — mesma divergência deliberada do `book.ConditionName`.
var eventLabel = map[string]string{
	"nenhum":    "Nenhum evento",
	"obstaculo": "Obstáculo",
	"atalho":    "Atalho",
}

func eventName(k string) string { return labelOrRaw(eventLabel, k) }

// rollReward devolve as DUAS pontas: a tabela dá recompensa e castigo na
// mesma linha, e separá-las esconderia que elas são um par.
func rollReward() (roll, error) {
	t, _ := book.ImprovTables()
	d, err := engine.RollDie(6)
	if err != nil {
		return roll{}, err
	}
	linha, err := engine.RowForRoll(t.RewardCastigo, d.Valor, "rewardCastigo")
	if err != nil {
		return roll{}, err
	}
	return roll{
		Rolagem: d.Valor,
		Texto:   labelOrRaw(t.RewardLabels, linha.Reward),
		Detalhe: "Castigo: " + labelOrRaw(t.CastigoLabels, linha.Castigo),
	}, nil
}

// rollIdea: Tabela 6-2, d20, p263.
func rollIdea() (roll, error) {
	_, m := book.ImprovTables()
	d, err := engine.RollDie(20)
	if err != nil {
		return roll{}, err
	}
	linha, err := engine.RowForRoll(m.Ideas, d.Valor, "ideias de masmorra")
	if err != nil {
		return roll{}, err
	}
	return roll{Rolagem: d.Valor, Texto: linha.Label}, nil
}

func labelOrRaw(mapa map[string]string, chave string) string {
	if r, ok := mapa[chave]; ok {
		return r
	}
	return chave
}

// ── a escrita da masmorra ────────────────────────────────────────────────────

var pacingLabel = map[string]string{
	"parte-de-sessao":  "Parte de uma sessão",
	"sessao-inteira":   "Sessão inteira",
	"aventura-inteira": "Aventura inteira",
}

func pacingName(p string) string { return labelOrRaw(pacingLabel, p) }

// improvSignals: os quatro históricos e o número de salas. Nada mais — o
// que se vê chega desenhado.
func improvSignals(v improvView) string {
	j := func(s []roll) string {
		if s == nil {
			return "[]"
		}
		b, _ := json.Marshal(s)
		return string(b)
	}
	return fmt.Sprintf(`{ruina: %s, perseguicao: %s, recompensa: %s, ideias: %s, salas: %d}`,
		j(v.Ruina), j(v.Perseguicao), j(v.Recompensa), j(v.Ideias), v.Salas)
}
