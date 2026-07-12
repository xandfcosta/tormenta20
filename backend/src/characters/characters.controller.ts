import {
  Body,
  Controller,
  Delete,
  Get,
  Param,
  ParseIntPipe,
  Patch,
  Post,
  UseGuards,
} from '@nestjs/common';
import { CurrentUser } from '../auth/current-user.decorator';
import type { AuthUser } from '../auth/auth-user.type';
import { JwtAuthGuard } from '../auth/jwt-auth.guard';
import { CharactersService } from './characters.service';
import { CharacterItemsService } from './characters-items.service';
import { CharacterEffectsService } from './characters-effects.service';
import { CharactersSpellsService } from './characters-spells.service';
import { CampaignMembersService } from '../campaign-members/campaign-members.service';
import {
  ConsumeItemDto,
  CreateCharacterDto,
  CreateExpertiseDto,
  CreateItemDto,
  UpdateAbilityChoicesDto,
  UpdateClassLevelDto,
  UpdateExpertiseDto,
  UpdateItemDto,
  UpdateLevelDto,
  UpdateProficienciesDto,
  UpdateVitalsDto,
} from './dto/character.dto';
import {
  CastSpellDto,
  LearnSpellDto,
  SetSpellPreparedDto,
} from './dto/character-spell.dto';
import {
  CLASSES,
  EXPERTISES,
  GODS,
  ORIGINS,
  RACES,
  SIZES,
} from './t20-constants';

@Controller('characters')
export class CharactersController {
  constructor(
    private readonly characters: CharactersService,
    private readonly members: CampaignMembersService,
    private readonly spells: CharactersSpellsService,
    private readonly items: CharacterItemsService,
    private readonly effects: CharacterEffectsService,
  ) {}

  @Get('options')
  options() {
    return {
      races: RACES,
      classes: CLASSES,
      origins: ORIGINS,
      gods: GODS,
      sizes: SIZES,
      expertises: EXPERTISES,
    };
  }

  @UseGuards(JwtAuthGuard)
  @Get()
  list(@CurrentUser() user: AuthUser) {
    return this.characters.list(user.id);
  }

  @UseGuards(JwtAuthGuard)
  @Get(':id')
  findOne(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.characters.findOne(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Get(':id/sheet')
  findSheet(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.characters.findOneWithComputed(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Get(':id/campaigns')
  listCampaigns(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.members.listForCharacter(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Post()
  create(@CurrentUser() user: AuthUser, @Body() dto: CreateCharacterDto) {
    return this.characters.create(user.id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/vitals')
  updateVitals(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateVitalsDto,
  ) {
    return this.characters.updateVitals(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/expertises')
  updateExpertise(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateExpertiseDto,
  ) {
    return this.characters.updateExpertise(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/expertises')
  addCustomExpertise(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: CreateExpertiseDto,
  ) {
    return this.characters.addCustomExpertise(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/expertises/:name')
  deleteExpertise(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('name') name: string,
  ) {
    return this.characters.deleteExpertise(user.id, id, decodeURIComponent(name));
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/items')
  addItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: CreateItemDto,
  ) {
    return this.items.addItem(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/items/:itemId')
  updateItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
    @Body() dto: UpdateItemDto,
  ) {
    return this.items.updateItem(user.id, id, itemId, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/items/:itemId')
  deleteItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
  ) {
    return this.items.deleteItem(user.id, id, itemId);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/items/:itemId/consume')
  consumeItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
    @Body() dto: ConsumeItemDto,
  ) {
    return this.items.consumeItem(user.id, id, itemId, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/active-effects/:effectId')
  removeActiveEffect(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('effectId', ParseIntPipe) effectId: number,
  ) {
    return this.effects.removeActiveEffect(user.id, id, effectId);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/end-scene')
  endScene(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.effects.endScene(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/end-day')
  endDay(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.effects.endDay(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/proficiencies')
  updateProficiencies(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateProficienciesDto,
  ) {
    return this.characters.updateProficiencies(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/level')
  updateLevel(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateLevelDto,
  ) {
    return this.characters.updateLevel(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/abilities')
  updateAbilityChoices(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateAbilityChoicesDto,
  ) {
    return this.characters.updateAbilityChoices(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/classes/level')
  updateClassLevel(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateClassLevelDto,
  ) {
    return this.characters.updateClassLevel(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/spells')
  learnSpell(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: LearnSpellDto,
  ) {
    return this.spells.learnSpell(user.id, id, dto.catalogSpellId);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/spells/:catalogSpellId')
  unlearnSpell(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('catalogSpellId') catalogSpellId: string,
  ) {
    return this.spells.unlearnSpell(user.id, id, catalogSpellId);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/spells/:catalogSpellId/prepared')
  setSpellPrepared(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('catalogSpellId') catalogSpellId: string,
    @Body() dto: SetSpellPreparedDto,
  ) {
    return this.spells.setSpellPrepared(
      user.id,
      id,
      catalogSpellId,
      dto.prepared,
    );
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/spells/:catalogSpellId/cast')
  castSpell(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('catalogSpellId') catalogSpellId: string,
    @Body() dto: CastSpellDto,
  ) {
    return this.spells.castSpell(user.id, id, catalogSpellId, dto.augments ?? []);
  }
}
