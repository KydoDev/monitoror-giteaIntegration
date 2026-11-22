<template>
  <div class="c-monitoror-tile-chips">
    <span v-for="(item, index) in items" :key="index" class="c-monitoror-tile-chips--chip"
          :class="chipClass(item)">
      {{ item.label }}<template v-if="item.detail"> <strong>{{ item.detail }}</strong></template>
    </span>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'

import TileItem from '@/types/tileItem'
import {statusClass, isBug} from '@/helpers/tileItem'

// Compact row of pills: label + optional detail, coloured by tag or status. Pills that do not fit are cut.
export default defineComponent({
  name: 'MonitororTileChips',
  props: {
    items: {
      type: Array as () => TileItem[],
      required: true,
    },
  },
  setup() {
    const chipClass = (item: TileItem): string => {
      if (item.tag !== undefined && isBug(item.tag)) {
        return 'c-monitoror-tile-chips--chip__bug'
      }
      if (item.status !== undefined) {
        return 'c-monitoror-tile-chips--chip__' + statusClass(item.status)
      }

      return ''
    }

    return {
      chipClass,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-tile-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    max-height: 62px;
    overflow: hidden;
  }

  .c-monitoror-tile-chips--chip {
    padding: 2px 10px;
    font-size: 18px;
    line-height: 24px;
    white-space: nowrap;
    border: 1px solid currentColor;
    border-radius: 14px;

    strong {
      margin-left: 2px;
    }

    &__bug,
    &__failed {
      color: #fff;
      background: #d63031;
      border-color: #d63031;
    }

    &__success {
      background: #2ecc71;
      border-color: #2ecc71;
      color: #11263a;
    }

    &__warning {
      background: #f39c12;
      border-color: #f39c12;
      color: #11263a;
    }

    &__running,
    &__queued {
      color: #fff;
      background: #3498db;
      border-color: #3498db;
    }

    &__canceled,
    &__disabled {
      color: #fff;
      background: #95a5a6;
      border-color: #95a5a6;
    }
  }
</style>
