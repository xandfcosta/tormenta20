package sheetui

import (
	"net/http"

	"t20engine/book"
	"t20engine/db/sqlcgen"
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
// é o que importa: o `sheet.PlanPoolSupremacy` (vale o maior, p256) e o
// `sheet.ParseTempHpPools` são os mesmos. O que se repete são as vinte linhas de
// transação, e elas morrem com o handler JSON.

// aplicaAsConcessoesDaPostura liga o que a flag concede.
//
// Uma concessão que falha NÃO derruba a postura que a pessoa acabou de pagar: o
// erro sobe como recusa e o PM já foi cobrado, então engolir seria pior. Por
// isso o laço para na primeira falha e devolve — a postura fica em pé com o que
// já aplicou, que é o estado que a tela mostra.
func (s Scene) applyTheGrantsStance(
	r *http.Request, row sqlcgen.Character, flag string,
) error {
	for _, spec := range flagGrants(row, flag) {
		if err := s.applyOneGrant(r, row, spec); err != nil {
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
func (s Scene) removeTheGrantsStance(
	r *http.Request, row sqlcgen.Character, flag string,
) error {
	daFlag := map[string]bool{}
	for _, spec := range flagGrants(row, flag) {
		daFlag[spec.ID] = true
	}
	if len(daFlag) == 0 {
		return nil
	}
	efeitos, err := s.deps.Queries().ListActiveEffectsByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, efeito := range efeitos {
		if !daFlag[efeito.Catalogid] {
			continue
		}
		if err := s.deps.Queries().DeleteEffectByID(r.Context(), efeito.ID); err != nil {
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
func flagGrants(row sqlcgen.Character, flag string) []book.Activation {
	fora := []book.Activation{}
	for _, spec := range book.Activations() {
		if spec.RequiresFlag == flag && spec.Grant != nil {
			fora = append(fora, spec)
		}
	}
	return fora
}

// aplicaUmaConcessao grava o efeito de uma concessão.
func (s Scene) applyOneGrant(
	r *http.Request, row sqlcgen.Character, spec book.Activation,
) error {
	if spec.Grant.Kind != "temp-hp" {
		return nil
	}
	quanto, ok := s.deps.PowerTempHpAmount(r, row, spec.Grant.Attribute)
	if !ok {
		return nil
	}
	return s.deps.ApplyPowerTempHp(r.Context(), row.ID, spec.ID, spec.Grant.Scope, quanto)
}
