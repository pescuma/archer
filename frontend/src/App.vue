<script setup>
import { reactive } from 'vue'
import { RouterView } from 'vue-router'
import { IconMoon, IconSun } from '@tabler/icons-vue'
import NavbarRouterLink from '@/components/NavbarRouterLink.vue'

const THEME_STORAGE_KEY = 'theme'
const LIGHT = 'light'
const DARK = 'dark'

const data = reactive({
  theme: localStorage.getItem(THEME_STORAGE_KEY) || LIGHT,
})

setTheme()

function setTheme() {
  if (data.theme === DARK) {
    document.body.setAttribute('data-bs-theme', DARK)
  } else {
    document.body.removeAttribute('data-bs-theme')
  }
}

function toggleTheme() {
  if (data.theme === LIGHT) data.theme = DARK
  else data.theme = LIGHT

  localStorage.setItem(THEME_STORAGE_KEY, data.theme)

  setTheme()
}
</script>

<template>
  <div class="page">
    <!-- Navbar -->
    <div class="sticky-top">
      <header class="navbar navbar-expand-md d-print-none">
        <div class="container-xl">
          <h1 class="navbar-brand navbar-brand-autodark d-none-navbar-horizontal pe-0 pe-md-3">archer</h1>

          <div class="navbar-nav flex-row order-md-last">
            <div class="d-none d-md-flex">
              <a
                href="#"
                class="nav-link px-0"
                :title="data.theme === LIGHT ? 'Enable dark mode' : 'Enable light mode'"
                data-bs-toggle="tooltip"
                data-bs-placement="bottom"
                @click.prevent="toggleTheme"
              >
                <IconMoon class="icon" v-if="data.theme === LIGHT" />
                <IconSun class="icon" v-else />
              </a>
            </div>
          </div>

          <div class="collapse navbar-collapse" id="navbar-menu">
            <div class="d-flex flex-column flex-md-row flex-fill align-items-stretch align-items-md-center">
              <ul class="navbar-nav">
                <NavbarRouterLink to="/" icon="home" text="Home" />
                <NavbarRouterLink to="/files" icon="files" text="Files" />
                <NavbarRouterLink to="/projects" icon="folders" text="Projects" />
                <NavbarRouterLink to="/repos" icon="database" text="Repositories" />
                <NavbarRouterLink to="/people" icon="users" text="People" />
              </ul>
            </div>
          </div>
        </div>
      </header>
    </div>

    <div class="page-wrapper">
      <RouterView />

      <footer class="footer footer-transparent d-print-none">
        <div class="container-xl">
          <div class="row text-center align-items-center flex-row">
            <div class="col-12 col-lg-auto">archer v0.2</div>
            <div class="col-lg-auto ms-lg-auto">
              <a href="https://github.com/pescuma/archer" target="_blank" class="link-secondary" rel="noopener">Source code</a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  </div>
</template>

<style scoped></style>