import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import {
  CreateExpertiseDto,
  UpdateExpertiseDto,
} from './dto/character.dto';
import { EXPERTISE_NAMES } from './t20-constants';
import { CharactersService } from './characters.service';

/**
 * Expertise (perícia) slice of the Character aggregate — add a custom
 * expertise, delete a custom one, toggle attribute/trained. Split out of
 * CharactersService for SRP; ownership still delegates to
 * CharactersService.findOne, and builtin names stay protected.
 */
@Injectable()
export class CharacterExpertisesService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly characters: CharactersService,
  ) {}

  async addCustomExpertise(
    ownerId: number,
    characterId: number,
    dto: CreateExpertiseDto,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const trimmed = dto.name.trim();
    const fieldErrors: Record<string, string[]> = {};
    if (!trimmed) {
      fieldErrors.name = ['Name is required'];
    } else if ((EXPERTISE_NAMES as readonly string[]).includes(trimmed)) {
      fieldErrors.name = [`"${trimmed}" is a builtin expertise — pick another name`];
    } else {
      const exists = await this.prisma.characterExpertise.findUnique({
        where: { characterId_name: { characterId, name: trimmed } },
        select: { id: true },
      });
      if (exists) {
        fieldErrors.name = [`Expertise "${trimmed}" already exists`];
      }
    }
    if (Object.keys(fieldErrors).length > 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors,
      });
    }
    return this.prisma.characterExpertise.create({
      data: {
        characterId,
        name: trimmed,
        attribute: dto.attribute,
        trained: true,
        custom: true,
      },
      select: {
        name: true,
        attribute: true,
        trained: true,
        custom: true,
      },
    });
  }

  async deleteExpertise(
    ownerId: number,
    characterId: number,
    name: string,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const row = await this.prisma.characterExpertise.findUnique({
      where: { characterId_name: { characterId, name } },
      select: { id: true, custom: true },
    });
    if (!row) {
      throw new NotFoundException(`Expertise "${name}" not found`);
    }
    if (!row.custom) {
      throw new BadRequestException(
        `Expertise "${name}" is builtin and cannot be removed`,
      );
    }
    await this.prisma.characterExpertise.delete({ where: { id: row.id } });
    return { name };
  }

  async updateExpertise(
    ownerId: number,
    characterId: number,
    dto: UpdateExpertiseDto,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const data: { attribute?: string; trained?: boolean } = {};
    if (dto.attribute !== undefined) data.attribute = dto.attribute;
    if (dto.trained !== undefined) data.trained = dto.trained;
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.characterExpertise.update({
      where: { characterId_name: { characterId, name: dto.name } },
      data,
      select: {
        name: true,
        attribute: true,
        trained: true,
        custom: true,
      },
    });
  }
}
