import { reactive } from 'vue'

export const filters = reactive({
  data: {
    file: null,
    proj: null,
    repo: null,
    person: null,
  },
})

filters.clear = () => {
  for (let f in filters.data) {
    filters.data[f] = null
  }
}

filters.patch = (fs) => {
  for (let f in fs) {
    filters.data[f] = fs[f]
  }
}

filters.toQueryString = (mapping) => {
  let fs = ''
  for (let f in filters.data) {
    let v = filters.data[f]
    if (!v) continue

    v = (v + '').replace(/\s+/g, ' ').trim().toLowerCase()

    if (mapping && mapping[f]) f = mapping[f]

    if (fs) fs += '&'
    fs += `${encodeURIComponent(f)}=${encodeURIComponent(v)}`
  }
  return fs
}
