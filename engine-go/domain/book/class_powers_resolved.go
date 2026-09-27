package book

import (
	"encoding/json"

	"t20engine/domain/catalog"
)

// O PODER DE CLASSE É UMA COISA; A CONCESSÃO DELE É OUTRA (ALE-403).
//
// O `class-powers.json` guardava as duas na mesma linha, e por isso a regra era
// COPIADA quando mais de uma classe concedia o mesmo poder — "Aumento de
// Atributo" tinha quatorze cópias byte a byte iguais, uma por classe. Elas
// ainda não tinham divergido, e nada as impedia: é o caminho que
// `general-powers` × `origins` percorreu até ter dezenove poderes com duas
// regras (ALE-401).
//
// Agora o arquivo tem duas espécies de linha:
//
//   - o VERBETE, sem `className`, com nome, descrição, página e o que mais a
//     regra tiver;
//   - a CONCESSÃO, com `className`, o nível ou a escolha que a destrava, e um
//     `powerUid` apontando para o verbete.
//
// # Por que a resolução mora AQUI e não num lugar compartilhado
//
// Os dois consumidores que precisam dela — o `PowerCatalogs` e o
// `class_power_details` — são deste pacote. O MOTOR não precisa: nenhuma das
// nove regras duplicadas tem modificador, e o `classActiveItems` pula poder sem
// modificador antes de olhar o nome.
//
// Compartilhar com o motor exigiria uma aresta nova entre `domain/engine` e
// `domain/catalog`, que hoje não se conhecem — e ela pagaria por um problema
// que o motor não tem.
func classPowerRows() []map[string]json.RawMessage {
	raw, ok := catalog.Resource("class-powers")
	if !ok {
		return nil
	}
	var linhas []map[string]json.RawMessage
	if json.Unmarshal(raw, &linhas) != nil {
		return nil
	}

	verbetePorUid := map[string]map[string]json.RawMessage{}
	for _, linha := range linhas {
		if _, éConcessão := linha["className"]; éConcessão {
			continue
		}
		uid := textoDe(linha["uid"])
		if uid != "" {
			verbetePorUid[uid] = linha
		}
	}

	fora := make([]map[string]json.RawMessage, 0, len(linhas))
	for _, linha := range linhas {
		if _, éConcessão := linha["className"]; !éConcessão {
			continue // o verbete não é um poder POSSUÍDO por ninguém
		}
		verbete := verbetePorUid[textoDe(linha["powerUid"])]
		if verbete == nil {
			fora = append(fora, linha) // concessão que carrega a própria regra
			continue
		}
		// A concessão vence sobre o verbete: ela é quem diz o nível, e nada
		// mais dela pode sobrescrever a regra porque ela não tem mais nada.
		juntas := make(map[string]json.RawMessage, len(verbete)+len(linha))
		for k, v := range verbete {
			juntas[k] = v
		}
		for k, v := range linha {
			juntas[k] = v
		}
		fora = append(fora, juntas)
	}
	return fora
}

// classPowersResolved devolve o `class-powers.json` com os apontamentos já
// expandidos, na forma que os leitores deste pacote sempre esperaram.
func classPowersResolved() []byte {
	linhas := classPowerRows()
	if linhas == nil {
		return nil
	}
	fora, err := json.Marshal(linhas)
	if err != nil {
		return nil
	}
	return fora
}

func textoDe(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}
