#!/usr/bin/env python3
"""Confere os PODERES CONCEDIDOS contra a seção deles (p127 e p132-136).

Desmembrado do `audit-powers.py` (ALE-408): a mesma faixa p124-137 guarda quatro
famílias, e os concedidos são os únicos que moram em DOIS catálogos —
`divine-powers.json` (80 pares, a lista completa) e `granted-powers.json` (48
pares, o subconjunto que o motor alcança). A divisão é deliberada e o
`catalog.go` a explica.

Que os dois CONCORDEM entre si não precisa do livro, e por isso essa metade desceu
para Go, no `TestEveryGrantedPowerSaysTheSameThingInBothFiles`. O que precisa do
livro, e mora aqui, é se o que eles dizem é o que a página diz.

As duas âncoras
---------------
O par (deus, poder) é declarado duas vezes: na TABELA da p127, cuja coluna de
pré-requisito é literalmente "Devoto de Azgher", e no VERBETE, que traz o nome do
deus num rótulo ao lado do título. A razão de haver duas âncoras, e o desenho de
ler a tabela primeiro, são do `engine-go/CLAUDE.md`.

O que é específico DESTA seção, e contraria o auditor irmão:

- **o título QUEBRA em duas linhas** ("Afinidade com" / "a Tormenta"). Nos poderes
  gerais o casamento de duas linhas era falso positivo e foi retirado; aqui ele é
  obrigatório. A decisão é do terreno, não da família;
- **o rótulo do deus tem duas diagramações** — linha própria ("Aharadak") ou
  colado na segunda linha do título ("Primordiais Kallyadranoch, Megalokk");
- **um poder pode ter DOIS deuses**, na tabela ("Devoto de Lena ou Thyatis") e no
  verbete ("Thwor, Valkaria").

Uso: `python3 scripts/audit-granted-powers.py`. Precisa do PDF (veja t20pdf).
Ele PROPÕE: nada é escrito.
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    RAIZ, blocos_de_tabela, celulas_da_tabela, chave, cobertura, junta, leitura,
    linhas_do_par)

PRIMEIRA, ULTIMA = 138, 142  # PDF; os verbetes, livro p132-136
PAGINA_DA_TABELA = 133       # PDF; livro p127
OFFSET_DO_PDF = 6
CONCEDIDOS = RAIZ / 'engine-go/domain/catalog/data/granted-powers.json'
DIVINOS = RAIZ / 'engine-go/domain/catalog/data/divine-powers.json'
DEUSES = RAIZ / 'engine-go/domain/catalog/data/gods.json'
COBERTURA_MINIMA = 0.5
RE_DEVOTO = re.compile(r'Devoto d[eoa]s?\s+(.+)', re.I)
SEPARA_DEUSES = re.compile(r',| e | ou ')


def deuses_da_frase(frase: str, conhecidos: set[str]) -> set[str] | None:
    """Os deuses que uma frase nomeia, ou None se ela não é uma lista de deuses.

    Devolver None e não um conjunto vazio é o que separa "esta linha não fala de
    deus" de "esta linha fala de zero deuses" — a segunda não existe, e tratar as
    duas igual faria toda linha de prosa virar um rótulo vazio.
    """
    if devoto := RE_DEVOTO.match(frase.strip()):
        frase = devoto.group(1)
    partes = [chave(p) for p in SEPARA_DEUSES.split(frase) if p.strip()]
    if partes and all(p in conhecidos for p in partes):
        return set(partes)
    return None


def le_a_tabela(conhecidos: set[str]) -> dict:
    """Âncora 1: `{chave do poder: {deuses}}` da tabela da p127."""
    da_tabela = {}
    for pares, corpo in blocos_de_tabela(PAGINA_DA_TABELA):
        for x_poder, x_pre in pares:
            linhas, _consumidas = linhas_do_par(corpo, x_poder, x_pre)
            for nome, pre, _nivel in linhas:
                lidos = deuses_da_frase(pre, conhecidos)
                if lidos is not None:
                    da_tabela[chave(nome)] = (nome, lidos)
    return da_tabela


def le_os_verbetes(nomes: dict, conhecidos: set[str]) -> tuple[dict, dict]:
    """Âncora 2: `{chave: (nome, página, {deuses}, regra)}`."""
    linhas, consumidas = [], celulas_da_tabela(PAGINA_DA_TABELA)
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        fora = consumidas if pagina == PAGINA_DA_TABELA else set()
        for x, y, texto in leitura(pagina):
            if (round(x, 1), round(y, 1)) not in fora:
                linhas.append((pagina, texto))

    achados, vistos = [], {}
    for i, (pagina, texto) in enumerate(linhas):
        # O TÍTULO E O RÓTULO PODEM DIVIDIR A LINHA, de duas formas: "Êxtase da
        # Loucura Aharadak, Nimb" numa linha só, e "Presas" / "Primordiais
        # Kallyadranoch, Megalokk" em duas. Por isso o rótulo se captura AQUI, no
        # casamento do título: o que sobra depois do nome é ele.
        #
        # Procurá-lo depois, varrendo sufixos das linhas seguintes, foi a
        # primeira versão e ela leu o deus de dentro do PRÓPRIO NOME: o "Inimigo
        # de Tenebra" não tem rótulo nenhum — ele é concedido por Azgher —, e o
        # sufixo "Tenebra" do título passou por rótulo, acusando o livro de
        # discordar de si mesmo.
        emendas = [texto]
        if i + 1 < len(linhas):
            emendas.append(f'{texto} {linhas[i + 1][1]}')
        achou = None
        for emenda in emendas:
            palavras = emenda.split()
            for corte in range(len(palavras), 0, -1):
                k = chave(' '.join(palavras[:corte]))
                if k in nomes and k not in vistos:
                    achou = (k, ' '.join(palavras[corte:]))
                    break
            if achou:
                break
        if achou:
            k, resto = achou
            vistos[k] = vistos.get(k, 0) + 1
            achados.append((i, pagina, k, deuses_da_frase(resto, conhecidos)))

    verbetes = {}
    for n, (i, pagina, k, colado) in enumerate(achados):
        fim = achados[n + 1][0] if n + 1 < len(achados) else len(linhas)
        corpo = [t for _p, t in linhas[i:fim]]
        # Se o rótulo não veio colado ao título, ele está numa linha PRÓPRIA logo
        # abaixo — e só ela vale, para o nome do poder nunca entrar na conta.
        lidos = colado
        if lidos is None:
            for linha in corpo[1:4]:
                lidos = deuses_da_frase(linha, conhecidos)
                if lidos:
                    break
        verbetes[k] = (nomes[k], pagina - OFFSET_DO_PDF, lidos, junta(corpo))
    return verbetes, vistos


def main() -> int:
    concedidos = json.load(open(CONCEDIDOS, encoding='utf-8'))
    divinos = json.load(open(DIVINOS, encoding='utf-8'))
    conhecidos = {chave(g['name']) for g in json.load(open(DEUSES, encoding='utf-8'))}
    nomes = {chave(p['name']): p['name'] for p in concedidos}
    por_chave = {chave(p['name']): p for p in concedidos}

    da_tabela = le_a_tabela(conhecidos)
    verbetes, vistos = le_os_verbetes(nomes, conhecidos)

    medidos = {'deuses': 0, 'pagina': 0, 'regra': 0}
    divergem, nas_duas, numa_so, em_nenhuma, do_livro = 0, 0, 0, [], []
    coberturas = []
    for k, poder in por_chave.items():
        do_catalogo = {chave(d) for d in poder.get('deuses', [])}
        tabela = da_tabela.get(k)
        verbete = verbetes.get(k)
        se_tabela = tabela[1] if tabela else None
        se_verbete = verbete[2] if verbete else None
        if se_tabela is None and se_verbete is None:
            em_nenhuma.append(f'{poder["name"]}: não ancorou em nenhuma das duas '
                              f'declarações — os deuses dele NÃO foram medidos')
        elif se_tabela is None or se_verbete is None:
            numa_so += 1
        else:
            nas_duas += 1
            if se_tabela != se_verbete:
                do_livro.append(f'{poder["name"]}: a tabela diz {sorted(se_tabela)} e o '
                                f'verbete diz {sorted(se_verbete)}')
        lido = se_verbete or se_tabela
        if lido is not None:
            medidos['deuses'] += len(lido)
            if lido != do_catalogo:
                print(f'  {poder["name"]}: deuses catálogo {sorted(do_catalogo)} × '
                      f'livro {sorted(lido)}')
                divergem += 1
        if verbete:
            medidos['pagina'] += 1
            if poder.get('bookPage') != verbete[1]:
                print(f'  {poder["name"]}: bookPage catálogo {poder.get("bookPage")} × '
                      f'livro {verbete[1]}')
                divergem += 1
            regra = verbete[3]
            if regra and poder.get('effect'):
                medidos['regra'] += 1
                razao, juntas, total = cobertura(poder['effect'], regra)
                coberturas.append((razao, poder['name']))
                if razao < COBERTURA_MINIMA:
                    print(f'  {poder["name"]}: {juntas} das {total} palavras do '
                          f'CATÁLOGO estão no livro ({razao:.0%}, piso '
                          f'{COBERTURA_MINIMA:.0%})')
                    divergem += 1

    if do_livro:
        print('\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO):')
        for linha in do_livro:
            print(f'  {linha}')
    for linha in em_nenhuma:
        print(f'  NÃO MEDIDO {linha}')
    repetidos = {k: v for k, v in vistos.items() if v > 1}
    if repetidos:
        print(f'  NÃO MEDIDO títulos achados mais de uma vez: {sorted(repetidos)}')

    print(f'\npoderes concedidos: {len(concedidos)} | pares no divine: {len(divinos)} '
          f'| ancoraram nas DUAS declarações: {nas_duas} | em UMA só: {numa_so} '
          f'| em NENHUMA (≠ corretos): {len(em_nenhuma)}')
    print(f'lidos na tabela da p127: {len(da_tabela)} | verbetes achados: '
          f'{len(verbetes)}/{len(concedidos)}')
    print('COMPARADOS: ' + ', '.join(f'{c} {n}' for c, n in medidos.items()))
    print(f'cobertura da regra, da pior para a melhor (piso {COBERTURA_MINIMA:.0%}):')
    for razao, nome in sorted(coberturas)[:8]:
        print(f'  {razao:5.0%} {nome}')
    print(f'divergem do livro: {divergem} | o livro discorda de si: {len(do_livro)}')
    return 1 if divergem or do_livro or em_nenhuma else 0


if __name__ == '__main__':
    raise SystemExit(main())
