import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import {
  SPELL_CATALOG,
  getActivation,
  isValid,
  validateApplyBuff,
} from '@tormenta20/t20-data';
import type { ActivationGrant, Modifier } from '@tormenta20/t20-data';
import { PrismaService } from '../prisma/prisma.service';
import { computeSheetForRow } from './character-sheet.mapper';
import {
  ACTIVE_EFFECT_SELECT,
  CharacterTempHpService,
} from './character-temp-hp.service';
import { CharactersService } from './characters.service';
import type {
  AdjustActiveEffectDto,
  ApplyEffectDto,
} from './dto/character.dto';

/** T20 rest-condition recovery multiplier (livro básico p.20): a night's
 * rest restores PV/PM = level × multiplier, floored. */
export type RestCondition = 'ruim' | 'normal' | 'confortavel' | 'luxuosa';

/** Delta for end-scene/end-day: the effect scopes the client should drop from
 *  its cached character (it already holds the rows). */
export type EffectsClearedResult = { clearedScopes: ('scene' | 'day')[] };
const REST_MULTIPLIER: Record<RestCondition, number> = {
  ruim: 0.5,
  normal: 1,
  confortavel: 2,
  luxuosa: 3,
};

/**
 * Active-effects + rest slice of the Character aggregate — remove a
 * single effect, end-scene / end-day effect expiry, and the night-rest
 * PV/PM recovery rule. Split out of CharactersService for SRP; the
 * realtime gateway triggers rest here. Ownership still delegates to
 * CharactersService.findOne.
 */
@Injectable()
export class CharacterEffectsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly characters: CharactersService,
    private readonly tempHp: CharacterTempHpService,
  ) {}

  /**
   * Apply a lasting effect as a scoped ActiveEffect from ONE of three
   * sources: a spell's structured buff (`spellId`), a power grant
   * (`powerId` → `getActivation(id).grant`, Fase 4) or a GM-entered manual
   * temp-PV pool (`manualTempHp`, F3 — 0 clears it). Authorization is a
   * domain rule delegated to `findOne` (owner OR a GM of a campaign the
   * character is in). Re-applying the same source+scope refreshes the row
   * (upsert on the unique key) instead of erroring; temp-PV pools obey
   * vale-o-maior (p256) via CharacterTempHpService.
   */
  async applyEffect(
    callerId: number,
    characterId: number,
    dto: ApplyEffectDto,
  ) {
    const character = await this.characters.findOne(callerId, characterId);
    if (dto.manualTempHp !== undefined) {
      return this.tempHp.setManualPool(characterId, dto.manualTempHp, dto.scope);
    }
    if (dto.powerId) {
      return this.applyPowerGrant(character, characterId, dto);
    }
    if (!dto.spellId) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `applyEffect requires spellId, powerId or manualTempHp — got ${JSON.stringify(dto)}`,
        fieldErrors: { spellId: ['Informe uma magia, um poder ou PV temporários'] },
      });
    }
    return this.applySpellBuff(characterId, dto.spellId, dto.scope);
  }

  /**
   * Spell path: the spell must carry a `buff` block (Phase-1 data) —
   * otherwise it's a rules no-op → 400. Same shared check the frontend uses.
   */
  private applySpellBuff(
    characterId: number,
    spellId: string,
    scopeOverride?: 'scene' | 'day',
  ) {
    if (!isValid(validateApplyBuff(spellId))) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Spell "${spellId}" has no applicable buff`,
        fieldErrors: { spellId: ['Magia sem efeito aplicável'] },
      });
    }
    // validateApplyBuff guarantees the catalog entry + buff exist.
    const buff = SPELL_CATALOG[spellId].buff!;
    const scope = scopeOverride ?? buff.defaultScope;
    return this.upsertEffect(
      characterId,
      'spell',
      spellId,
      scope,
      JSON.stringify(buff.modifiers),
    );
  }

  /**
   * Power path (Fase 4): resolve the hand-authored grant. `temp-hp` grants
   * compute the pool server-side — nível + atributo FINAL do grant (Alma de
   * Bronze p41: nível + Força) via the same computeSheetForRow the read-heal
   * uses, so racial mods and boosts count — and go through the vale-o-maior
   * pool path (F1, p256). `active-effect` grants persist their modifiers
   * verbatim.
   */
  private applyPowerGrant(
    character: Parameters<typeof computeSheetForRow>[0] & { level: number },
    characterId: number,
    dto: ApplyEffectDto,
  ) {
    const grant = this.powerGrantOf(dto.powerId!);
    const scope = dto.scope ?? grant.scope;
    if (grant.kind === 'temp-hp') {
      const { amount } = this.tempHpModifier(character, grant);
      return this.tempHp.applyPool(characterId, 'power', dto.powerId!, scope, amount);
    }
    const modifiers = JSON.stringify(grant.modifiers);
    return this.upsertEffect(characterId, 'power', dto.powerId!, scope, modifiers);
  }

  private powerGrantOf(powerId: string): ActivationGrant {
    const spec = getActivation(powerId);
    if (!spec) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Power "${powerId}" not found in the activation registry`,
        fieldErrors: { powerId: ['Poder desconhecido'] },
      });
    }
    if (!spec.grant) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Power "${powerId}" has no applicable grant`,
        fieldErrors: { powerId: ['Poder sem efeito aplicável'] },
      });
    }
    return spec.grant;
  }

  private tempHpModifier(
    character: Parameters<typeof computeSheetForRow>[0] & { level: number },
    grant: Extract<ActivationGrant, { kind: 'temp-hp' }>,
  ): Modifier {
    const sheet = computeSheetForRow(character);
    const amount = character.level + sheet.attributes[grant.attribute].total;
    return {
      target: { k: 'tempHp' },
      amount,
      bonusType: 'untyped',
      note: 'PV temporários',
    };
  }

  /** Return just the upserted ActiveEffect row (a delta) — the client merges
   *  it into the cached character rather than re-reading the whole aggregate. */
  private upsertEffect(
    characterId: number,
    source: 'spell' | 'power',
    catalogId: string,
    scope: 'scene' | 'day',
    modifiers: string,
  ) {
    return this.prisma.activeEffect.upsert({
      where: {
        characterId_catalogId_scope: { characterId, catalogId, scope },
      },
      create: { characterId, source, catalogId, scope, modifiers },
      update: { modifiers, source },
      select: ACTIVE_EFFECT_SELECT,
    });
  }

  /**
   * Debit (or top up) a persisted tempHp pool effect (Fase 4 — damage is
   * routed temp-first by the HUD). Amount floors at 0; a drained pool row is
   * deleted and reported as `{ removed: true }` so the client drops it.
   *
   * @example effects.adjustActiveEffect(userId, charId, effectId, { tempHpDelta: -3 })
   */
  async adjustActiveEffect(
    callerId: number,
    characterId: number,
    effectId: number,
    dto: AdjustActiveEffectDto,
  ) {
    await this.characters.findOne(callerId, characterId);
    const effect = await this.prisma.activeEffect.findUnique({
      where: { id: effectId },
      select: { ...ACTIVE_EFFECT_SELECT, characterId: true },
    });
    if (!effect || effect.characterId !== characterId) {
      throw new NotFoundException(
        `Active effect ${effectId} not found for character ${characterId}`,
      );
    }
    const modifiers = this.tempHpModifiersOf(effectId, effect);
    const idx = modifiers.findIndex((m) => m.target.k === 'tempHp');
    const amount = Math.max(0, modifiers[idx].amount + dto.tempHpDelta);
    if (amount === 0) {
      await this.prisma.activeEffect.delete({ where: { id: effectId } });
      return { removed: true as const, id: effectId };
    }
    modifiers[idx] = { ...modifiers[idx], amount };
    return this.prisma.activeEffect.update({
      where: { id: effectId },
      data: { modifiers: JSON.stringify(modifiers) },
      select: ACTIVE_EFFECT_SELECT,
    });
  }

  /** Parse the row's modifiers, insisting on a tempHp target (400 otherwise). */
  private tempHpModifiersOf(
    effectId: number,
    effect: { catalogId: string; modifiers: string },
  ): Modifier[] {
    let parsed: unknown;
    try {
      parsed = JSON.parse(effect.modifiers);
    } catch {
      parsed = null;
    }
    const modifiers = Array.isArray(parsed) ? (parsed as Modifier[]) : [];
    if (!modifiers.some((m) => m?.target?.k === 'tempHp')) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Active effect ${effectId} ("${effect.catalogId}") has no tempHp modifier to adjust`,
        fieldErrors: { tempHpDelta: ['Efeito sem PV temporários'] },
      });
    }
    return modifiers;
  }

  async removeActiveEffect(
    ownerId: number,
    characterId: number,
    effectId: number,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const effect = await this.prisma.activeEffect.findUnique({
      where: { id: effectId },
      select: { id: true, characterId: true },
    });
    if (!effect || effect.characterId !== characterId) {
      throw new NotFoundException(`Active effect ${effectId} not found`);
    }
    await this.prisma.activeEffect.delete({ where: { id: effectId } });
    return { id: effectId };
  }

  // Delta: which scopes were cleared. The client drops the matching cached
  // effects itself (it already holds them), so no full-Character re-read.
  async endScene(
    ownerId: number,
    characterId: number,
  ): Promise<EffectsClearedResult> {
    await this.characters.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: 'scene' },
    });
    return { clearedScopes: ['scene'] };
  }

  async endDay(
    ownerId: number,
    characterId: number,
  ): Promise<EffectsClearedResult> {
    await this.characters.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: { in: ['scene', 'day'] } },
    });
    return { clearedScopes: ['scene', 'day'] };
  }

  /**
   * T20 night's-rest recovery (livro básico p.20): restore PV and PM by
   * `floor(level × condition multiplier)`, clamped to max. The heal rule
   * lives here on the Character aggregate — the realtime gateway only
   * triggers it. Returns the new current values so the caller (session
   * tracker) can mirror them onto the live entry.
   *
   * @example characters.restVitals(gmUserId, charId, 'confortavel')
   */
  async restVitals(
    userId: number,
    characterId: number,
    condition: RestCondition,
  ): Promise<{ hpCurrent: number; mpCurrent: number }> {
    const character = await this.characters.findOne(userId, characterId);
    const gain = Math.floor(character.level * REST_MULTIPLIER[condition]);
    const hpCurrent = Math.min(character.hpMax, character.hpCurrent + gain);
    const mpCurrent = Math.min(character.mpMax, character.mpCurrent + gain);
    await this.prisma.character.update({
      where: { id: characterId },
      data: { hpCurrent, mpCurrent },
    });
    return { hpCurrent, mpCurrent };
  }
}
