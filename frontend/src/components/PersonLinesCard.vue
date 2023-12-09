<script setup>
import _ from 'lodash'
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'
import {computed} from "vue";

const props = defineProps({
  person: Object,
})

const name = computed(()=>{
  let names = []
  names.push(props.person.name)
  for (let name of props.person.names) {
      names.push(name)
  }
  return _.join(names, ', ')
})
</script>

<template>
  <div class="card card-sm">
    <div class="card-header">
      <h3 class="card-title">{{ name }}</h3>
    </div>

    <div class="card-body">
      <div class="row">
        <div class="col-lg-6">
          <LinesChangedCard :person-id="person.id" />
        </div>
        <div class="col-lg-6">
          <LinesSurvivedCard :person-id="person.id" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
