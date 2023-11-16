export function sortParams(page, pageSize, sort, asc) {
  return `sort=${encodeURIComponent(sort)}&asc=${asc}&offset=${(page - 1) * pageSize}&limit=${pageSize}`
}
