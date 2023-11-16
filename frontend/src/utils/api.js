import _ from 'lodash'
import moment from 'moment'
import { LRUCache } from 'lru-cache'
import axios from 'axios'
import { reactive } from 'vue'

let cache = new LRUCache({ max: 100 })
let responses = []

export const api = reactive({
  loading: false,
  errors: '',
})

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function onResult(err) {
  responses.push({
    err: err,
    time: moment(),
  })

  let limit = moment().subtract(1, 'minute')
  responses = _.chain(responses)
    .filter((r) => r.time.isAfter(limit))
    .value()

  api.errors = _.chain(responses)
    .filter((r) => r.err)
    .map((e) => `[${e.time.format('HH:mm:ss')}] ${e.err}`)
    .join('\n')
    .value()
}

api.get = async function (url) {
  api.loading = true
  try {
    await sleep(1)

    let result = cache.get(url)
    if (result) {
      return result
    }

    let response = await axios.get(url)
    cache.set(url, response.data)
    onResult(null)
    return response.data
  } catch (e) {
    onResult(`get ${url}: ${e.message}`)
    throw e.message
  } finally {
    api.loading = false
  }
}

api.patch = async function (url, body) {
  api.loading = true
  try {
    let response = await axios.patch(url, body)
    cache.clear()
    onResult(null)
    return response.data
  } catch (e) {
    onResult(`patch ${url}: ${e.message}`)
    throw e.message
  } finally {
    api.loading = false
  }
}
