package sheet

// O estado de JOGO da ficha (ALE-222): o que muda durante a partida e não é a
// ficha em si — os situacionais ligados, os usos gastos e o preço pago pelas
// posturas.
//
// Os três viviam em `localStorage`. Decisão do dono, 2026-08-22: "o servidor
// mantém estado, ponto final." Os comentários dos stores no front registravam a
// decisão CONTRÁRIA e foram atualizados junto, não apagados.
//
// CUIDADO com o vizinho: `conditionals` é o opt-in do JOGADOR (Fúria, Ataque
// Poderoso); `conditions` são as do LIVRO (p394-395, Caído/Atordoado) e moram na
// coluna `characters.activeConditions`. Ver a colisão C6 no GLOSSARIO.md.

// PowerUseDTO é quanto de um poder já se gastou num escopo.
type PowerUseDTO struct {
	PowerID string `json:"powerId"`
	Scope   string `json:"scope"`
	Used    int64  `json:"used"`
}

// StanceDTO é o que foi PAGO para entrar numa postura — não se ela está ligada.
// Quem diz isso é o situacional de mesmo nome, na lista `conditionals`.
type StanceDTO struct {
	Flag   string `json:"flag"`
	Steps  int64  `json:"steps"`
	PmPaid int64  `json:"pmPaid"`
}
