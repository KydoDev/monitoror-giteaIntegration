<template>
  <div class="c-monitoror-sparkline">
    <svg viewBox="0 0 100 40" preserveAspectRatio="none">
      <path v-if="areaPath" :d="areaPath" fill="currentColor" class="c-monitoror-sparkline--area"></path>
      <path v-if="linePath" :d="linePath" fill="none" stroke="currentColor" stroke-width="4"
            stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke"
            class="c-monitoror-sparkline--line"></path>
    </svg>
  </div>
</template>

<script lang="ts">
import {computed, defineComponent} from 'vue'

const WIDTH = 100
const HEIGHT = 40
const MARGIN_Y = 3

type Point = { x: number, y: number }

// Smooth line chart (like the commit activity graph): the first value is the oldest one.
// The chart stretches to fill the space of the tile.
export default defineComponent({
  name: 'MonitororSparkline',
  props: {
    values: {
      type: Array as () => number[],
      required: true,
    },
  },
  setup(props) {
    const points = computed((): Point[] => {
      const count = props.values.length
      if (count === 0) {
        return []
      }

      const max = Math.max(1, ...props.values)
      const usableHeight = HEIGHT - 2 * MARGIN_Y

      return props.values.map((value, index): Point => ({
        x: count === 1 ? WIDTH / 2 : index * WIDTH / (count - 1),
        y: HEIGHT - MARGIN_Y - Math.max(0, value) / max * usableHeight,
      }))
    })

    const round = (n: number): number => Math.round(n * 100) / 100

    // Cubic curves with horizontal tangents at each point: smooth and never overshoots the values
    const linePath = computed((): string | undefined => {
      const pts = points.value
      if (pts.length === 0) {
        return
      }
      if (pts.length === 1) {
        return `M0 ${round(pts[0].y)} L${WIDTH} ${round(pts[0].y)}`
      }

      let path = `M${round(pts[0].x)} ${round(pts[0].y)}`
      for (let i = 1; i < pts.length; i++) {
        const middle = (pts[i - 1].x + pts[i].x) / 2
        path += ` C${round(middle)} ${round(pts[i - 1].y)} ${round(middle)} ${round(pts[i].y)} ` +
          `${round(pts[i].x)} ${round(pts[i].y)}`
      }

      return path
    })

    const areaPath = computed((): string | undefined => {
      if (linePath.value === undefined || points.value.length < 2) {
        return
      }

      return `${linePath.value} L${WIDTH} ${HEIGHT} L0 ${HEIGHT} Z`
    })

    return {
      linePath,
      areaPath,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-sparkline svg {
    display: block;
    width: 100%;
    height: 100%;
    overflow: visible;
  }

  .c-monitoror-sparkline--line {
    opacity: 0.55;
  }

  .c-monitoror-sparkline--area {
    opacity: 0.12;
  }
</style>
