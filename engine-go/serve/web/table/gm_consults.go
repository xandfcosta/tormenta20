package table

import (
	"encoding/json"
	"fmt"
)

// As abas das Consultas do mestre (ALE-522). O valor é o do sinal
// `$gm_consult`, e a primeira é a que abre.
const (
	gmConsultNotes    = "notes"
	gmConsultGroup    = "group"
	gmConsultCast     = "cast"
	gmConsultBestiary = "bestiary"
)

var gmConsultTabs = []struct{ ID, Label string }{
	{gmConsultNotes, "Notas"},
	{gmConsultGroup, "Grupo"},
	{gmConsultCast, "Elenco"},
	{gmConsultBestiary, "Bestiário"},
}

// consultCurrent é o `aria-current` da aba. O falso REMOVE o atributo, que é o
// que o `.ds-tabs` lê.
func consultCurrent(id string) string {
	return fmt.Sprintf("$gm_consult === %q ? 'page' : false", id)
}

func consultShown(id string) string { return fmt.Sprintf("$gm_consult === %q", id) }

// groupAtTable é a contagem do cabeçalho do Grupo: "2 de 3 na mesa".
func groupAtTable(group []Member) string {
	here := 0
	for _, m := range group {
		if m.Presence != nil && m.Presence.AtTable {
			here++
		}
	}
	return fmt.Sprintf("%d de %d na mesa", here, len(group))
}

// memberMeta é a linha de baixo do nome: o nível, as classes e a Defesa —
// partida quando o personagem está caído (p394), porque é aqui que o mestre a
// confere para decidir se o ataque acerta.
func memberMeta(m Member) string {
	meta := fmt.Sprintf("Nível %d", m.Level)
	if m.Classes != "" {
		meta += " · " + m.Classes
	}
	meta += " · Defesa " + m.Defense
	if m.Presence != nil && !m.Presence.AtTable {
		meta += " · fora da mesa"
	}
	return meta
}

// poolOrNothing não desenha o pool que o personagem não tem: um "PM 0/0" de
// quem não usa mana é uma barra vazia que parece um herói esgotado.
func poolOrNothing(b tableBar) *tableBar {
	if b.Max == 0 {
		return nil
	}
	return &b
}

// npcMeta é a linha do livro, e a origem quando o mestre o renomeou.
func npcMeta(npc castNpc) string {
	if npc.FromBook != "" && npc.FromBook != npc.Name {
		return npc.Summary + " · cópia de " + npc.FromBook
	}
	return npc.Summary
}

// startsEditingNotes abre a caixa e põe o cursor nela. O foco espera o
// quadro seguinte, porque a caixa só aparece quando o Datastar reavaliar o
// `data-show` dela.
const startsEditingNotes = "$gm_notes_editing = true, requestAnimationFrame(() => document.getElementById('gm-notes-text')?.focus())"

// nameMatches é o filtro de uma linha pela busca da aba: vazio mostra todos, e
// o nome casa por pedaço, sem caixa. O nome vai como literal JSON, que é texto
// JavaScript válido com aspas e acentos escapados.
func nameMatches(signal, name string) string {
	literal, err := json.Marshal(name)
	if err != nil {
		literal = []byte(`""`)
	}
	return fmt.Sprintf("$%s === '' || %s.toLowerCase().includes($%s.toLowerCase())", signal, literal, signal)
}
