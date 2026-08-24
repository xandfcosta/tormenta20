package api

import (
	"t20engine/aovivo"
)

// A Mesa do jogador como DADO — o piloto Datastar (ALE-219).
//
// Puro de propósito: o handler busca, este arquivo decide, o template só
// desenha. É o que deixa a regra provável sem HTTP nenhum, pela mesma razão que
// o `selfInitiativeEntry` é transport-agnostic (ALE-213) — o que importa não é
// o transporte.
//
// Nenhuma regra NOVA mora aqui. O estado já chega redigido por
// `stateForRole`/`redactForPlayers`, que é o gargalo único da ALE-210, e a
// derivação do turno é a tradução literal do `playerTurnState` da SPA. Um
// piloto que reescrevesse a regra mediria a reescrita, não o Datastar.

// mesaView é uma tela inteira da Mesa. Campos exportados porque `html/template`
// não enxerga os minúsculos — a única razão, e ela é do pacote de template.
type mesaView struct {
	CampaignID int64
	SessionID  int64
	SessionNum int64
	// SceneActive vem do estado JÁ REDIGIDO: fora de cena o `redactForPlayers`
	// devolve fila limpa, então o falso aqui É a trava da ALE-210 e não
	// uma segunda decisão tomada na tela.
	SceneActive bool
	Round       int
	Turn        mesaTurn
	Grupo       []mesaMembro
	Fila        []mesaLinha
	Eu          *mesaEu
}

// mesaTurn é de quem é a vez, do ponto de vista de quem olha. Espelha o
// `LiveTurnState` da SPA.
type mesaTurn struct {
	Kind  string // "mine" | "other" | "idle"
	Label string
}

// mesaBarra é uma barra de vital já com a porcentagem e a COR resolvidas: o
// template não faz conta nem escolhe tom, porque conta em template é regra
// escondida onde ninguém a testa.
type mesaBarra struct {
	Atual int64
	Max   int64
	Pct   int
	// Tom é a CLASSE do preenchimento — a cor diz "quão mal", não só a largura
	// (espelha `hpFillVar` do `vital-bar.tsx`).
	//
	// Classe e não `var(--token)` inline por duas razões que se somam: o
	// `html/template` sanitiza contexto CSS e um `var(--hp-full)` interpolado
	// vira `ZgotmplZ`, e classe é o que o scanner do Tailwind sabe procurar.
	// Como o nome nasce aqui e não no template, o scanner NÃO o vê — por isso
	// os quatro estão declarados no `@source inline(...)` do `mesa.css`.
	Tom string
}

// mesaMembro é um personagem do grupo no cartão "Grupo".
type mesaMembro struct {
	Nome    string
	Nivel   int64
	Classes string
	PV      mesaBarra
	PM      mesaBarra
}

// mesaLinha é uma linha da fila de iniciativa como o jogador a vê.
type mesaLinha struct {
	ID         string
	Rotulo     string
	Iniciativa int
	PC         bool
	Minha      bool
	NaVez      bool
	// PV nil = linha sem vida rastreada. `Oculto` é outra coisa: o mestre
	// escondeu, e a flag sobrevive à redação de propósito (ALE-210) — "sem
	// barra" e "escondido" não são a mesma coisa, e a segunda é informação.
	PV        *mesaBarra
	Oculto    bool
	Condicoes []string
}

// mesaEu é o personagem de quem olha, quando ele tem um nesta mesa. Nil é um
// estado normal: o convidado que assiste não registra iniciativa.
type mesaEu struct {
	CharacterID int64
	Nome        string
	Bonus       int64
	NaFila      bool
}

// mesaTurnOf traduz `playerTurnState` (session-player-view.tsx) para o Go.
//
// Fora de combate ninguém está na vez. A linha na vez sendo de um personagem
// MEU é o único caso em que a faixa acende.
func mesaTurnOf(st *aovivo.SessionRuntimeState, meus map[int64]bool) mesaTurn {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return mesaTurn{Kind: "idle"}
	}
	naVez := st.Initiative[st.TurnIndex]
	if naVez.CharacterID != nil && meus[*naVez.CharacterID] {
		return mesaTurn{Kind: "mine"}
	}
	return mesaTurn{Kind: "other", Label: naVez.Label}
}

// mesaBarraDe resolve a porcentagem (presa em 0..100) e o tom.
//
// Máximo ausente ou zero devolve 0% em vez de dividir: uma linha sem pool não
// tem barra cheia nem vazia, ela não tem barra — e é quem chama que decide não
// desenhar.
func mesaBarraDe(atual, max int64, arcano bool) mesaBarra {
	barra := mesaBarra{Atual: atual, Max: max, Tom: "bg-mp-arcane"}
	if max > 0 {
		pct := int(atual * 100 / max)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		barra.Pct = pct
	}
	if !arcano {
		barra.Tom = hpTomDe(barra.Pct)
	}
	return barra
}

// hpTomDe é a tradução literal do `hpFillVar` (vital-bar.tsx): a COR da barra de
// PV diz "quão mal", e os limiares são os mesmos dos dois lados de propósito —
// duas escadas divergiriam em silêncio, cada tela chamando de "ferido" uma
// coisa diferente.
func hpTomDe(pct int) string {
	if pct <= 25 {
		return "bg-hp-critical"
	}
	if pct <= 50 {
		return "bg-hp-hurt"
	}
	return "bg-hp-full"
}

// mesaFilaDe desenha a fila que o jogador recebeu — já redigida.
func mesaFilaDe(st *aovivo.SessionRuntimeState, meus map[int64]bool) []mesaLinha {
	fila := make([]mesaLinha, 0, len(st.Initiative))
	for i := range st.Initiative {
		e := &st.Initiative[i]
		linha := mesaLinha{
			ID:         e.ID,
			Rotulo:     e.Label,
			Iniciativa: e.Initiative,
			PC:         e.Type == "character",
			Minha:      e.CharacterID != nil && meus[*e.CharacterID],
			NaVez:      i == st.TurnIndex,
			Oculto:     e.HpHidden != nil && *e.HpHidden,
			Condicoes:  e.Conditions,
		}
		// O `HpMax` nil depois da redação é como o servidor DIZ "isto não é seu
		// para ver". Desenhar barra aqui inventaria um número.
		if e.HpMax != nil {
			barra := mesaBarraDe(aovivo.DerefOr(e.HpCurrent, 0), *e.HpMax, false)
			linha.PV = &barra
		}
		fila = append(fila, linha)
	}
	return fila
}

// mesaViewOf monta a tela a partir das partes já buscadas. Tudo o que decide
// mora aqui; o handler ao lado só sabe buscar.
func mesaViewOf(
	st *aovivo.SessionRuntimeState,
	campaignID, sessionID, sessionNum int64,
	grupo []mesaMembro,
	meus map[int64]bool,
	eu *mesaEu,
) mesaView {
	if eu != nil {
		eu.NaFila = false
		for i := range st.Initiative {
			if id := st.Initiative[i].CharacterID; id != nil && *id == eu.CharacterID {
				eu.NaFila = true
				break
			}
		}
	}
	return mesaView{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		SessionNum:  sessionNum,
		SceneActive: st.SceneActive,
		Round:       st.Round,
		Turn:        mesaTurnOf(st, meus),
		Grupo:       grupo,
		Fila:        mesaFilaDe(st, meus),
		Eu:          eu,
	}
}
