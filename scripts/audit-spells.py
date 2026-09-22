#!/usr/bin/env python3
"""Confere os APRIMORAMENTOS das magias contra o livro, magia por magia (ALE-340).

Rodar é ATO DELIBERADO, como o `genoracle` e o `audit-bestiary.py`: a ferramenta
PROPÕE, o diff se revisa contra o PDF, e é a revisão que decide.

    python3 scripts/audit-spells.py                 # relatório de todas
    python3 scripts/audit-spells.py --suspeitas     # só as 32 da ALE-340
    python3 scripts/audit-spells.py --magia luz     # uma, com o texto do livro

Ele NÃO escreve no catálogo. A correção é humana, e a razão está abaixo.

Como ele sabe que está lendo a magia certa
------------------------------------------
Duas âncoras, e as duas têm de bater com o que o catálogo já tem:

1. o cabeçalho `Lista N (Escola)` — `Arcana 2 (Ilusão)`, `Universal 1
   (Evocação)` —, que dá CÍRCULO e ESCOLA;
2. a linha `Execução: …; Alcance: …`.

Com as duas batendo, o que divergir depois é defeito do catálogo. Sem elas, a
magia sai como NÃO MEDIDA — que não é a mesma coisa que "correta", e é a
distinção que a medição original da issue já fazia para as 15 dela.

Por que ele compara TEXTO e não só a CONTAGEM
---------------------------------------------
Porque contar é cego para o caso que a conferência à mão achou: o `+1 PM` da
Invisibilidade (p195) tem UM aprimoramento no livro e UM no catálogo, e o
catálogo escreve "alcance curto" onde o livro diz "o alcance para toque". A
contagem bate, o valor está errado, e um medidor que só conte diz que está tudo
bem — o mesmo erro que deixou 44 verbetes do bestiário verdes na ALE-151.

A armadilha do PDF, que é a mesma do bestiário
----------------------------------------------
`pdftotext -layout` junta colunas VIZINHAS na mesma linha de texto, e foi isso
que derrubou 15 magias na medição da issue. A leitura aqui é por COORDENADA, com
as colunas achadas por frequência do x onde o corpo começa — o mesmo código do
`audit-bestiary.py`, que já pagou esse preço.

Precisa do `pdftotext` (poppler) e do PDF do livro na raiz do repositório.
"""
import argparse
import html
import os
import pathlib
import json
import re
import subprocess
import unicodedata
from collections import Counter

# O CATÁLOGO sai da localização do script; o LIVRO, não — e a diferença é que o
# PDF é gitignorado (ele não é nosso para distribuir). Numa worktree, o
# catálogo a auditar é o de lá e o livro continua no checkout principal.
#
# Com um caminho fixo para os DOIS, rodar numa worktree audita o catálogo do
# checkout principal: o relatório sai sobre um arquivo que não é o que se está
# editando, e as correções parecem não ter pegado. Custou uma rodada.
RAIZ = pathlib.Path(__file__).resolve().parent.parent
PDF = os.environ.get('T20_BOOK_PDF') or str(RAIZ / 't20-book.pdf')
if not pathlib.Path(PDF).exists():
    # O checkout principal é o palpite seguinte, e ele é DITO: um auditor que
    # caísse em silêncio num PDF vazio reportaria 198 "não medidas" com cara de
    # resultado.
    vizinho = pathlib.Path('/mnt/HD/projects/tormenta20/t20-book.pdf')
    if not vizinho.exists():
        raise SystemExit(
            f'não achei o livro em {PDF}. Ele é gitignorado — aponte o '
            f'T20_BOOK_PDF para o PDF do checkout principal.')
    PDF = str(vizinho)
SPELLS = str(RAIZ / 'engine-go/domain/catalog/data/spells.json')

# O capítulo de Magia, em página de PDF. O livro = PDF - 6, e o catálogo guarda
# a página do LIVRO em `bookPage`.
PRIMEIRA, ULTIMA = 184, 217
OFFSET = 6

RE_BLOCO = re.compile(
    r'<block xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)"[^>]*>(.*?)</block>', re.S)
RE_LINHA = re.compile(r'<line[^>]*>(.*?)</line>', re.S)
RE_PALAVRA = re.compile(
    r'<word xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</word>', re.S)

# `Arcana 2 (Ilusão)`, `Universal 1 (Evocação)`, `Divina 3 (Abjuração)`.
#
# INSENSÍVEL A CAIXA, e isso não é frouxidão: o livro compõe alguns desses
# cabeçalhos em versalete, e a camada de texto do PDF os entrega em MINÚSCULA —
# `arcana 1 (Encantamento)`, `divina 2 (adivinhação)`. Exigindo maiúscula, o
# Hipnotismo e o Globo da Verdade de Gwen (ambos na p194) ficavam sem cabeçalho,
# e o corpo da magia ANTERIOR engolia os aprimoramentos deles: o Heroísmo
# aparecia com sete quando tem um, e o catálogo — que estava certo — parecia ter
# perdido seis.
RE_LISTA = re.compile(r'^(Arcana|Divina|Universal)\s+(\d)\s+\(([^)]+)\)\s*$', re.I)
RE_EXECUCAO = re.compile(r'^Execução:')
# `+2 PM:`, `+0 PM (Apenas Arcanos):`, `Truque:` — as três formas do livro.
RE_AUGMENT = re.compile(r'^(?:\+(\d+)\s*PM(?:\s*\(Apenas\s+([^)]+)\))?|Truque)\s*:\s*(.*)$')

# A MARCA D'ÁGUA do PDF e o rodapé entram no corpo como texto normal, e saem
# grudados no último aprimoramento da página. O molde do bestiário tem a mesma
# linha, e pela mesma razão: sem ela o texto lido não casa com nada e parece
# erro de transcrição.
LIXO = re.compile(r'Mateus Santos|mateush\.santos|^Capítulo|^\d{1,3}$')

MINIMO_DE_BLOCOS_POR_COLUNA = 3


def blocos_da_pagina(pagina: int):
    """(xMin, xMax, yMin, [linhas]) de cada bloco declarado pelo PDF."""
    xml = subprocess.run(
        ['pdftotext', '-bbox-layout', '-f', str(pagina), '-l', str(pagina), PDF, '-'],
        capture_output=True, text=True).stdout
    for m in RE_BLOCO.finditer(xml):
        x0, ybloco, x1 = float(m.group(1)), float(m.group(2)), float(m.group(3))
        linhas, y0 = [], None
        for lm in RE_LINHA.finditer(m.group(4)):
            palavras = [
                (float(x), float(y), html.unescape(t))
                for x, y, t in RE_PALAVRA.findall(lm.group(1))
            ]
            if not palavras:
                continue
            if y0 is None:
                y0 = palavras[0][1]
            palavras.sort(key=lambda w: w[0])
            linhas.append(' '.join(t for _x, _y, t in palavras))
        if linhas:
            yield x0, x1, y0 if y0 is not None else ybloco, linhas


def inicios_das_colunas(bs) -> list[float]:
    """Os x onde as colunas COMEÇAM, achados por FREQUÊNCIA.

    Não serve fundir faixas de x: o título de seção é centralizado e atravessa a
    calha, e uma linha dessas funde duas colunas numa só. O que é estável é o x
    onde o corpo começa.
    """
    contagem = Counter(round(b[0], 1) for b in bs)
    inicios = sorted(x for x, n in contagem.items() if n >= MINIMO_DE_BLOCOS_POR_COLUNA)
    return inicios or [min((b[0] for b in bs), default=0.0)]


def linhas_da_pagina(pagina: int) -> list[str]:
    """As linhas da página na ordem de LEITURA: coluna por coluna, de cima para
    baixo."""
    bs = list(blocos_da_pagina(pagina))
    if not bs:
        return []
    inicios = inicios_das_colunas(bs)

    def coluna(x0: float) -> int:
        cabem = [i for i, ini in enumerate(inicios) if x0 >= ini - 2]
        return cabem[-1] if cabem else 0

    bs.sort(key=lambda b: (coluna(b[0]), b[2]))
    saida: list[str] = []
    for _x0, _x1, _y, linhas in bs:
        saida.extend(linhas)
    return saida


def chave(nome: str) -> str:
    """Normaliza para casar nome do livro com nome do catálogo."""
    sem_acento = ''.join(
        c for c in unicodedata.normalize('NFD', nome.lower())
        if unicodedata.category(c) != 'Mn')
    return re.sub(r'[^a-z0-9]+', '', sem_acento)


def normaliza_frase(t: str) -> str:
    """O texto de um aprimoramento, comparável: sem acento, sem pontuação, sem
    espaço duplicado. A comparação é por CONTEÚDO — o catálogo reescreve as
    frases do livro em forma mais curta de propósito."""
    sem_acento = ''.join(
        c for c in unicodedata.normalize('NFD', t.lower())
        if unicodedata.category(c) != 'Mn')
    return re.sub(r'\s+', ' ', re.sub(r'[^a-z0-9 ]+', ' ', sem_acento)).strip()


def catalogo() -> dict:
    with open(SPELLS, encoding='utf-8') as f:
        return json.load(f)


def texto_do_capitulo() -> list[tuple[int, str]]:
    """(página do PDF, linha) de todo o capítulo, na ordem de leitura."""
    fora = []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        for linha in linhas_da_pagina(pagina):
            # LARGA A LINHA INTEIRA, e não o trecho que casou: a marca d'água
            # é "mateush.santos42@gmail.com", e substituir só o nome deixa
            # "42@gmail.com" grudado no fim do aprimoramento.
            limpa = linha.strip()
            if limpa and not LIXO.search(limpa):
                fora.append((pagina, limpa))
    return fora


def magias_do_livro(nomes: dict) -> dict:
    """Acha cada magia pelo par NOME + `Lista N (Escola)`, e devolve o corpo dela.

    O casamento é pelo CABEÇALHO e não por busca de nome solto: "Luz" aparece
    dezenas de vezes no capítulo, e só uma delas é o título da magia — a que tem
    a linha de lista logo abaixo.
    """
    linhas = texto_do_capitulo()
    achadas: dict[str, dict] = {}
    aberta = None
    for i, (pagina, linha) in enumerate(linhas):
        m = RE_LISTA.match(linha)
        if m and i > 0:
            titulo, k, usadas = tituloAcima(linhas, i, nomes)
            if aberta is not None:
                # O título da magia que começa agora já foi anexado à anterior —
                # ele entrou antes de sabermos que era título. São `usadas`
                # linhas, e não uma: o livro quebra nome longo em duas.
                for _ in range(usadas):
                    if aberta['corpo']:
                        aberta['corpo'].pop()
            if k in nomes:
                aberta = {
                    'id': nomes[k], 'titulo': titulo, 'pagina': pagina,
                    'lista': m.group(1), 'circulo': int(m.group(2)),
                    'escola': m.group(3), 'corpo': [],
                }
                achadas[nomes[k]] = aberta
                continue
            # Cabeçalho de uma magia que o catálogo não tem: fecha a anterior,
            # senão o corpo dela engole a magia seguinte inteira.
            aberta = None
            continue
        if aberta is not None:
            aberta['corpo'].append(linha)
    return achadas


def tituloAcima(linhas, i: int, nomes: dict):
    """O título da magia cujo cabeçalho está em `i`, e quantas LINHAS ele ocupa.

    O livro quebra nome longo em duas linhas — "Barragem elemental" / "de
    Vectorius" —, e elas nem sempre vêm no mesmo bloco do cabeçalho. Procurar só
    a linha de cima deixa de fora toda magia de nome comprido: eram QUARENTA
    magias não medidas antes desta função, contra quinze da medição original da
    issue. Um instrumento que casa menos que o anterior não é conservador, é pior.

    A busca vai do título mais LONGO para o mais curto, pela mesma razão do
    `audit-bestiary.py`: começando pelo curto, "Luz" casaria antes de "Luz do
    Amanhecer".
    """
    for quantas in (3, 2, 1):
        if i - quantas < 0:
            continue
        junto = ' '.join(linhas[j][1] for j in range(i - quantas, i)).strip()
        k = chave(junto)
        if k in nomes:
            return junto, k, quantas
    return linhas[i - 1][1], chave(linhas[i - 1][1]), 1


def augments_do_corpo(corpo: list[str]) -> list[dict]:
    """Os aprimoramentos de um corpo, com o texto até o próximo aprimoramento."""
    fora: list[dict] = []
    atual = None
    for linha in corpo:
        m = RE_AUGMENT.match(linha)
        if m:
            atual = {
                'pmCost': int(m.group(1)) if m.group(1) is not None else 0,
                'truque': m.group(1) is None,
                'classOnly': (m.group(2) or '').lower() or None,
                'texto': [m.group(3)],
            }
            fora.append(atual)
            continue
        if atual is not None and linha:
            atual['texto'].append(linha)
    for a in fora:
        # O PDF QUEBRA PALAVRA COM HÍFEN no fim da linha: "perma- nente",
        # "mate- rial". Juntar sem desfazer isso produz um texto que não casa
        # com nada e uma leitura que parece defeito de transcrição.
        a['texto'] = re.sub(r'(\w)-\s+(\w)', r'\1\2', ' '.join(a['texto'])).strip()
    return fora


def execucao_do_corpo(corpo: list[str]) -> str:
    for linha in corpo:
        if RE_EXECUCAO.match(linha):
            return linha
    return ''


def confere(spell: dict, lido: dict) -> dict:
    """Compara o catálogo com o livro e devolve o que diverge."""
    doLivro = augments_do_corpo(lido['corpo'])
    doCatalogo = spell.get('augments', [])
    queixas: list[str] = []

    # ÂNCORA 1: círculo e escola. Sem ela, não é esta magia.
    if lido['circulo'] != spell.get('circle'):
        queixas.append(
            f"círculo: livro {lido['circulo']}, catálogo {spell.get('circle')}")
    if chave(lido['escola']) != chave(spell.get('school', '')):
        queixas.append(
            f"escola: livro {lido['escola']!r}, catálogo {spell.get('school')!r}")

    # A CONTAGEM, que é o que a medição da issue mediu.
    if len(doLivro) != len(doCatalogo):
        queixas.append(
            f"aprimoramentos: livro {len(doLivro)}, catálogo {len(doCatalogo)}")

    # O TEXTO, par a par — e SÓ quando as contagens batem.
    #
    # Com contagens diferentes, parear por índice alinha o terceiro do livro com
    # o terceiro do catálogo e inventa uma queixa de custo para cada um depois
    # do que falta. Ruído com cara de achado: quem lê o relatório não distingue
    # "este valor está errado" de "a lista escorregou uma posição".
    if len(doLivro) == len(doCatalogo):
        for i, (livro, cat) in enumerate(zip(doLivro, doCatalogo)):
            if livro['pmCost'] != cat.get('pmCost'):
                queixas.append(
                    f"[{i}] custo: livro +{livro['pmCost']} PM, catálogo +{cat.get('pmCost')} PM")
            umSo = livro['classOnly'], cat.get('classOnly')
            if (umSo[0] is None) != (umSo[1] is None):
                queixas.append(f"[{i}] classOnly: livro {umSo[0]!r}, catálogo {umSo[1]!r}")
    return {
        'id': spell['id'], 'titulo': lido['titulo'], 'pagina': lido['pagina'] - OFFSET,
        'livro': doLivro, 'catalogo': doCatalogo, 'queixas': queixas,
    }


def parecidas(a: str, b: str) -> float:
    """Quanto duas frases dividem de vocabulário, de 0 a 1.

    Jaccard sobre palavras e não distância de edição: o catálogo REESCREVE as
    frases do livro em forma curta — "muda o alcance para curto e o alvo para 1
    objeto" vira "Muda alcance para curto e alvo para 1 objeto" —, então o que
    sobrevive é o vocabulário, não a sequência.
    """
    pa = set(normaliza_frase(a).split())
    pb = set(normaliza_frase(b).split())
    if not pa or not pb:
        return 0.0
    return len(pa & pb) / len(pa | pb)


# Abaixo disto são duas frases diferentes que por acaso dividem "muda o alcance
# para". Acima, é a MESMA frase reescrita. O valor foi calibrado nos três casos
# conhecidos (Âncora Dimensional, Potência Divina, Consagrar), que dão 0,45+.
PISO_DE_PARECENCA = 0.42


def relatorioDeOrfaos(cat: dict, lidas: dict, resultados: list) -> None:
    """Para cada aprimoramento que falta na magia dele, procura no catálogo INTEIRO.

    A pergunta é outra e o achado é outro: um aprimoramento que existe no
    catálogo mas pendurado na magia ERRADA não é transcrição a fazer, é
    transcrição a MOVER — e ele conta duas vezes no relatório normal, como
    falta numa magia e como sobra na vizinha.

    Três casos foram achados à mão antes desta função, e os três eram a magia
    VIZINHA na mesma página. É o que sugere que o deslize foi de importação e
    não de leitura.

    # O TERRENO É SÓ QUEM TEM CONTAGEM DIFERENTE, e isso não é economia

    Rodando sobre as 196 casadas, esta função acusou 105 "a transcrever" contra
    as 44 que a contagem acha — e as 61 de diferença eram RUÍDO: o catálogo
    CONDENSA as frases do livro de propósito, então uma magia com a contagem
    certa tem parecença baixa sem que falte nada. Uma lista de 105 com cara de
    descoberta, que é o que esta ferramenta inteira existe para não produzir.

    Onde a contagem BATE, parecença baixa quer dizer "foi reescrito", e a
    pergunta sobre o texto é outra (o valor está certo?), que é a do relatório
    normal. Aqui a pergunta é só: o que falta, e será que já está em outra magia?

    # O QUE ELE DEVOLVE É PISTA, E NÃO VEREDICTO

    Aprimoramento genérico existe, e é comum: "aumenta o número de alvos em +1"
    aparece igual em várias magias. A Acalmar Animal (p178) FALTA um desses, e o
    candidato que sai com parecença 1,00 é a Tranquilidade — que tem o dela, por
    direito. Dois textos idênticos não são um deslocamento.

    O que separa um do outro é ler as DUAS magias no livro, e só os casos em que
    a magia candidata NÃO devia ter aquilo são deslocamento. Confirmados assim:
    os quatro da Âncora Dimensional na Amarras Etéreas, e o da Potência Divina
    na Palavra Primordial.
    """
    todos = [(sid, i, a['description'])
             for sid, s in cat.items() for i, a in enumerate(s.get('augments', []))]
    achados, semDono = 0, 0
    print('--- APRIMORAMENTOS QUE FALTAM, e onde eles podem estar ---\n')
    for r in resultados:
        if len(r['livro']) <= len(r['catalogo']):
            continue
        doCatalogo = {normaliza_frase(a.get('description', '')) for a in r['catalogo']}
        for a in r['livro']:
            if any(parecidas(a['texto'], d) >= PISO_DE_PARECENCA for d in doCatalogo):
                continue
            candidatos = sorted(
                ((parecidas(a['texto'], desc), sid, i, desc)
                 for sid, i, desc in todos if sid != r['id']),
                reverse=True)
            melhor = candidatos[0] if candidatos else (0.0, '', 0, '')
            marca = 'Truque' if a['truque'] else f"+{a['pmCost']} PM"
            if melhor[0] >= PISO_DE_PARECENCA:
                achados += 1
                print(f"{r['id']} (p{r['pagina']}) {marca}")
                print(f"   ESTÁ EM {melhor[1]}[{melhor[2]}] (parecença {melhor[0]:.2f})")
                print(f"   catálogo: {melhor[3][:110]}")
            else:
                semDono += 1
                print(f"{r['id']} (p{r['pagina']}) {marca}: FALTA MESMO")
                print(f"   livro: {a['texto'][:110]}")
    print(f"\nDESLOCADOS (existem, na magia errada): {achados}")
    print(f"A TRANSCREVER (não existem em lugar nenhum): {semDono}")


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--magia', help='confere UMA magia e mostra o texto do livro')
    p.add_argument('--suspeitas', action='store_true',
                   help='só as magias que divergem')
    p.add_argument('--orfaos', action='store_true',
                   help='procura cada aprimoramento faltante no catálogo INTEIRO')
    args = p.parse_args()

    cat = catalogo()
    nomes = {chave(s['name']): sid for sid, s in cat.items()}
    lidas = magias_do_livro(nomes)

    naoMedidas = sorted(set(cat) - set(lidas))
    resultados = [confere(dict(cat[sid], id=sid), lidas[sid]) for sid in sorted(lidas)]
    divergentes = [r for r in resultados if r['queixas']]

    if args.orfaos:
        relatorioDeOrfaos(cat, lidas, resultados)
        return

    if args.magia:
        alvo = [r for r in resultados if r['id'] == args.magia]
        if not alvo:
            print(f"{args.magia}: NÃO MEDIDA — o cabeçalho não foi achado no capítulo")
            return
        r = alvo[0]
        print(f"=== {r['titulo']} (p{r['pagina']}) — {r['id']}")
        print(f"\nLIVRO ({len(r['livro'])} aprimoramentos):")
        for a in r['livro']:
            marca = 'Truque' if a['truque'] else f"+{a['pmCost']} PM"
            so = f" (Apenas {a['classOnly']})" if a['classOnly'] else ''
            print(f"  {marca}{so}: {a['texto'][:300]}")
        print(f"\nCATÁLOGO ({len(r['catalogo'])} aprimoramentos):")
        for a in r['catalogo']:
            so = f" (classOnly={a['classOnly']})" if a.get('classOnly') else ''
            print(f"  +{a.get('pmCost')} PM{so}: {a.get('description', '')[:300]}")
        if r['queixas']:
            print("\nQUEIXAS:")
            for q in r['queixas']:
                print("  -", q)
        return

    print(f"magias no catálogo:  {len(cat)}")
    print(f"casadas com o livro: {len(lidas)}")
    print(f"NÃO MEDIDAS:         {len(naoMedidas)}")
    print(f"com divergência:     {len(divergentes)}")
    if naoMedidas:
        print("\n--- NÃO MEDIDAS (o cabeçalho não foi achado; NÃO quer dizer corretas) ---")
        for sid in naoMedidas:
            print(f"  {sid}  (livro p{cat[sid].get('bookPage')})")
    print("\n--- DIVERGENTES ---")
    for r in divergentes:
        print(f"\n{r['id']}  (p{r['pagina']})  livro={len(r['livro'])} catálogo={len(r['catalogo'])}")
        for q in r['queixas']:
            print("   -", q)


if __name__ == '__main__':
    main()
