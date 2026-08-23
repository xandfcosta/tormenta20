package api

import (
	"context"
	"database/sql"
	"errors"

	"t20engine/db/sqlcgen"
)

// ENTRAR NUMA MESA, como regra e não como manipulador HTTP (ALE-249).
//
// Esta é a NONA vez que a migração encontra regra soldada ao transporte, e a
// maior de longe: o `handleAddMember` tinha ~110 linhas com SETE checagens
// misturadas a `writeError`. Duas telas precisam das sete — a da SPA, que ainda
// existe, e a cena do servidor — e enquanto elas viviam dentro de um
// `http.HandlerFunc` a segunda só tinha duas saídas: chamar a própria rota por
// dentro, ou copiar as sete.
//
// Cada recusa tem ERRO PRÓPRIO, e não uma string, porque cada uma vira uma
// frase diferente na tela e um status diferente no fio. Quem chama decide as
// duas coisas; a regra não sabe se existe HTTP.

var (
	errCampanhaInexistente   = errors.New("campanha não existe")
	errConviteExigido        = errors.New("convite válido é obrigatório")
	errPersonagemInexistente = errors.New("personagem não existe")
	errPersonagemDeOutro     = errors.New("personagem é de outra pessoa")
	errPapelInvalido         = errors.New("papel inválido")
	errJaTemPersonagem       = errors.New("já tem personagem nesta campanha")
)

// pedidoDeEntrada é o que se precisa saber para deixar alguém sentar à mesa.
type pedidoDeEntrada struct {
	CampanhaID   int64
	PersonagemID int64
	// Convite é dispensado para o DONO da campanha, que não precisa de convite
	// para a própria mesa.
	Convite  string
	Papel    string
	QuemPede int64
}

// entrarNaMesa aplica as sete travas e faz o instantâneo.
//
// A ORDEM importa e é a mesma do handler original: campanha, depois convite,
// depois personagem. Checar o personagem antes do convite diria a um estranho
// se um id de personagem existe — informação que ele não deveria conseguir
// sondar sem estar convidado.
func (s *Server) entrarNaMesa(ctx context.Context, p pedidoDeEntrada) (sqlcgen.CampaignMember, error) {
	c, err := s.queries.GetCampaign(ctx, p.CampanhaID)
	if errors.Is(err, sql.ErrNoRows) {
		return sqlcgen.CampaignMember{}, errCampanhaInexistente
	}
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	// O dono entra sem convite; qualquer outra pessoa precisa do token EXATO.
	if c.Ownerid != p.QuemPede {
		if !c.Invitetoken.Valid || p.Convite == "" || p.Convite != c.Invitetoken.String {
			return sqlcgen.CampaignMember{}, errConviteExigido
		}
	}

	dono, err := s.queries.GetCharacterOwner(ctx, p.PersonagemID)
	if errors.Is(err, sql.ErrNoRows) {
		return sqlcgen.CampaignMember{}, errPersonagemInexistente
	}
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	if dono != p.QuemPede {
		return sqlcgen.CampaignMember{}, errPersonagemDeOutro
	}

	papel := p.Papel
	if papel == "" {
		papel = "player"
	}
	if !campaignMemberRoles[papel] {
		return sqlcgen.CampaignMember{}, errPapelInvalido
	}

	// As duas travas abaixo já falharam ABERTAS uma vez (ALE-156): o erro era
	// descartado com `_`, e erro de banco virava `false`, que significa "pode
	// entrar". Checagem de autorização ou de unicidade nunca descarta erro: na
	// dúvida, NEGA. O `return err` aqui é o que garante isso.
	if papel == "player" {
		temPc, err := s.queries.HasPlayerPc(ctx, sqlcgen.HasPlayerPcParams{
			Campaignid: p.CampanhaID, Ownerid: p.QuemPede,
		})
		if err != nil {
			return sqlcgen.CampaignMember{}, err
		}
		if temPc {
			return sqlcgen.CampaignMember{}, errJaTemPersonagem
		}
	}

	// Modelo de INSTANTÂNEO (ALE-33): o personagem do elenco é um molde, e a
	// mesa guarda uma CÓPIA dele. A deduplicação é por "este molde já foi
	// copiado aqui" e não por participação do molde — o molde nunca é membro.
	temCopia, err := s.campaignHasCopyOf(ctx, p.PersonagemID, p.CampanhaID)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	if temCopia {
		return sqlcgen.CampaignMember{}, errAlreadyInCampaign
	}

	// O `joinCampaign` pode devolver `errAlreadyInCampaign` por conta própria:
	// é a corrida perdida para um pedido simultâneo, e o desfecho é o mesmo da
	// checagem de fora — não um erro interno que culparia o servidor.
	return s.joinCampaign(ctx, p.PersonagemID, p.CampanhaID, p.QuemPede, papel)
}
