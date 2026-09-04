package api

import (
	"database/sql"
	"t20engine/db/sqlcgen"
)

// AS REGRAS DE CAMPANHA E DE MESA, com casa própria (ALE-278, fatia 6).
//
// Treze métodos que respondem às mesmas duas perguntas: **de quem é esta mesa**
// e **quem pode entrar nela**. Eram do `*Server` pelo motivo de sempre — os
// handlers JSON que os chamavam eram dele — e a ALE-277 apagou esses handlers.
// Sobraram duas chamadoras, e nenhuma é um servidor: a cena de campanhas e a
// cena da Mesa.
//
// O `resolveRole` é a mais compartilhada e a que mais justifica o tipo: ela
// decide o PAPEL de quem pede (dono é "gm", quem tem personagem membro é
// "player", o resto é recusa), e as duas cenas dependem dela para desenhar
// coisas diferentes. Deixá-la no servidor obrigaria as duas a receber um
// servidor.
//
// Ele carrega as consultas e o `*sql.DB` — entrar numa mesa CLONA o personagem
// e escreve o membro na MESMA transação, e o clone sem o membro é um herói
// duplicado que não está em mesa nenhuma.
//
// O receptor é `rules` e não uma letra, e isso não é estilo: `campaigns.go` já
// usa `c` para a `sqlcgen.Campaign` em nove lugares, e um receptor `c` fez o
// compilador reclamar de `c.Ownerid` num tipo que não tem dono. A primeira
// tentativa desta fatia morreu assim.
type campaignRules struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func (s *Server) campaignRules() campaignRules {
	return campaignRules{db: s.db, queries: s.queries}
}
