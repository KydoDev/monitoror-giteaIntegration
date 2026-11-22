<template>
  <div ref="outer" class="c-monitoror-tile-item-pills">
  <div ref="inner" class="c-monitoror-tile-item-pills--items">
    <!-- Same look as the sub-tiles of a GROUP tile: status colour on the text, progress bar under running items -->
    <div v-for="(item, index) in items" :key="index" class="c-monitoror-sub-tile"
         :class="['c-monitoror-sub-tile__theme-' + theme, 'c-monitoror-sub-tile__status-' + subTileStatus(item)]">
      <div class="c-monitoror-sub-tile--content">
        <div class="c-monitoror-sub-tile--label">{{ item.label }}</div>

        <monitoror-tile-icon :tile-type="tileType" class="c-monitoror-sub-tile--icon"></monitoror-tile-icon>

        <div v-if="item.detail || isQueued(item)" class="c-monitoror-sub-tile--progress-time"
             :class="{'c-monitoror-sub-tile--progress-time__overtime': isOvertime(item)}">
          <template v-if="isQueued(item)">&bull;&bull;&bull;</template>
          <template v-else>{{ item.detail }}</template>
        </div>

        <div v-if="isRunning(item) || isQueued(item)" class="c-monitoror-sub-tile--progress">
          <div class="c-monitoror-sub-tile--progress-bar" :style="progressBarStyle(item)"></div>
        </div>
      </div>
    </div>
  </div>
  </div>
</template>

<script lang="ts">
import MonitororTileIcon from '@/components/TileIcon.vue'
import TileStatus from '@/enums/tileStatus'
import TileType from '@/enums/tileType'
import TileItem from '@/types/tileItem'
import {defineComponent, ref} from 'vue'
import useFitScale from '@/composables/useFitScale'

// Sub-tile status modifiers (see SubTile.vue)
const SUB_TILE_STATUSES: { [status: string]: string } = {
  [TileStatus.Success]: 'succeeded',
  [TileStatus.Failed]: 'failed',
  [TileStatus.Warning]: 'warning',
  [TileStatus.Running]: 'running',
  [TileStatus.Queued]: 'queued',
  [TileStatus.Canceled]: 'canceled',
  [TileStatus.ActionRequired]: 'action-required',
}

export default defineComponent({
  name: 'MonitororTileItemPills',
  components: {
    MonitororTileIcon,
  },
  props: {
    items: {
      type: Array as () => TileItem[],
      required: true,
    },
    tileType: {
      type: String as () => TileType,
      required: true,
    },
    theme: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    const outer = ref<HTMLElement>()
    const inner = ref<HTMLElement>()
    useFitScale(outer, inner, '.c-monitoror-sub-tile--label', () => props.items)

    const subTileStatus = (item: TileItem): string => {
      return (item.status && SUB_TILE_STATUSES[item.status]) || 'unknown'
    }

    const isRunning = (item: TileItem): boolean => item.status === TileStatus.Running
    const isQueued = (item: TileItem): boolean => item.status === TileStatus.Queued
    const isOvertime = (item: TileItem): boolean => item.progress !== undefined && item.progress > 100

    // Without estimate the bar stays full, like a sub-tile running for the first time
    const progressBarStyle = (item: TileItem): { transform: string } => {
      const progress = isRunning(item) && item.progress !== undefined ? Math.min(100, Math.max(0, item.progress)) : 100

      return {transform: `translateX(-${100 - progress}%)`}
    }

    return {
      outer,
      inner,
      subTileStatus,
      isRunning,
      isQueued,
      isOvertime,
      progressBarStyle,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-tile-item-pills {
    overflow: hidden;
  }

  .c-monitoror-tile-item-pills--items {
    display: flex;
    flex-wrap: wrap;
    align-content: flex-start;
    gap: 7px 0;
    min-width: 0;
    text-align: left;

    // One line per pill: long run titles are cut, the time stays visible
    .c-monitoror-sub-tile {
      min-width: 0;
      max-width: 100%;
    }

    .c-monitoror-sub-tile--content {
      display: flex;
      max-width: 100%;
      box-sizing: border-box;
    }

    .c-monitoror-sub-tile--label {
      min-width: 0;
      max-width: none;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .c-monitoror-sub-tile--progress-time {
      flex: none;
      white-space: nowrap;
    }
  }
</style>
