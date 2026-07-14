import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { SPELL_CATALOG } from '@tormenta20/t20-data';
import { PrismaService } from '../prisma/prisma.service';
import { CharactersService } from './characters.service';
import type { ApplyEffectDto } from './dto/character.dto';

/** T20 rest-condition recovery multiplier (livro básico p.20): a night's
 * rest restores PV/PM = level × multiplier, floored. */
export type RestCondition = 'ruim' | 'normal' | 'confortavel' | 'luxuosa';
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
  ) {}

  /**
   * Apply a spell's structured buff to a character as a scoped ActiveEffect.
   * Authorization is a domain rule delegated to `findOne` (owner OR a GM of a
   * campaign the character is in), so a player buffs their own PC and a GM
   * buffs any table member. Re-applying the same buff+scope refreshes the row
   * (upsert on the unique key) instead of erroring. The spell must carry a
   * `buff` block (Phase-1 data) — otherwise it's a rules no-op → 400.
   */
  async applyEffect(
    callerId: number,
    characterId: number,
    dto: ApplyEffectDto,
  ) {
    await this.characters.findOne(callerId, characterId);
    const spell = SPELL_CATALOG[dto.spellId];
    if (!spell?.buff) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Spell "${dto.spellId}" has no applicable buff`,
        fieldErrors: { spellId: ['Magia sem efeito aplicável'] },
      });
    }
    const scope = dto.scope ?? spell.buff.defaultScope;
    const modifiers = JSON.stringify(spell.buff.modifiers);
    await this.prisma.activeEffect.upsert({
      where: {
        characterId_catalogId_scope: {
          characterId,
          catalogId: dto.spellId,
          scope,
        },
      },
      create: {
        characterId,
        source: 'spell',
        catalogId: dto.spellId,
        scope,
        modifiers,
      },
      update: { modifiers, source: 'spell' },
    });
    return this.characters.findOne(callerId, characterId);
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

  async endScene(ownerId: number, characterId: number) {
    await this.characters.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: 'scene' },
    });
    return this.characters.findOne(ownerId, characterId);
  }

  async endDay(ownerId: number, characterId: number) {
    await this.characters.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: { in: ['scene', 'day'] } },
    });
    return this.characters.findOne(ownerId, characterId);
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
