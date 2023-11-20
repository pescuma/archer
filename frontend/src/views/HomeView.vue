<script setup>
import FileCountCard from '@/components/FileCountCard.vue'
import ProjectCountCard from '@/components/ProjectCountCard.vue'
import RepositoryCountCard from '@/components/RepositoryCountCard.vue'
import PersonCountCard from '@/components/PersonCountCard.vue'
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'
import CommitsTable from '../components/CommitsTable.vue'
import FilesTable from '@/components/FilesTable.vue'
import PeopleTable from '@/components/PeopleTable.vue'
import ProjectsTable from '@/components/ProjectsTable.vue'
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
    <FilesTable size="sm" />
  </div>
  <div class="col-lg-6">
    <ProjectsTable size="sm" />
  </div>

  <div class="col-lg-6">
    <LinesChangedCard />
  </div>
  <div class="col-lg-6">
    <CommitsTable size="sm" />
  </div>

  <div class="col-lg-6">
    <LinesSurvivedCard />
  </div>
  <div class="col-lg-6">
    <PeopleTable size="sm" />
  </div>
</template>
