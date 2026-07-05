import { Module } from '@nestjs/common';
import { CampaignMembersModule } from '../campaign-members/campaign-members.module';
import { CharactersController } from './characters.controller';
import { CharactersService } from './characters.service';

@Module({
  imports: [CampaignMembersModule],
  controllers: [CharactersController],
  providers: [CharactersService],
})
export class CharactersModule {}
