<script setup>
import { onMounted, reactive, ref } from 'vue'
import { IconFolders } from '@tabler/icons-vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const card = ref(null)

const data = reactive({
  text: '',
  details: '',
})

onMounted(function () {
  card.value.request('/api/stats/count/projects', function (response) {
    data.text = (response.total - response.external).toLocaleString() + ' Projects'
    data.details = response.external.toLocaleString() + ' external'
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="count">
    <div class="row align-items-center">
      <div class="col-auto">
        <span class="bg-azure text-white avatar">
          <IconFolders />
        </span>
      </div>
      <div class="col">
        <div class="font-weight-medium">{{ data.text }}</div>
        <div class="text-muted">{{ data.details }}</div>
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
