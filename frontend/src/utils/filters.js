import { reactive } from 'vue'

export const filters = reactive({
  data: {
    repo_name: null,
  },
})

filters.clear = (fs) => {
  for (let f in filters.data) {
    filters.data[f] = null
  }
}

filters.patch = (fs) => {
  for (let f in fs) {
    filters.data[f] = fs[f]
  }
}

filters.toQueryString = (filterMapping) => {
  if (!filterMapping) filterMapping = {}

  let fs = ''
  for (let f in filters.data) {
    let v = filters.data[f]
    if (!v) continue

    if (filterMapping[f]) f = filterMapping[f]

    if (fs) fs += '&'
    fs += `${f}=${encodeURIComponent(v)}`
  }

  return fs
}
