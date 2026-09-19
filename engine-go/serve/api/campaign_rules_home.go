package api

import (
	"t20engine/infra/db/sqlcgen"
)

// O QUE SOBROU DAS REGRAS DE CAMPANHA (ALE-348).
//
// Elas respondiam a duas perguntas — de quem é esta mesa, e quem pode entrar
// nela —, e as duas viraram caso de uso: `session.Access` e `campaign.Seating`.
// Sobram três métodos, e os três são TRANSPORTE: eles pegam a recusa tipada e
// devolvem um número de HTTP para a rota JSON da bancada.
//
// **O `*sql.DB` saiu**: entrar numa mesa clona o personagem e escreve o membro
// na MESMA transação, e essa transação é do `campaign.Seating` agora.
//
// O receptor é `rules` e não uma letra: `campaigns.go` já usa `c` para a
// `sqlcgen.Campaign`, e um receptor `c` aqui faz o compilador reclamar de
// `c.Ownerid` num tipo que não tem dono.
type campaignRules struct {
	queries *sqlcgen.Queries
}

func (s *Server) campaignRules() campaignRules {
	return campaignRules{queries: s.queries}
}
