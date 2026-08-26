package api

// O VERBETE DO LIVRO VIRANDO BLOCO DO MESTRE (ALE-269, superfície 6b),
// portado de `frontend/src/features/gm-tools/creature-from-monster.ts`.
//
// É o "editar este ogro" que a ALE-137 pediu, e é o caminho PRINCIPAL de criar
// um NPC — decisão do dono: a maioria dos NPCs de campanha nasce como cópia do
// bestiário com dois ou três números mexidos, e escrever do zero é a exceção.
// Os dois caminhos terminam no mesmo formulário; o que muda é a SEMENTE.
//
// Os campos passam direto, porque o livro modela criatura e NPC do mesmo jeito.
// As duas coisas que NÃO passam direto estão comentadas onde acontecem.

// copiaDoVerbete copia um verbete do livro para um bloco editável do mestre.
//
// @example copiaDoVerbete(ogro).SourceMonsterID // "ogro"
func copiaDoVerbete(v verbete) CreatureBlock {
	return CreatureBlock{
		ND:         v.ND,
		Tipo:       v.Tipo,
		Size:       v.Size,
		Iniciativa: v.Iniciativa,
		Percepcao:  v.Percepcao,
		// O PM ATRAVESSA COMO PONTEIRO, e não desreferenciado: a maioria das
		// criaturas não tem a linha, e um zero diria "tem mana e está sem" —
		// que é outro estado. Os dois lados guardam a ausência de propósito.
		PM:           v.PM,
		Defesa:       v.Defesa,
		Fortitude:    v.Fortitude,
		Reflexos:     v.Reflexos,
		Vontade:      v.Vontade,
		HP:           v.HP,
		Deslocamento: v.Deslocamento,
		// ATRIBUTO AUSENTE VIRA ZERO, e esta é uma PERDA CONHECIDA — a mesma que
		// a SPA documenta e aceita. Nove verbetes têm `inteligencia: null` e um
		// tem `forca: null`, porque o livro escreve TRAVESSÃO: o Zumbi não tem
		// Inteligência (p297). No bloco do mestre isso vira 0, e "+0" afirma que
		// ele tem a média de um humano.
		//
		// Aceitar a perda é deliberado nos dois lados: o bloco é NUMÉRICO e não
		// sabe dizer "não tem", e ensiná-lo exigiria mexer no struct, no
		// formulário e na validação. A partir da cópia o bloco é DELE e ele
		// edita; quem guarda a ausência de verdade é o CATÁLOGO, que é a fonte
		// (ALE-151).
		Forca:            ouZero(v.Forca),
		Destreza:         ouZero(v.Destreza),
		Constituicao:     ouZero(v.Constituicao),
		Inteligencia:     ouZero(v.Inteligencia),
		Sabedoria:        ouZero(v.Sabedoria),
		Carisma:          ouZero(v.Carisma),
		Attacks:          copiaOsAtaques(v.Attacks),
		Skills:           copiaAsPericias(v.Skills),
		Equipment:        v.Equipamento,
		Treasure:         v.Tesouro,
		SpecialAbilities: copiaAsFrases(v.SpecialAbilities),
		SourceMonsterID:  v.ID,
	}
}

func ouZero(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

// As três cópias abaixo existem para o bloco do mestre não COMPARTILHAR fatia
// com o catálogo embutido: o catálogo é imutável e servido a todo mundo, e o
// bloco nasce para ser editado. Sem a cópia, mexer num ataque do NPC mexeria no
// verbete que o bestiário desenha para a mesa inteira — e a fonte do livro
// passaria a mentir para quem a consultasse depois.
func copiaOsAtaques(de []CreatureAttack) []CreatureAttack {
	fora := make([]CreatureAttack, len(de))
	copy(fora, de)
	return fora
}

func copiaAsPericias(de []CreatureSkill) []CreatureSkill {
	fora := make([]CreatureSkill, len(de))
	copy(fora, de)
	return fora
}

func copiaAsFrases(de []string) []string {
	fora := make([]string, len(de))
	copy(fora, de)
	return fora
}
