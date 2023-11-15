<script setup>
import { onMounted, reactive, ref } from 'vue'
import { IconUsers } from '@tabler/icons-vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const card = ref(null)

const data = reactive({
  text: '',
  details: '',
})

onMounted(function () {
  card.value.request('/api/stats/count/people', function (response) {
    data.text = response.total.toLocaleString() + ' People'
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="count">
    <div class="card-body">
      <div class="row align-items-center">
        <div class="col-auto">
          <span class="bg-purple text-white avatar">
            <IconUsers />
          </span>
        </div>
        <div class="col">
          <div class="font-weight-medium">{{ data.text }}</div>
          <div class="text-muted">{{ data.details }}</div>
        </div>
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
