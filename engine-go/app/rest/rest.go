// Package rest são os gestos do painel de RECUPERAÇÃO, e os dois não fazem a
// mesma coisa.
//
// O glossário já brigou com essa confusão e ganhou (ALE-233): **`rest` é o
// PAINEL**, e dentro dele mora UMA ação de descanso — o **descanso de dia**, que
// é a palavra do livro (T20 p106) e devolve PV e PM. A outra, **expirar efeitos
// de cena**, não devolve ponto nenhum: ela encerra a duração "cena". Os nomes
// daqui seguem essa divisão, e nenhum chama a primeira de descanso.
//
// # Por que um caso de uso, e não um método de adaptador
//
// Os três gestos por ficha têm DOIS chamadores, que é o sinal que paga a camada:
// a mesa inteira (o mestre descansando o grupo) e a própria ficha (o jogador
// encerrando a cena dele, pela aba Efeitos). Enquanto viviam no `serve/api`, o
// segundo chamador só os alcançava porque estava no mesmo pacote.
//
// # A CONTA não está aqui
//
// A recuperação por nível e acomodação é regra do livro e mora pura em
// `domain/sheet` (`AfterNightRest`), com o exemplo trabalhado do Helior a
// prendê-la. O que este pacote faz é autorizar, chamar a conta, gravar e
// espelhar.
package rest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Access é a porta de entrada de UM personagem: o dono passa, o mestre de uma
// campanha dele passa, o administrador passa, e mais ninguém.
//
// O admin entra pela mesma porta do dono e do mestre: uma mesa que ele
// administra inclui as fichas que estão nela (ALE-120).
type Access struct {
	queries *sqlcgen.Queries
}

func NewAccess(q *sqlcgen.Queries) Access { return Access{queries: q} }

// Character carrega a ficha e cobra a trava.
func (a Access) Character(ctx context.Context, quem app.Caller, id int64) (sqlcgen.Character, error) {
	row, err := a.queries.GetCharacter(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, fmt.Errorf("o personagem %d não existe: %w", id, app.ErrNotFound)
	}
	if err != nil {
		return row, fmt.Errorf("carregar o personagem %d: %w", id, err)
	}
	if row.Ownerid == quem.ID || quem.IsAdmin {
		return row, nil
	}
	mestra, err := a.queries.IsCampaignGmForCharacter(ctx, sqlcgen.IsCampaignGmForCharacterParams{
		Characterid: id, Ownerid: quem.ID,
	})
	if err != nil {
		return row, fmt.Errorf("conferir se %d mestra o personagem %d: %w", quem.ID, id, err)
	}
	if !mestra {
		return row, fmt.Errorf("o personagem %d é de outra pessoa: %w", id, app.ErrForbidden)
	}
	return row, nil
}

// Scopes são os gestos por FICHA: o que acaba quando a cena ou o dia acaba.
type Scopes struct {
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	access   Access
}

func NewScopes(q *sqlcgen.Queries, catalogs *engine.Catalogs) Scopes {
	return Scopes{queries: q, catalogs: catalogs, access: NewAccess(q)}
}

// Access é a trava deste pacote, exposta para quem precisa só dela.
func (s Scopes) Access() Access { return s.access }

// EndScene expira a duração "cena" da ficha: os efeitos de escopo `scene`, os
// usos "1/cena" e as posturas.
//
// Os três juntos, e é isso que a ALE-220 consertou: o encerrar da SESSÃO não
// limpava efeito nenhum, e a bênção de duração "cena" sobrevivia à cena. Hoje
// existe um caminho só, e o da sessão chega aqui uma ficha por vez.
func (s Scopes) EndScene(ctx context.Context, quem app.Caller, characterID int64) error {
	if _, err := s.access.Character(ctx, quem, characterID); err != nil {
		return err
	}
	if err := s.queries.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{
		Characterid: characterID, Scope: "scene",
	}); err != nil {
		return fmt.Errorf("expirar os efeitos de cena do personagem %d: %w", characterID, err)
	}
	return s.clearScenePlay(ctx, characterID)
}

// EndDay expira a cena E o dia. É o descanso do livro, e por isso leva os dois
// escopos: dormir encerra a cena em curso junto.
func (s Scopes) EndDay(ctx context.Context, quem app.Caller, characterID int64) error {
	if _, err := s.access.Character(ctx, quem, characterID); err != nil {
		return err
	}
	if err := s.queries.DeleteSceneAndDayEffects(ctx, characterID); err != nil {
		return fmt.Errorf("expirar os efeitos de dia do personagem %d: %w", characterID, err)
	}
	if err := s.clearScenePlay(ctx, characterID); err != nil {
		return err
	}
	if err := s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: characterID, Scope: "day",
	}); err != nil {
		return fmt.Errorf("zerar os usos de dia do personagem %d: %w", characterID, err)
	}
	return nil
}

// clearScenePlay leva os usos "1/cena", as posturas e o que durava UMA VEZ.
//
// A duração de turno entra aqui porque o fim da cena é a outra ponta dela: o
// giro da vez a expira enquanto o combate corre (p233), e uma cena encerrada
// logo depois de uma reação nunca gira. Sem isto o "1 turno" sobreviveria à
// própria cena, que é o defeito que a ALE-220 já consertou para a duração
// "cena".
func (s Scopes) clearScenePlay(ctx context.Context, characterID int64) error {
	if err := s.queries.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{
		Characterid: characterID, Scope: engine.TurnScope(),
	}); err != nil {
		return fmt.Errorf("expirar os efeitos de turno do personagem %d: %w", characterID, err)
	}
	if err := s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: characterID, Scope: "scene",
	}); err != nil {
		return fmt.Errorf("zerar os usos de cena do personagem %d: %w", characterID, err)
	}
	if err := s.queries.ClearCharacterStances(ctx, characterID); err != nil {
		return fmt.Errorf("baixar as posturas do personagem %d: %w", characterID, err)
	}
	return nil
}

// NightRest aplica a recuperação de uma noite e GRAVA. Devolve os vitais novos
// para quem chama espelhá-los no rastreador.
//
// A conta é do livro e mora pura no `domain/sheet`: aqui só se autoriza, chama e
// grava.
func (s Scopes) NightRest(
	ctx context.Context, quem app.Caller, characterID int64, condicao string,
) (sheet.RestedVitals, error) {
	row, err := s.access.Character(ctx, quem, characterID)
	if err != nil {
		return sheet.RestedVitals{}, err
	}
	// A conta do livro recebe o poço DERIVADO e não a coluna: o descanso devolve
	// uma fração do máximo, e com o máximo velho um personagem que subiu de nível
	// recuperaria pelo teto de ontem.
	var depois sheet.RestedVitals
	if _, err := sheet.ApplyToPools(ctx, s.queries, s.catalogs, row,
		func(pocos sheet.Pools) (sheet.Pools, error) {
			depois = sheet.AfterNightRest(row.Level, condicao,
				sheet.RestedVitals{HpCurrent: pocos.HpCurrent, MpCurrent: pocos.MpCurrent},
				pocos.HpMax, pocos.MpMax)
			pocos.HpCurrent, pocos.MpCurrent = depois.HpCurrent, depois.MpCurrent
			return pocos, nil
		}); err != nil {
		return sheet.RestedVitals{}, fmt.Errorf("gravar os vitais do personagem %d: %w", characterID, err)
	}
	return depois, nil
}
