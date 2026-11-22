<template>
  <div class="c-monitoror-contributions-graph">
    <svg :viewBox="viewBox" preserveAspectRatio="xMidYMid meet">
      <rect v-for="cell in cells" :key="cell.index"
            :x="cell.x" :y="cell.y" :width="cellSize" :height="cellSize" rx="2"
            :fill="cell.color"></rect>
    </svg>
  </div>
</template>

<script lang="ts">
import {computed, defineComponent} from 'vue'

const CELL_SIZE = 10
const CELL_GAP = 3
const DAYS_PER_WEEK = 7
// Colors of the 5 levels: no contribution, then 4 quartiles of the busiest day (GitHub dark palette)
const LEVEL_COLORS = ['#2d333b', '#0e4429', '#006d32', '#26a641', '#39d353']

type Cell = {
  index: number,
  x: number,
  y: number,
  color: string,
}

// One column per week, one row per weekday: the first value is a Sunday, the last value is today.
export default defineComponent({
  name: 'MonitororContributionsGraph',
  props: {
    values: {
      type: Array as () => number[],
      required: true,
    },
  },
  setup(props) {
    const weeks = computed((): number => {
      return Math.max(1, Math.ceil(props.values.length / DAYS_PER_WEEK))
    })

    const viewBox = computed((): string => {
      const step = CELL_SIZE + CELL_GAP
      return `0 0 ${weeks.value * step - CELL_GAP} ${DAYS_PER_WEEK * step - CELL_GAP}`
    })

    const cells = computed((): Cell[] => {
      const max = Math.max(0, ...props.values)

      return props.values.map((count, index) => {
        let level = 0
        if (count > 0 && max > 0) {
          level = Math.min(4, Math.ceil((count / max) * 4))
        }

        return {
          index,
          x: Math.floor(index / DAYS_PER_WEEK) * (CELL_SIZE + CELL_GAP),
          y: (index % DAYS_PER_WEEK) * (CELL_SIZE + CELL_GAP),
          color: LEVEL_COLORS[level],
        }
      })
    })

    return {
      cellSize: CELL_SIZE,
      viewBox,
      cells,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-contributions-graph svg {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
