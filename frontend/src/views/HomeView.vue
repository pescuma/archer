<script setup>
import FileCountCard from '@/components/FileCountCard.vue'
import ProjectCountCard from '@/components/ProjectCountCard.vue'
import RepositoryCountCard from '@/components/RepositoryCountCard.vue'
import PersonCountCard from '@/components/PersonCountCard.vue'
</script>

<template>
  <div class="page-body">
    <div class="container-xl">
      <div class="row row-deck row-cards">
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
          <div class="card">
            <div class="card-body">
              <h3 class="card-title">Churn</h3>
              <div id="chart-mentions" class="chart-lg"></div>
            </div>
          </div>
        </div>

        <div class="col-lg-6">
          <div class="card">
            <div class="card-header border-0">
              <div class="card-title">Commits</div>
            </div>
            <div class="position-relative">
              <div class="position-absolute top-0 left-0 px-3 mt-1 w-75">
                <div class="row g-2">
                  <div class="col-auto">
                    <div class="chart-sparkline chart-sparkline-square" id="sparkline-activity"></div>
                  </div>
                </div>
              </div>
              <div id="chart-development-activity"></div>
            </div>
            <div class="card-table table-responsive">
              <table class="table table-vcenter">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Commit</th>
                    <th>Date</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td class="w-1">
                      <span class="avatar avatar-sm" style="background-image: url(./static/avatars/000m.jpg)"></span>
                    </td>
                    <td class="td-truncate">
                      <div class="text-truncate">Fix dart Sass compatibility (#29755)</div>
                    </td>
                    <td class="text-nowrap text-muted">28 Nov 2019</td>
                  </tr>
                  <tr>
                    <td class="w-1">
                      <span class="avatar avatar-sm">JL</span>
                    </td>
                    <td class="td-truncate">
                      <div class="text-truncate">Change deprecated html tags to text decoration classes (#29604)</div>
                    </td>
                    <td class="text-nowrap text-muted">27 Nov 2019</td>
                  </tr>
                  <tr>
                    <td class="w-1">
                      <span class="avatar avatar-sm" style="background-image: url(./static/avatars/002m.jpg)"></span>
                    </td>
                    <td class="td-truncate">
                      <div class="text-truncate">justify-content:between ⇒ justify-content:space-between (#29734)</div>
                    </td>
                    <td class="text-nowrap text-muted">26 Nov 2019</td>
                  </tr>
                  <tr>
                    <td class="w-1">
                      <span class="avatar avatar-sm" style="background-image: url(./static/avatars/003m.jpg)"></span>
                    </td>
                    <td class="td-truncate">
                      <div class="text-truncate">Update change-version.js (#29736)</div>
                    </td>
                    <td class="text-nowrap text-muted">26 Nov 2019</td>
                  </tr>
                  <tr>
                    <td class="w-1">
                      <span class="avatar avatar-sm" style="background-image: url(./static/avatars/000f.jpg)"></span>
                    </td>
                    <td class="td-truncate">
                      <div class="text-truncate">Regenerate package-lock.json (#29730)</div>
                    </td>
                    <td class="text-nowrap text-muted">25 Nov 2019</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
