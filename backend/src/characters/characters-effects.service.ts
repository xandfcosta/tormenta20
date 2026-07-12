import { Injectable, NotFoundException } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CharactersService } from './characters.service';

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
