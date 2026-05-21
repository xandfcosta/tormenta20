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
import type { AuthUser } from '../auth/auth.service';
import { JwtAuthGuard } from '../auth/jwt-auth.guard';
import { CharactersService } from './characters.service';
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
  CLASSES,
  EXPERTISES,
  GODS,
  ORIGINS,
  RACES,
  SIZES,
} from './t20-constants';

@Controller('characters')
export class CharactersController {
  constructor(private readonly characters: CharactersService) {}

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
    return this.characters.addItem(user.id, id, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Patch(':id/items/:itemId')
  updateItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
    @Body() dto: UpdateItemDto,
  ) {
    return this.characters.updateItem(user.id, id, itemId, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/items/:itemId')
  deleteItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
  ) {
    return this.characters.deleteItem(user.id, id, itemId);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/items/:itemId/consume')
  consumeItem(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('itemId', ParseIntPipe) itemId: number,
    @Body() dto: ConsumeItemDto,
  ) {
    return this.characters.consumeItem(user.id, id, itemId, dto);
  }

  @UseGuards(JwtAuthGuard)
  @Delete(':id/active-effects/:effectId')
  removeActiveEffect(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
    @Param('effectId', ParseIntPipe) effectId: number,
  ) {
    return this.characters.removeActiveEffect(user.id, id, effectId);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/end-scene')
  endScene(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.characters.endScene(user.id, id);
  }

  @UseGuards(JwtAuthGuard)
  @Post(':id/end-day')
  endDay(
    @CurrentUser() user: AuthUser,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.characters.endDay(user.id, id);
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
}
