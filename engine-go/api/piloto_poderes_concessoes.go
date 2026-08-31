package api

import (
	"encoding/json"
	"net/http"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// AS CONCESSÕES DE UMA POSTURA (ALE-272, fatia 8).
//
// Entrar em Fúria não muda só uma flag: os poderes de GATILHO daquela flag que
// o personagem possui e que CONCEDEM alguma coisa viram efeito de verdade na
// ficha. Hoje o catálogo tem um caso — a Alma de Bronze do bárbaro (p41), que
// dá uma reserva de PV temporários de nível + Força enquanto a Fúria dura — e
// sair da postura leva a reserva embora.
//
// # Por que este arquivo não chama o handler JSON
//
// O `applyPowerGrant` da API JSON escreve na resposta HTTP no MEIO da
// transação: ele decide o status e o corpo dentro do mesmo bloco que grava. Não
// dá para reusá-lo de um comando do Datastar sem refatorar um caminho que a SPA
// ainda usa — e a SPA sai na próxima fatia. O que se compartilha é a REGRA, que
// é o que importa: o `planPoolSupremacy` (vale o maior, p256) e o
// `parseTempHpPools` são os mesmos. O que se repete são as vinte linhas de
// transação, e elas morrem com o handler JSON.

// aplicaAsConcessoesDaPostura liga o que a flag concede.
//
// Uma concessão que falha NÃO derruba a postura que a pessoa acabou de pagar: o
// erro sobe como recusa e o PM já foi cobrado, então engolir seria pior. Por
// isso o laço para na primeira falha e devolve — a postura fica em pé com o que
// já aplicou, que é o estado que a tela mostra.
func (s *Server) aplicaAsConcessoesDaPostura(
	r *http.Request, row sqlcgen.Character, flag string,
) error {
	for _, spec := range asConcessoesDaFlag(row, flag) {
		if err := s.aplicaUmaConcessao(r, row, spec); err != nil {
			return err
		}
	}
	return nil
}

// removeAsConcessoesDaPostura apaga os efeitos que a postura tinha ligado.
//
// A busca é pelo id do PODER na coluna `catalogId` do efeito — é assim que o
// efeito guarda de onde veio, e é o que permite encerrar sem lembrar de nada
// entre uma requisição e outra.
func (s *Server) removeAsConcessoesDaPostura(
	r *http.Request, row sqlcgen.Character, flag string,
) error {
	daFlag := map[string]bool{}
	for _, spec := range asConcessoesDaFlag(row, flag) {
		daFlag[spec.ID] = true
	}
	if len(daFlag) == 0 {
		return nil
	}
	efeitos, err := s.queries.ListActiveEffectsByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, efeito := range efeitos {
		if !daFlag[efeito.Catalogid] {
			continue
		}
		if err := s.queries.DeleteEffectByID(r.Context(), efeito.ID); err != nil {
			return err
		}
	}
	return nil
}

// asConcessoesDaFlag são as ativações de gatilho daquela flag que concedem algo.
//
// Ela NÃO filtra pelo que o personagem possui, e isso é uma folga deliberada:
// quem chega aqui já entrou na postura, e a postura é de uma classe. Filtrar
// duas vezes daria uma segunda leitura da posse, que é justamente o que a aba
// faz uma vez só.
func asConcessoesDaFlag(row sqlcgen.Character, flag string) []activationOfBook {
	fora := []activationOfBook{}
	for _, spec := range activationsOfBook() {
		if spec.RequiresFlag == flag && spec.Grant != nil {
			fora = append(fora, spec)
		}
	}
	return fora
}

// aplicaUmaConcessao grava o efeito de uma concessão.
func (s *Server) aplicaUmaConcessao(
	r *http.Request, row sqlcgen.Character, spec activationOfBook,
) error {
	if spec.Grant.Kind != "temp-hp" {
		return nil
	}
	quanto, ok := s.powerTempHpAmount(r, row, spec.Grant.Attribute)
	if !ok {
		return nil
	}
	return s.gravaAReservaDePv(r, row.ID, spec.ID, spec.Grant.Scope, quanto)
}

// gravaAReservaDePv aplica a reserva de PV temporários sob o VALE O MAIOR.
//
// "Se você receber PV temporários de mais de uma fonte, considere apenas o
// maior valor" (p256). Quem decide isso é o `planPoolSupremacy`, o mesmo plano
// que a API JSON usa — uma segunda conta aqui daria duas respostas para a
// pergunta de qual reserva sobrevive.
func (s *Server) gravaAReservaDePv(
	r *http.Request, id int64, powerID, scope string, quanto int,
) error {
	mods := []map[string]any{
		{"target": map[string]any{"k": "tempHp"}, "amount": quanto, "bonusType": "untyped", "note": "PV temporários"},
	}
	modJSON, _ := json.Marshal(mods)

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)

	efeitos, err := q.ListActiveEffectsByCharacter(r.Context(), id)
	if err != nil {
		return err
	}
	plano := planPoolSupremacy(parseTempHpPools(efeitos), powerID, scope, quanto)
	// SUPERADA: já existe reserva maior, e nada é escrito. Não é erro — é a
	// regra do livro dizendo que esta não vale.
	if plano.superseded {
		return tx.Commit()
	}
	for _, z := range plano.zeroWrites {
		if err := q.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{
			Modifiers: z.modifiers, ID: z.effectID,
		}); err != nil {
			return err
		}
	}
	for _, apagar := range plano.deleteIDs {
		if err := q.DeleteEffectByID(r.Context(), apagar); err != nil {
			return err
		}
	}
	if _, err := q.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: "power", Catalogid: powerID, Scope: scope,
		Modifiers: string(modJSON), Createdat: plataforma.NowISO(),
	}); err != nil {
		return err
	}
	return tx.Commit()
}
