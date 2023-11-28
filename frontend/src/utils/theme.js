import { reactive } from 'vue'

const THEME_STORAGE_KEY = 'theme'
const LIGHT = 'light'
const DARK = 'dark'

export const theme = reactive({
  name: localStorage.getItem(THEME_STORAGE_KEY) || DARK,
})

theme.isDark = () => theme.name === DARK
theme.isLight = () => theme.name === LIGHT

theme.init = () => refresh()

theme.toggle = () => {
  if (theme.name === LIGHT) theme.name = DARK
  else theme.name = LIGHT

  localStorage.setItem(THEME_STORAGE_KEY, theme.name)

  refresh()
}

function refresh() {
  if (theme.name === DARK) {
    document.body.setAttribute('data-bs-theme', DARK)
  } else {
    document.body.removeAttribute('data-bs-theme')
  }
}
