#!/usr/bin/env python3
"""Confere as magias contra o livro: a COBERTURA e os APRIMORAMENTOS (ALE-340, ALE-456).

Rodar é ATO DELIBERADO, como o `genoracle` e o `audit-bestiary.py`: a ferramenta
PROPÕE, o diff se revisa contra o PDF, e é a revisão que decide.

    python3 scripts/audit-spells.py                 # cobertura + todas
    python3 scripts/audit-spells.py --cobertura     # só "falta alguma?"
    python3 scripts/audit-spells.py --suspeitas     # só as 32 da ALE-340
    python3 scripts/audit-spells.py --magia luz     # uma, com o texto do livro

Ele NÃO escreve no catálogo. A correção é humana, e a razão está abaixo.

São DUAS perguntas, e por muito tempo só uma tinha quem a fizesse
------------------------------------------------------------------
**"As magias que eu tenho estão certas?"** é o resto deste arquivo, e ele
percorre o CATÁLOGO. **"O livro tem alguma que eu não tenho?"** é a COBERTURA, e
ela percorre o LIVRO — denominador invertido, como as outras fatias da ALE-410.

A diferença não era visível: uma corrida com `0 divergentes` parecia dizer que o
catálogo estava completo, e não dizia nada sobre isso. A épica carregou por
semanas um "199 magias no livro contra 198 no app — falta conferir 1" que
NINGUÉM tinha medido, e nenhuma corrida verde deste arquivo podia desmenti-lo.

Medido (ALE-456): são **198**, e as três contagens fecham — o índice, o título
dos verbetes e o catálogo. **Não falta magia nenhuma**, e o 199 era a premissa.

De onde ele veio não dá para afirmar, e por isso não se afirma aqui. O que se
sabe é que CONTAR ENTRADAS DE ÍNDICE não dá o número de magias: as duas listas
somam 226 entradas (124 arcanas + 102 divinas) e 28 magias estão nas DUAS, o que
leva às 198. Qualquer contagem que não trate essa sobreposição erra, e erra para
cima.

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

# A leitura do PDF é do `t20pdf`, compartilhada com os outros auditores
# (ALE-391).
from t20pdf import (  # noqa: E402
    LIXO, RAIZ, chave, linhas_da_pagina, links_em_negrito, normaliza_frase,
    trechos_com_estilo)

SPELLS = str(RAIZ / 'engine-go/domain/catalog/data/spells.json')
PRIMEIRA, ULTIMA = 184, 217
OFFSET = 6

# ─── A COBERTURA: as duas âncoras que o livro oferece ────────────────────────
#
# O ÍNDICE, p174-177 (PDF 180-183): "Lista de Magias Arcanas" e, depois dela, a
# divina. Cada magia é um LINK EM NEGRITO apontando para o verbete.
INDICE_PRIMEIRA, INDICE_ULTIMA = 180, 183
# O TÍTULO do verbete é `Tormenta20` a 24 pontos — o corpo é `IowanOldStyle` a
# 13, a linha de círculo é 14 e "Descrição das magias" é 39.
TAMANHO_DO_TITULO = 24
FAMILIA_DO_TITULO = 'Tormenta20'
# Título longo quebra em duas linhas ("Abençoar" / "Alimentos"). Junta-se o
# trecho seguinte quando ele está na MESMA coluna e na linha de baixo; a coluna
# entra na conta porque dois títulos de colunas vizinhas na mesma altura não são
# um título — foi esse o erro que fez uma contagem dar 247 (ALE-456).
MESMA_COLUNA = 30.0
LINHA_SEGUINTE = 32.0
LARGURA_DA_COLUNA = 120.0
RE_LISTA = re.compile(r'^(Arcana|Divina|Universal)\s+(\d)\s+\(([^)]+)\)\s*$', re.I)
RE_EXECUCAO = re.compile(r'^Execução:')
RE_AUGMENT = re.compile(r'^(?:\+(\d+)\s*PM(?:\s*\(Apenas\s+([^)]+)\))?|Truque)\s*:\s*(.*)$')

def indiceDoLivro() -> dict:
    """{chave: (nome, página do verbete)} das duas listas de magias (p174-177).

    O link dá o DESTINO de graça, então esta âncora confere o `bookPage` do
    catálogo sem ninguém transcrever página nenhuma.

    Uma magia pode estar nas DUAS listas — são 28 —, e por isso o retorno é por
    nome e não uma contagem: somar as linhas das duas listas dá 231 e conta as
    28 duas vezes. É a explicação mais barata para um "199" que nunca foi
    medido, e foi ela que esta fatia veio conferir (ALE-456).
    """
    fora = {}
    for pagina in range(INDICE_PRIMEIRA, INDICE_ULTIMA + 1):
        for texto, destino in links_em_negrito(pagina):
            nome = texto.rstrip('.').strip()
            if nome:
                fora.setdefault(chave(nome), (nome, destino - OFFSET))
    return fora


def titulosDosVerbetes() -> dict:
    """{chave: (nome, página)} do título de cada verbete (p178-211).

    É a SEGUNDA âncora, e ela é independente da primeira: o índice é uma lista
    de links e isto é o tipo em que o livro desenha o título. As duas têm de dar
    o mesmo conjunto — se uma leitura quebrar, a outra a denuncia, e nenhuma das
    duas precisa de um número escrito à mão para isso.
    """
    fora = {}
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        brutos = [t for t in trechos_com_estilo(pagina)
                  if t.tamanho == TAMANHO_DO_TITULO and FAMILIA_DO_TITULO in t.familia]
        brutos.sort(key=lambda t: (round(t.x / LARGURA_DA_COLUNA), t.y))
        atual, ancora = None, None
        for trecho in brutos:
            emendado = (atual is not None
                        and abs(trecho.x - ancora.x) < MESMA_COLUNA
                        and 0 < trecho.y - ancora.y < LINHA_SEGUINTE)
            if emendado:
                atual = atual + ' ' + trecho.texto
            else:
                if atual:
                    fora.setdefault(chave(atual), (atual, pagina - OFFSET))
                atual = trecho.texto
            ancora = trecho
        if atual:
            fora.setdefault(chave(atual), (atual, pagina - OFFSET))
    return fora


def relatorioDeCobertura(cat: dict) -> int:
    """Quantas magias o LIVRO tem contra quantas entraram. Devolve as falhas.

    Esta é a metade que faltava, e a ausência dela não era visível: o resto do
    auditor percorre o CATÁLOGO e pergunta "cada magia que eu tenho existe no
    livro?". Uma corrida com `0 divergentes` dizia nada sobre uma magia que o
    livro tem e o catálogo não — a pergunta inversa, que é a de COBERTURA, não
    tinha quem a fizesse (ALE-456).
    """
    indice = indiceDoLivro()
    verbetes = titulosDosVerbetes()
    nocat = {chave(m['name']): sid for sid, m in cat.items()}
    falhas = []

    print('══ COBERTURA ' + '═' * 46)
    print(f'  índice do livro (p{INDICE_PRIMEIRA - OFFSET}-{INDICE_ULTIMA - OFFSET}, '
          f'links em negrito): {len(indice)}')
    print(f'  verbetes (p{PRIMEIRA - OFFSET}-{ULTIMA - OFFSET}, título a '
          f'{TAMANHO_DO_TITULO}pt):        {len(verbetes)}')
    print(f'  no catálogo:                                 {len(cat)}')

    # AS DUAS ÂNCORAS PRIMEIRO, UMA CONTRA A OUTRA. Se elas discordam, a leitura
    # está torta e nada do que vem depois vale — é o mesmo "a tabela não fecha"
    # dos auditores de saque, que param antes de encostar no catálogo.
    for k, (nome, _p) in sorted(indice.items()):
        if k not in verbetes:
            falhas.append(f'{nome}: está no índice e não tem título de verbete — '
                          'uma das duas leituras está torta')
    for k, (nome, _p) in sorted(verbetes.items()):
        if k not in indice:
            falhas.append(f'{nome}: tem verbete e não está no índice — '
                          'uma das duas leituras está torta')
    if falhas:
        print('\n  AS DUAS ÂNCORAS DISCORDAM, e por isso o catálogo nem foi conferido:')
        for f in falhas:
            print(f'   ✗ {f}')
        return len(falhas)

    # A PÁGINA SAI DO TÍTULO, e NÃO do destino do link.
    #
    # O link parecia o instrumento melhor — ele vem de graça e não depende de
    # medir tipo. Medido: ele erra em duas das 198. A âncora de destino do PDF
    # aponta para o começo do QUADRO, e quando o verbete nasce no pé de uma
    # coluna o destino cai na página seguinte: a Dificultar Detecção e a
    # Esculpir Sons saem uma página adiante de onde o título está desenhado.
    #
    # Duas falhas em 198 é pouco o bastante para parecer defeito do catálogo, e
    # foi assim que elas chegaram — como três queixas, das quais só UMA era
    # real (a Tranca Arcana, que o catálogo citava na p210 e cujo verbete abre
    # na p209). O link fica valendo para a PRESENÇA, que é o que ele mede bem.
    for k, (nome, _destino) in sorted(indice.items()):
        sid = nocat.get(k)
        if sid is None:
            falhas.append(f'{nome}: o livro tem e o catálogo não')
            continue
        pagina = verbetes[k][1]
        if cat[sid].get('bookPage') != pagina:
            falhas.append(f'{nome}: o catálogo cita p{cat[sid].get("bookPage")} e o '
                          f'título do verbete está desenhado na p{pagina}')
    for k, sid in sorted(nocat.items()):
        if k not in indice:
            falhas.append(f'{cat[sid]["name"]}: está no catálogo e o livro não tem')

    if not falhas:
        print(f'\n  as três fecham em {len(cat)}: nada falta e nada sobra')
    for f in falhas:
        print(f'   ✗ {f}')
    return len(falhas)


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
    p.add_argument('--cobertura', action='store_true',
                   help='só a cobertura: quantas o livro tem × quantas entraram')
    args = p.parse_args()

    cat = catalogo()

    # A COBERTURA VEM PRIMEIRO, e sempre. Ela responde "falta alguma?", e o
    # resto do auditor responde "as que eu tenho estão certas?" — perguntas
    # diferentes, e durante um bom tempo só a segunda tinha quem a fizesse.
    if args.cobertura:
        raise SystemExit(min(1, relatorioDeCobertura(cat)))
    if not args.magia and not args.orfaos:
        relatorioDeCobertura(cat)
        print()
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
