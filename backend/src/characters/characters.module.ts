import { Module } from '@nestjs/common';
import { CampaignMembersModule } from '../campaign-members/campaign-members.module';
import { CharactersController } from './characters.controller';
import { CharactersService } from './characters.service';
import { CharactersSpellsService } from './characters-spells.service';

@Module({
  imports: [CampaignMembersModule],
  controllers: [CharactersController],
  providers: [CharactersService, CharactersSpellsService],
})
export class CharactersModule {}
