<script setup>
import _ from 'lodash'
import cytoscape from 'cytoscape'
import { nextTick, onMounted, ref, watch } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import { filters } from '@/utils/filters'

let card = ref(null)
let cy

function refresh() {
  let f = filters.toQueryString()
  card.value.request(`/api/arch?${f}`, async function (response) {
    for (let i = 0; i < 3; i++) {
      await nextTick()
    }

    const els = []

    const roots = _.chain(response)
      .filter((r) => r.root)
      .map((r) => r.root)
      .uniq()
      .value()

    for (const r of roots) {
      els.push({
        data: {
          id: r,
          name: r,
        },
      })
    }

    for (const r of response) {
      if (!r.name) continue

      r.parent = r.root
      els.push({ data: r })
    }
    for (const r of response) {
      if (!r.source) continue

      els.push({ data: r })
    }

    cy = cytoscape({
      container: document.getElementById('cy'),
      elements: els,
      style: [
        {
          selector: 'node',
          style: {
            'background-opacity': 0,
            'border-color': tabler.getColor('azure'),
            'border-width': 2,
            label: 'data(name)',
            color: tabler.getColor('body-color'),
            'text-valign': 'bottom',
            'text-margin-y': '2px',
            'text-outline-color': tabler.getColor('bg-surface'),
            'text-outline-width': '1px',
          },
        },
        {
          selector: ':parent',
          style: {
            'background-color': tabler.getColor('azure'),
            'background-opacity': 0.1,
            'border-color': tabler.getColor('azure'),
            'border-width': 5,
            label: 'data(name)',
            color: tabler.getColor('body-color'),
            'font-size': 36,
            'text-valign': 'top',
            'text-margin-y': '0',
            'text-outline-color': tabler.getColor('bg-surface'),
            'text-outline-width': '1px',
          },
        },
        {
          selector: 'edge',
          style: {
            width: 3,
            'line-color': _.memoize((edge) => {
              let s = edge.sourceEndpoint()
              let d = edge.targetEndpoint()
              if (s.y > d.y) {
                return tabler.getColor('red')
              } else {
                return tabler.getColor('gray-400')
              }
            }),
            'target-arrow-color': _.memoize((edge) => {
              let s = edge.sourceEndpoint()
              let d = edge.targetEndpoint()
              if (s.y > d.y) {
                return tabler.getColor('red')
              } else {
                return tabler.getColor('gray-400')
              }
            }),
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
          },
        },
      ],
      layout: {
        name: 'dagre',
        nodeDimensionsIncludeLabels: true,
      },
    })

    cy.nodeHtmlLabel([
      {
        query: 'node',
        halign: 'center',
        valign: 'center',
        halignBox: 'center',
        valignBox: 'center',
        tpl: (data) => {
          return `<i class="ti ti-${data.type}"></i>`
        },
      },
    ])
  })
}

onMounted(refresh)

watch(() => filters.data, refresh, { deep: true })
</script>

<template>
  <CardWithPlaceholder ref="card" type="table">
    <div class="card-header">
      <h3 class="card-title">Architecture</h3>
    </div>

    <div class="card-body">
      <div id="cy" style="width: 100%; height: 450px; display: block"></div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
