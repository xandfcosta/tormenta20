package master

import (
	"encoding/json"
	"fmt"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O IMPROVISO: as tabelas do Cap 6 e o esqueleto de masmorra na hora, numa tela
// só — a ideia de masmorra em d20 é uma tabela do Cap 6, e um lugar só é um
// lugar só para procurar.
//
// O dado rola no SERVIDOR (ver `engine/improvisation.go`), e o HISTÓRICO viaja
// nos sinais: o mestre rola várias vezes seguidas, e cada rolagem não pode
// virar uma entrada no histórico do navegador.

// ── o que a tela guarda ──────────────────────────────────────────────────────

// sorteio é uma rolagem já resolvida na linha dela. O NÚMERO viaja junto porque
// o mestre quer vê-lo: "saiu 4" é parte da resposta, e sem ele a tabela vira um
// oráculo que ninguém confere.
type roll struct {
	Roll int    `json:"r"`
	Text string `json:"t"`
	// Detail é a segunda linha, quando a tabela tem uma — o teste e a CD do
	// evento de perseguição, ou o castigo que acompanha a recompensa.
	Detail string `json:"d,omitempty"`
}

// O histórico guarda mais que "o último": o mestre que rola na mesma tabela
// duas vezes na cena quer COMPARAR, e guardar só o último joga a comparação
// fora no instante da segunda rolagem.
const historyDepth = 5

// improvView é a cena inteira. Os quatro históricos são separados porque as
// tabelas são independentes — rolar ruína não pode empurrar o evento de
// perseguição para fora da tela.
type improvView struct {
	Ruin   []roll
	Chase  []roll
	Reward []roll
	Ideas  []roll
	// A masmorra não é sorteio: é uma conta sobre o número de salas.
	Rooms   int
	Size    *book.DungeonSize
	Threats int
	// OverCeiling diz que o número de salas passou do maior tamanho do livro.
	// Não é erro: é o livro recomendando parar, e a tela diz isso em vez de
	// esconder o campo.
	OverCeiling bool
}

const (
	salasMinimo = 1
	salasMaximo = 200
	salasPadrao = 6
)

// loadImprov monta a cena a partir dos históricos que vieram nos sinais.
func loadImprov(v improvView) improvView {
	_, dungeon := book.ImprovTables()
	v.Rooms = clamp(v.Rooms, salasMinimo, salasMaximo, salasPadrao)

	for i := range dungeon.SizeTable {
		t := dungeon.SizeTable[i]
		if v.Rooms >= t.MinRooms && v.Rooms <= t.MaxRooms {
			v.Size = &t
			break
		}
	}
	v.OverCeiling = v.Size == nil
	if n, err := engine.PlannedThreats(v.Rooms, dungeon.RoomsPerThreat); err == nil {
		v.Threats = n
	}
	return v
}

// empilha põe o sorteio novo na frente e corta o excesso.
func push(history []roll, novo roll) []roll {
	outside := append([]roll{novo}, history...)
	if len(outside) > historyDepth {
		outside = outside[:historyDepth]
	}
	return outside
}

// ── as quatro rolagens ───────────────────────────────────────────────────────

// rollRuin: Tabela 6-4, d6, p272.
func rollRuin() (roll, error) {
	t, _ := book.ImprovTables()
	d, err := engine.RollDie(6)
	if err != nil {
		return roll{}, err
	}
	row, err := engine.RowForRoll(t.Ruin, d.Value, "ruina")
	if err != nil {
		return roll{}, err
	}
	return roll{Roll: d.Value, Text: row.Label}, nil
}

// rollChase: Tabela 6-5, d20, p274.
//
// O TIPO é a manchete, e não o exemplo. A primeira pergunta numa perseguição é
// "isto atrapalha ou ajuda?", e é o `kind` que responde. Trocar os dois perde o
// tipo inteiro na faixa "nenhum evento", onde o exemplo do livro é um travessão
// — a rolagem sai como "4 —".
//
// A CD e o exemplo entram no DETALHE: o mestre no meio da cena quer o número
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
	row, err := engine.RowForRoll(t.ChaseEvents, d.Value, "chaseEvents")
	if err != nil {
		return roll{}, err
	}
	s := roll{Roll: d.Value, Text: eventName(row.Kind)}
	var parts []string
	if row.Test != nil && row.CD != nil {
		parts = append(parts, fmt.Sprintf("%s (CD %d)", *row.Test, *row.CD))
	}
	// O travessão é como o livro escreve "não há exemplo", e repeti-lo na tela
	// só ocuparia a linha com um traço.
	if row.Example != "" && row.Example != "—" {
		parts = append(parts, row.Example)
	}
	s.Detail = strings.Join(parts, " · ")
	return s, nil
}

// Os tipos de evento de perseguição, como se lê. O dado vem em caixa baixa e
// sem acento, e mostrar o valor CRU ("obstaculo") obrigaria a olhar a tabela ao
// lado — mesma decisão do `book.ConditionName`.
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
	row, err := engine.RowForRoll(t.RewardPunishment, d.Value, "rewardCastigo")
	if err != nil {
		return roll{}, err
	}
	return roll{
		Roll:   d.Value,
		Text:   labelOrRaw(t.RewardLabels, row.Reward),
		Detail: "Castigo: " + labelOrRaw(t.PunishmentLabels, row.Punishment),
	}, nil
}

// rollIdea: Tabela 6-2, d20, p263.
func rollIdea() (roll, error) {
	_, m := book.ImprovTables()
	d, err := engine.RollDie(20)
	if err != nil {
		return roll{}, err
	}
	row, err := engine.RowForRoll(m.Ideas, d.Value, "ideias de masmorra")
	if err != nil {
		return roll{}, err
	}
	return roll{Roll: d.Value, Text: row.Label}, nil
}

func labelOrRaw(board map[string]string, key string) string {
	if r, ok := board[key]; ok {
		return r
	}
	return key
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
	return fmt.Sprintf(`{ruina: %s, perseguicao: %s, recompensa: %s, ideias: %s, rooms: %d}`,
		j(v.Ruin), j(v.Chase), j(v.Reward), j(v.Ideas), v.Rooms)
}
