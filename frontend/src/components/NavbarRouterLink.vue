<script setup>
import { RouterLink } from 'vue-router'

const props = defineProps({
  icon: String,
  to: String,
  text: String,
})
</script>

<template>
  <RouterLink :to="props.to" custom v-slot="{ isActive, href, navigate }">
    <li :class="isActive ? 'nav-item active' : 'nav-item'">
      <a class="nav-link" :href="href" @click="navigate">
        <span class="nav-link-icon d-md-none d-lg-inline-block">
          <i :class="'icon ti ti-' + props.icon"></i>
        </span>
        <span class="nav-link-title">{{ props.text }}</span>
      </a>
    </li>
  </RouterLink>
</template>

<style scoped></style>
