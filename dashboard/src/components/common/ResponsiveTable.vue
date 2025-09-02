<template>
  <div>
    <!-- Desktop/tablet table -->
    <div class="hidden md:block overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th v-for="col in columns" :key="col.key" scope="col"
                class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                :class="col.headerClass">
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="item in items" :key="rowKeyValue(item)">
            <td v-for="col in columns" :key="col.key" class="px-4 py-2 text-sm text-gray-900" :class="col.cellClass">
              <slot :name="`cell-${col.key}`" :item="item">
                {{ item[col.key] }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile stacked cards -->
    <div class="md:hidden space-y-3">
      <div v-for="item in items" :key="rowKeyValue(item)" class="bg-white rounded border shadow-sm">
        <div class="p-3">
          <div v-for="col in columns" :key="col.key" class="grid grid-cols-3 gap-2 py-1">
            <div class="col-span-1 text-xs font-medium text-gray-500">{{ col.label }}</div>
            <div class="col-span-2 text-sm text-gray-900 break-words">
              <slot :name="`cell-${col.key}`" :item="item">
                {{ item[col.key] }}
              </slot>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

type Column = {
  key: string
  label: string
  headerClass?: string
  cellClass?: string
}

const props = defineProps<{
  columns: Column[]
  items: any[]
  rowKey?: string | ((row: any) => string | number)
}>()

const rowKeyValue = (row: any) => {
  if (!props.rowKey) return JSON.stringify(row)
  if (typeof props.rowKey === 'function') return props.rowKey(row)
  return row[props.rowKey]
}
</script>
