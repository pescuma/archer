<script setup>
import FileCountCard from '@/components/FileCountCard.vue'
import ProjectCountCard from '@/components/ProjectCountCard.vue'
import RepositoryCountCard from '@/components/RepositoryCountCard.vue'
import PersonCountCard from '@/components/PersonCountCard.vue'
import CommitsSeenCard from '@/components/CommitsSeenCard.vue'
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'
import FilesSeenCard from '@/components/FilesSeenCard.vue'
import PeopleSeenCard from '@/components/PeopleSeenCard.vue'
import { onMounted } from 'vue'
</script>

<template>
  <div class="col-sm-3 col-lg-3">
    <FileCountCard />
  </div>
  <div class="col-sm-3 col-lg-3">
    <ProjectCountCard />
  </div>
  <div class="col-sm-3 col-lg-3">
    <RepositoryCountCard />
  </div>
  <div class="col-sm-3 col-lg-3">
    <PersonCountCard />
  </div>

  <div class="col-lg-6">
    <div class="row row-cards">
      <div class="col-12">
        <LinesChangedCard />
      </div>
      <div class="col-12">
        <LinesSurvivedCard />
      </div>
    </div>
  </div>
  <div class="col-lg-6">
    <CommitsSeenCard />
  </div>

  <div class="col-lg-6">
    <FilesSeenCard />
  </div>
  <div class="col-lg-6">
    <PeopleSeenCard />
  </div>
</template>
