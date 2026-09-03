package api

import (
	"fmt"
	"strings"
	"t20engine/web/master"
)

// As expressões que o Datastar executa para o elenco de NPCs (ALE-269, 6b).
//
// Elas moram num arquivo à parte do `piloto_mesa_npc.go` pela regra da casa: lá
// estão as ROTAS e o que decide, aqui está o que a tela dispara.

func comandoDoNPC(v mesaView, npc npcDoElenco, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/elenco/npc/%d/%s')",
		v.CampaignID, v.SessionID, npc.ID, acao)
}

// guardaNoElenco LIMPA o campo do nome depois de postar, e isso não é
// arrumação: o painel do bestiário é UM só para os 80 verbetes, e o campo do
// nome é um nó COMPARTILHADO por todos eles.
//
// Sem a limpeza, o mestre guarda "Ogro Capitão", troca para o Goblin e encontra
// o nome do ogro esperando por ele — e guarda um goblin chamado Ogro Capitão sem
// perceber. É a mesma família do link de redefinir senha (o guia do `engine-go/`
// a descreve inteira), com uma diferença que a torna mais fácil: aqui quem
// escreve no nó compartilhado é o mestre, não um remendo do servidor. O conserto
// é o mesmo — quem limpa é o gesto que TERMINA, porque ele sabe que houve um
// anterior.
// A rota sai da BASE do bestiário (`/mesa/{c}/{s}/bestiario`) e não de
// ids soltos: ela já carrega a campanha e a sessão, e derivar daqui é o que
// impede este botão de apontar para a mesa 0/0 — o defeito que a prévia das
// notas teve por nascer de uma view sintética.
func guardaNoElenco(v master.BestiaryView) string {
	base := strings.TrimSuffix(v.BestiaryBase(), "/bestiario")
	return fmt.Sprintf("@post('%s/elenco/npc/do-verbete'); $nomedonpc = ''", base)
}
