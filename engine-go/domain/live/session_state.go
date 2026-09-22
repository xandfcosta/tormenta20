package live

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// InitiativeMaxEntries é o teto de combatentes numa fila só: guarda contra um
// Add em disparada, e acima de ~20 a tela já pagina mal.
const InitiativeMaxEntries = 50

// InitiativeEntry é uma linha de combatente na fila de iniciativa da sessão.
//
// Os campos numéricos opcionais são PONTEIRO com `omitempty` para o JSON
// distinguir "ausente" de "zero" — um combatente sem iniciativa rolada não é um
// combatente com iniciativa 0, e achatar os dois apagaria a diferença no fio.
type InitiativeEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Initiative  int    `json:"initiative"`
	Type        string `json:"type"` // "character" | "npc"
	CharacterID *int64 `json:"characterId,omitempty"`
	// MonsterID liga a linha ao verbete do bestiário, para o mestre abrir o bloco
	// do monstro sem procurar no catálogo. Ausente em NPC digitado à mão — e é
	// por isso que é ponteiro: "sem bloco" é diferente de "bloco vazio".
	MonsterID *string `json:"monsterId,omitempty"`
	// CreatureID liga a linha ao bloco de criatura que o MESTRE escreveu.
	// Diferente do `MonsterID`, que aponta para o verbete imutável do livro:
	// este é editável e pertence à campanha, e é o que responde "o ogro que eu
	// modifiquei". Uma linha tem um ou outro, nunca os dois.
	CreatureID *int64 `json:"creatureId,omitempty"`
	// Conditions são as condições do livro ativas nesta linha (p394-395).
	// Moram na LINHA e não no bloco de criatura pelo mesmo motivo que os PV
	// atuais: condição é estado de combate, e o vilão recorrente não volta na
	// semana seguinte ainda caído. Para PC a fonte continua sendo a ficha —
	// aqui é o caminho do NPC, que ficha não tem.
	Conditions []string `json:"conditions,omitempty"`
	// HpHidden esconde os PV desta linha dos JOGADORES: saber que o ogro está com
	// 12 de 130 muda a decisão de quem está na mesa, e essa é a informação do
	// mestre. Ponteiro porque a maioria das linhas não decide nada a respeito.
	HpHidden *bool `json:"hpHidden,omitempty"`
	// MpHidden é o irmão do `HpHidden` para o mana, e os dois são TRI-ESTADO:
	// nulo é "o mestre não decidiu", e aí vale o padrão do pool — o PV do grupo
	// aparece, o resto não. Explícito manda nos dois sentidos, que é como o
	// mestre REVELA o ogro de propósito.
	MpHidden  *bool  `json:"mpHidden,omitempty"`
	HpCurrent *int64 `json:"hpCurrent,omitempty"`
	HpMax     *int64 `json:"hpMax,omitempty"`
	MpCurrent *int64 `json:"mpCurrent,omitempty"`
	MpMax     *int64 `json:"mpMax,omitempty"`
}

// SessionRuntimeState é o rastreador vivo da sessão: a fila ordenada por
// iniciativa DESC, a rodada, e o índice de quem está na vez (-1 antes do combate
// e depois do Reset). É a forma persistida em `Session.runtimeState` e a que sai
// no evento `session-state`.
type SessionRuntimeState struct {
	// Seq é a ORDEM da mutação que produziu este instantâneo. Não exportado de
	// propósito: é metadado de transporte e não pode entrar no fio. O
	// `encoding/json` ignora campo não exportado, e o `CloneState` o carrega de
	// graça porque copia a struct inteira.
	Seq        uint64
	Initiative []InitiativeEntry `json:"initiative"`
	Round      int               `json:"round"`
	TurnIndex  int               `json:"turnIndex"`
	// TurnsTaken conta os turnos desde o começo do combate, e é CONTADO em vez
	// de derivado: rodada × tamanho da lista mente assim que alguém entra ou
	// morre no meio do combate, que é o normal numa mesa.
	TurnsTaken int `json:"turnsTaken"`
	// Scene é a cena EM CURSO — nil fora de cena. Ver `scene.go`: ela substituiu
	// um booleano que dizia "o combate está ligado", e o livro chama de cena um
	// pedaço distinto da história, do qual o combate é um tipo (p252).
	Scene *Scene `json:"scene,omitempty"`
	// ScenesSoFar é quantas cenas esta sessão já teve. Ele mora no estado e não
	// é derivado da cena em curso porque a sessão continua tendo tido três
	// cenas depois que a terceira acaba.
	ScenesSoFar int `json:"scenesSoFar,omitempty"`
}

// EmptyRuntimeState é um rastreador novo. Cada chamada devolve uma fatia nova,
// para duas sessões nunca compartilharem a fila.
func EmptyRuntimeState() *SessionRuntimeState {
	return &SessionRuntimeState{Initiative: []InitiativeEntry{}, Round: 0, TurnIndex: -1}
}

// EntryPatch é a atualização parcial de uma linha: só os campos não nulos são
// aplicados. Separado do `InitiativeEntry` para "deixa como está" (nulo) ser
// diferente de "põe zero".
type EntryPatch struct {
	Label       *string `json:"label"`
	Initiative  *int    `json:"initiative"`
	Type        *string `json:"type"`
	CharacterID *int64  `json:"characterId"`
	HpCurrent   *int64  `json:"hpCurrent"`
	HpMax       *int64  `json:"hpMax"`
	MpCurrent   *int64  `json:"mpCurrent"`
	MpMax       *int64  `json:"mpMax"`
	HpHidden    *bool   `json:"hpHidden"`
	MpHidden    *bool   `json:"mpHidden"`
	// Conditions substitui a lista inteira, como o endpoint da ficha faz: um
	// patch por condição exigiria dizer "some" e "tire", e a tela sempre sabe o
	// conjunto final.
	Conditions *[]string `json:"conditions"`
	// CreatureID liga a linha ao bloco de criatura do mestre depois de a linha
	// já existir — é o "detalhar este NPC", que cria o bloco e o prende ao
	// combatente que já estava na mesa.
	CreatureID *int64 `json:"creatureId"`
}

// sortInitiative mantém a lista DESC por iniciativa, com o empate desfeito pelo
// rótulo em colação pt-BR (sensível a acento: "Ávila" < "Bravo"). O colador é
// criado A CADA CHAMADA porque `Collator` não é seguro para uso concorrente.
func sortInitiative(st *SessionRuntimeState) {
	c := collate.New(language.BrazilianPortuguese)
	sort.SliceStable(st.Initiative, func(i, j int) bool {
		a, b := st.Initiative[i], st.Initiative[j]
		if a.Initiative != b.Initiative {
			return a.Initiative > b.Initiative
		}
		return c.CompareString(a.Label, b.Label) < 0
	})
}

func FindEntryIndex(st *SessionRuntimeState, entryID string) int {
	for i := range st.Initiative {
		if st.Initiative[i].ID == entryID {
			return i
		}
	}
	return -1
}

// turnEntryID devolve o id de quem está na vez, ou "" quando não há ninguém —
// para a reordenação devolver a vez ao MESMO combatente, e não ao mesmo índice.
func turnEntryID(st *SessionRuntimeState) string {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return ""
	}
	return st.Initiative[st.TurnIndex].ID
}

// restoreTurn devolve o turnIndex à linha que estava na vez antes da reordenação.
func restoreTurn(st *SessionRuntimeState, id string) {
	if id == "" {
		return
	}
	if idx := FindEntryIndex(st, id); idx >= 0 {
		st.TurnIndex = idx
	}
}

// AddEntry acrescenta um combatente (o id sai do `newID`), reordena e preserva
// quem está na vez. Falha quando a fila está cheia.
func AddEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	if len(st.Initiative) >= InitiativeMaxEntries {
		return fmt.Errorf("Initiative tracker is full (max %d entries)", InitiativeMaxEntries)
	}
	onTurn := turnEntryID(st)
	input.Label = numberedLabel(st, input.Label)
	input.ID = newID()
	st.Initiative = append(st.Initiative, input)
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// numberedLabel numera o REPETIDO na fila: o segundo Ogro entra como "Ogro 2".
//
// Quem numera é o SERVIDOR, pela mesma razão do tabuleiro: achar o próximo
// número livre é decisão sobre o estado, e duas telas adivinhando produziriam
// dois "Ogro 2". A regra é a MESMA função que o mapa usa
// (`NextInstanceLabelAmong`), para as duas superfícies não numerarem diferente.
//
// Sem isto, quatro ogros davam quatro linhas chamadas "Ogro" — e a fila nomeia
// os botões dela pelo rótulo ("Remover Ogro"), então nem o mestre nem um leitor
// de tela conseguiam dizer qual é qual.
//
// O rótulo VAZIO passa reto: quem valida a obrigatoriedade é quem materializa a
// linha, e inventar "1" aqui esconderia o erro dele.
func numberedLabel(st *SessionRuntimeState, label string) string {
	if strings.TrimSpace(label) == "" {
		return label
	}
	used := make([]string, 0, len(st.Initiative))
	livre := true
	for _, entry := range st.Initiative {
		used = append(used, entry.Label)
		if entry.Label == label {
			livre = false
		}
	}
	// O PRIMEIRO de cada espécie fica sem número: "Ogro", e só o segundo vira
	// "Ogro 2". Numerar desde o começo encheria a fila de "1" que não ajudam a
	// distinguir nada quando há um só.
	if livre {
		return label
	}
	return NextInstanceLabelAmong(used, label)
}

// UpsertCharacterEntry acrescenta a linha do personagem ou — se ele já está na
// fila — atualiza só a iniciativa dele, que é a rerrolagem, mantendo os PV/PM do
// meio do combate.
func UpsertCharacterEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	idx := -1
	if input.CharacterID != nil {
		for i := range st.Initiative {
			if st.Initiative[i].CharacterID != nil && *st.Initiative[i].CharacterID == *input.CharacterID {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return AddEntry(st, input, newID)
	}
	onTurn := turnEntryID(st)
	st.Initiative[idx].Initiative = input.Initiative
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// UpdateEntry aplica um patch parcial numa linha. Só reordena (preservando quem
// está na vez) quando a iniciativa muda.
func UpdateEntry(st *SessionRuntimeState, entryID string, patch EntryPatch) error {
	idx := FindEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if patch.Label != nil {
		e.Label = *patch.Label
	}
	if patch.Type != nil {
		e.Type = *patch.Type
	}
	if patch.CharacterID != nil {
		e.CharacterID = patch.CharacterID
	}
	if patch.HpCurrent != nil {
		e.HpCurrent = patch.HpCurrent
	}
	if patch.HpMax != nil {
		e.HpMax = patch.HpMax
	}
	if patch.MpCurrent != nil {
		e.MpCurrent = patch.MpCurrent
	}
	if patch.MpMax != nil {
		e.MpMax = patch.MpMax
	}
	if patch.MpHidden != nil {
		e.MpHidden = patch.MpHidden
	}
	if patch.HpHidden != nil {
		e.HpHidden = patch.HpHidden
	}
	if patch.CreatureID != nil {
		e.CreatureID = patch.CreatureID
	}
	if patch.Conditions != nil {
		e.Conditions = *patch.Conditions
	}
	if patch.Initiative != nil {
		e.Initiative = *patch.Initiative
		onTurn := turnEntryID(st)
		sortInitiative(st)
		restoreTurn(st, onTurn)
	}
	return nil
}

// RemoveEntry tira uma linha e conserta o turnIndex: desloca para a esquerda
// quando sai alguém antes da vez, e vira a rodada quando quem estava na vez era
// o último.
func RemoveEntry(st *SessionRuntimeState, entryID string) error {
	idx := FindEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	st.Initiative = append(st.Initiative[:idx], st.Initiative[idx+1:]...)
	if len(st.Initiative) == 0 {
		st.TurnIndex = -1
		return nil
	}
	if idx < st.TurnIndex {
		st.TurnIndex--
	} else if idx == st.TurnIndex && st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
	return nil
}

// AdvanceTurn passa a vez, dando a volta no índice 0 e somando uma rodada. Do
// pré-combate (turnIndex -1) ele põe o primeiro na vez sem somar rodada.
//
// Sem CENA não avança. A guarda não é defensiva: ela é o que dá ao estado uma
// direção única — turno só existe dentro de cena —, e é dela que o
// `parseRuntimeBlob` tira o direito de deduzir a cena de um turno em curso.
// CloneState copia o estado para o broadcast: as entradas vão por VALOR e a
// fatia é recriada. Os vitais `*int64` são compartilhados mas nunca mutados no
// lugar (patch e delta sempre atribuem ponteiro novo), então o instantâneo é
// seguro de serializar fora da trava.
//
// Por VALOR e não campo a campo: listar campos é uma lista que envelhece, e o
// campo novo que ficasse de fora continuaria compilando e passaria a zerar o
// valor em silêncio — na cópia que vai para o fio e para o banco.
func CloneState(s *SessionRuntimeState) *SessionRuntimeState {
	out := *s
	out.Initiative = make([]InitiativeEntry, len(s.Initiative))
	copy(out.Initiative, s.Initiative)
	return &out
}

// AS MUTAÇÕES DA FILA SÃO PÚBLICAS, e isso é a fronteira e não conveniência
// (ALE-344).
//
// Elas eram privadas porque o único chamador morava no mesmo pacote: o STORE.
// Com ele em `app/session`, onde a orquestração pertence, o que sobra aqui é a
// regra — e a regra de um pacote de domínio existe para ser chamada de fora.
// O `domain/board` sempre foi assim: o `AddToken` e os irmãos dele já eram
// públicos, e a assimetria entre os dois era acidente de história.
//
// O que elas NÃO fazem continua sendo o que as define: nenhuma trava, nenhuma
// gravação, nenhum aviso. Recebem o estado, mudam o estado.
func AdvanceTurn(st *SessionRuntimeState) {
	// SÓ A CENA DE AÇÃO tem vez a passar (p252): numa conversa na corte não há
	// rodada, e "passar o turno" não quer dizer nada. Antes a condição era "a
	// cena está ligada", que era a mesma coisa só porque a única cena que o app
	// sabia abrir era o combate.
	if !st.CountsRounds() || len(st.Initiative) == 0 {
		return
	}
	if st.TurnIndex < 0 {
		st.TurnIndex = 0
		if st.Round < 1 {
			st.Round = 1
		}
		st.TurnsTaken++
		return
	}
	st.TurnIndex++
	st.TurnsTaken++
	if st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
	RefreshTurn(st)
}

// RewindTurn desfaz um "Próximo turno". Cruzar a virada de volta devolve a
// rodada; desfazer o primeiro turno devolve ao pré-combate (turnIndex -1) sem
// zerar a rodada, porque a rodada 1 JÁ começou e voltar não desfaz isso.
func RewindTurn(st *SessionRuntimeState) {
	if len(st.Initiative) == 0 || st.TurnIndex < 0 {
		return
	}
	// Desfazer um turno desconta um turno: o contador é o que JÁ aconteceu, e
	// voltar diz que não aconteceu.
	if st.TurnsTaken > 0 {
		st.TurnsTaken--
	}
	if st.TurnIndex > 0 {
		st.TurnIndex--
		return
	}
	if st.Round > 1 {
		st.TurnIndex = len(st.Initiative) - 1
		st.Round--
		return
	}
	st.TurnIndex = -1
}

// StartScene liga a cena, e só isso: a ordem se monta DEPOIS. É por esse gesto
// RedactForPlayers devolve uma CÓPIA do estado sem os PV das linhas que o mestre
// escondeu. A flag continua na cópia de propósito: o jogador precisa saber que
// existe vida ali e que ela está oculta — sem isso, "sem barra" e "escondido"
// viram a mesma coisa na tela, e o segundo é informação.
//
// Fora de cena o jogador não recebe fila NENHUMA. A trava mora aqui, e não numa
// condição de render, porque não mandar é diferente de não desenhar: a primeira
// é segurança, a segunda é UX. E mora nesta função em particular porque ela é o
// gargalo pelo qual os DOIS caminhos do estado passam — o broadcast por sala de
// papel e o ack do `get-session-state`.
func RedactForPlayers(st *SessionRuntimeState) *SessionRuntimeState {
	if !st.InScene() {
		// Rastreador limpo e não `CloneState` com a lista zerada: a rodada e o
		// contador de turnos também são da cena, e "rodada 7, ninguém na fila"
		// é uma contradição que o jogador leria como defeito.
		return EmptyRuntimeState()
	}
	out := CloneState(st)
	for i := range out.Initiative {
		e := &out.Initiative[i]
		// O PV do GRUPO é o único pool que a mesa vê sem o mestre mandar.
		hideIfSecret(e.HpHidden, e.Type == "character", &e.HpCurrent, &e.HpMax, &e.HpHidden)
		hideIfSecret(e.MpHidden, false, &e.MpCurrent, &e.MpMax, &e.MpHidden)
	}
	return out
}

// hideIfSecret apaga um pool da cópia da mesa quando ela não tem direito a ele, e
// DEIXA A MARCA quando apaga.
//
// A escolha do mestre é tri-estado no ponteiro: nulo é "ele não decidiu", e aí
// vale o `visibleByDefault`. É isso que permite o padrão ser por POOL — o PV do
// grupo nasce à vista, o PV do NPC e todo PM nascem escondidos — sem que
// "revelei de propósito" e "nunca decidi" virem a mesma coisa.
//
// A marca não é detalhe: sem ela o jogador recebe `HpMax` nulo tanto para "este
// capanga não tem PV rastreado" quanto para "o mestre está escondendo". Por isso
// só marcamos o pool que EXISTE — senão a mesa leria "PV ocultos pelo mestre" de
// um combatente que nunca teve PV.
func hideIfSecret(choice *bool, visibleByDefault bool, current, max **int64, mark **bool) {
	visible := visibleByDefault
	if choice != nil {
		visible = !*choice
	}
	if visible || *max == nil {
		return
	}
	*current, *max = nil, nil
	hidden := true
	*mark = &hidden
}

// StateForRole é o que UM socket pode ver. Existe porque o broadcast não é o
// único caminho do estado até a tela: o `ack` do `get-state` hidrata o cliente e
// responde a quem pediu. Papel desconhecido cai em jogador — errar para o lado
// que mostra seria vazar por omissão.
func StateForRole(role string, st *SessionRuntimeState) *SessionRuntimeState {
	if role == "gm" {
		return st
	}
	return RedactForPlayers(st)
}

// ResetInitiative esvazia a fila e desliga a CENA junto: reiniciar é voltar ao
// ponto de partida, e o ponto de partida é fora de cena com a fila vazia. Deixar
// a cena ligada aqui produziria o estado "em cena, ninguém na fila" sem ninguém
// ter pedido por ele.
func ResetInitiative(st *SessionRuntimeState) {
	st.Initiative = []InitiativeEntry{}
	st.Round = 0
	st.TurnIndex = -1
	st.TurnsTaken = 0
	st.Scene = nil
}

func PtrInt64(v int64) *int64 { return &v }

func DerefOr(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}
