<script setup>
import _ from 'lodash'
import { RouterView } from 'vue-router'
import NavbarRouterLink from '@/components/NavbarRouterLink.vue'
import { api } from '@/utils/api'
import { filters } from '@/utils/filters'
import { theme } from '@/utils/theme'
import { computed, reactive } from 'vue'

window.api = api

theme.init()

const fui = reactive({
  filters: filters.data,
  patch: filters.patch,
  clear: filters.clear,
  visible: false,
})

fui.count = computed(() => {
  let result = 0
  for (let f in filters.data) {
    if (filters.data[f]) result++
  }
  return result
})
</script>

<template>
  <div :class="'page ' + (api.loading ? 'cursor-loading' : '')">
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
                :title="theme.isDark() ? 'Enable dark mode' : 'Enable light mode'"
                data-bs-toggle="tooltip"
                data-bs-placement="bottom"
                @click.prevent="theme.toggle"
              >
                <icon-sun class="icon" v-if="theme.isDark()" />
                <icon-moon class="icon" v-else />
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

          <div v-if="api.loading" style="float: right; position: absolute; right: 10px">
            <div class="spinner-border text-blue" role="status"></div>
          </div>
          <div v-if="!api.loading && api.errors" style="float: right; position: absolute; right: 10px" class="text-red" :title="api.errors">
            <icon-alert-triangle class="icon" />
          </div>
        </div>
      </header>
    </div>

    <div class="page-wrapper">
      <div class="page-body">
        <div class="container-xl">
          <div class="row row-deck row-cards">
            <div class="col-12">
              <form class="card">
                <div class="card-header">
                  <h3 class="card-title">
                    Filters
                    <span class="badge ms-1" v-if="fui.count > 0">{{ fui.count }}</span>
                  </h3>
                  <div class="card-actions btn-actions">
                    <a href="#" class="btn-action" v-if="fui.count > 0" @click.prevent="fui.clear()">
                      <icon-trash class="icon" />
                    </a>
                    <a href="#" class="btn-action" @click.prevent="fui.visible = !fui.visible">
                      <icon-chevron-up class="icon" v-if="fui.visible" />
                      <icon-chevron-down class="icon" v-else />
                    </a>
                  </div>
                </div>

                <div class="card-body" v-if="fui.visible">
                  <div class="row">
                    <div class="col-3">
                      <div class="mb-3">
                        <label class="form-label">File</label>
                        <div class="input-group mb-2">
                          <input type="text" class="form-control" v-model.lazy="fui.filters.file" />
                          <a href="#" class="btn btn-icon text-muted" @click.prevent="fui.filters.file = ''">
                            <icon-trash class="icon" />
                          </a>
                        </div>
                      </div>
                    </div>

                    <div class="col-3">
                      <div class="mb-3">
                        <label class="form-label">Project</label>
                        <div class="input-group mb-2">
                          <input type="text" class="form-control" v-model.lazy="fui.filters.proj" />
                          <a href="#" class="btn btn-icon text-muted" @click.prevent="fui.filters.proj = ''">
                            <icon-trash class="icon" />
                          </a>
                        </div>
                      </div>
                    </div>

                    <div class="col-3">
                      <div class="mb-3">
                        <label class="form-label">Repository</label>
                        <div class="input-group mb-2">
                          <input type="text" class="form-control" v-model.lazy="fui.filters.repo" />
                          <a href="#" class="btn btn-icon text-muted" @click.prevent="fui.filters.repo = ''">
                            <icon-trash class="icon" />
                          </a>
                        </div>
                      </div>
                    </div>

                    <div class="col-3">
                      <div class="mb-3">
                        <label class="form-label">Person</label>
                        <div class="input-group mb-2">
                          <input type="text" class="form-control" v-model.lazy="fui.filters.person" />
                          <a href="#" class="btn btn-icon text-muted" @click.prevent="fui.filters.person = ''">
                            <icon-trash class="icon" />
                          </a>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </form>
            </div>

            <RouterView />
          </div>
        </div>
      </div>

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
