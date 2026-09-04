package api

import (
	"context"

	"t20engine/db/sqlcgen"
)

// O ESTADO DE JOGO da ficha vive em `sheet` desde a ALE-278: `sheet.sheet.PowerUseDTO`
// e `sheet.sheet.StanceDTO` são forma de DADO e viajam dentro do `sheet.CharacterDTO`.
// O que ficou aqui são os quatro handlers que os gravam, que é encanamento.

// clearScenePlayState é o que o DESCANSO DE CENA leva embora: os usos "1/cena" e
// as posturas.
//
// Ele se pendura no descanso da ficha e não no `EndScene` da sessão de
// propósito: os usos entram pelo caminho que JÁ limpa a ficha, junto dos
// efeitos. Era o `EndScene` da sessão que estava errado — ele não limpava
// efeito nenhum, e a bênção de duração "cena" sobrevivia ao fim da cena. A
// ALE-220 fechou isso pelo lado de lá: encerrar a cena agora percorre o grupo e
// chama ESTE caminho para cada ficha.
func (s *Server) clearScenePlayState(ctx context.Context, id int64) error {
	if err := s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: id, Scope: "scene",
	}); err != nil {
		return err
	}
	return s.queries.ClearCharacterStances(ctx, id)
}

// clearDayPlayState é o DESCANSO DE DIA: leva o da cena e mais os usos "1/dia".
func (s *Server) clearDayPlayState(ctx context.Context, id int64) error {
	if err := s.clearScenePlayState(ctx, id); err != nil {
		return err
	}
	return s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: id, Scope: "day",
	})
}
