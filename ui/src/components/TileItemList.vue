<template>
  <div ref="outer" class="c-monitoror-tile-item-list">
  <ul ref="inner" class="c-monitoror-tile-item-list--items">
    <li v-for="(item, index) in items" :key="index" class="c-monitoror-tile-item-list--item">
      <div class="c-monitoror-tile-item-list--row">
        <span v-if="item.status" class="c-monitoror-tile-item-list--status"
              :class="'c-monitoror-tile-item-list--status__' + statusClass(item.status)"></span>
        <span class="c-monitoror-tile-item-list--label">{{ item.label }}</span>
        <span class="c-monitoror-tile-item-list--break"></span>
        <span v-if="item.tag" class="c-monitoror-tile-item-list--tag"
              :class="{'c-monitoror-tile-item-list--tag__bug': isBug(item.tag)}">{{ item.tag }}</span>
        <span v-if="item.detail" class="c-monitoror-tile-item-list--detail">{{ item.detail }}</span>
      </div>
      <div v-if="item.progress !== undefined" class="c-monitoror-tile-item-list--progress"
           :class="{'c-monitoror-tile-item-list--progress__overtime': item.progress > 100}">
        <div class="c-monitoror-tile-item-list--progress-bar" :style="{width: progressWidth(item.progress)}"></div>
      </div>
    </li>
  </ul>
  </div>
</template>

<script lang="ts">
import {defineComponent, ref} from 'vue'

import TileItem from '@/types/tileItem'
import {statusClass, isBug} from '@/helpers/tileItem'
import useFitScale from '@/composables/useFitScale'

// One item per line, in a box with hidden overflow: texts grow to use the free space of the box,
// shrink a little when it is full, and items that still do not fit are cut.
export default defineComponent({
  name: 'MonitororTileItemList',
  props: {
    items: {
      type: Array as () => TileItem[],
      required: true,
    },
  },
  setup(props) {
    const outer = ref<HTMLElement>()
    const inner = ref<HTMLElement>()
    // Second layout: the details on a line of their own, so that bigger texts fit when there is room in height
    useFitScale(outer, inner, '.c-monitoror-tile-item-list--label', () => props.items,
      ['', 'c-monitoror-tile-item-list__stacked'])

    const progressWidth = (progress: number): string => {
      return Math.min(100, Math.max(0, progress)) + '%'
    }

    return {
      outer,
      inner,
      statusClass,
      isBug,
      progressWidth,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-tile-item-list {
    overflow: hidden;
  }

  .c-monitoror-tile-item-list--items {
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: 20px;
    line-height: 1.25;
  }

  .c-monitoror-tile-item-list--item {
    padding: 3px 0;
  }

  .c-monitoror-tile-item-list--row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .c-monitoror-tile-item-list--break {
    display: none;
  }

  .c-monitoror-tile-item-list__stacked {
    .c-monitoror-tile-item-list--item {
      padding: 4px 0;
    }

    .c-monitoror-tile-item-list--row {
      flex-wrap: wrap;
      row-gap: 0;
    }

    .c-monitoror-tile-item-list--break {
      display: block;
      flex-basis: 100%;
      height: 0;
    }

    .c-monitoror-tile-item-list--detail {
      flex: 1 1 0;
      max-width: none;
      text-align: left;
    }
  }

  .c-monitoror-tile-item-list--label {
    flex: 1 1 auto;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .c-monitoror-tile-item-list--detail {
    flex: 0 1 auto;
    max-width: 50%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 17px;
    opacity: 0.65;
    text-align: right;
  }

  .c-monitoror-tile-item-list--tag {
    flex: 0 0 auto;
    padding: 0 7px;
    font-size: 14px;
    line-height: 20px;
    border: 1px solid currentColor;
    border-radius: 10px;
    opacity: 0.7;

    &__bug {
      color: #fff;
      background: #d63031;
      border-color: #d63031;
      opacity: 1;
    }
  }

  .c-monitoror-tile-item-list--status {
    flex: 0 0 auto;
    width: 12px;
    height: 12px;
    background: var(--color-unknown);
    border: 2px solid currentColor;
    border-radius: 100%;
    box-sizing: border-box;

    &__success {
      background: #2ecc71;
    }

    &__failed {
      background: #e74c3c;
    }

    &__warning {
      background: #f39c12;
    }

    &__running,
    &__queued {
      background: #3498db;
    }

    &__running {
      animation: tileItemBlink 1.5s ease-in-out infinite;
    }

    &__canceled,
    &__disabled {
      background: #95a5a6;
    }
  }

  .c-monitoror-tile-item-list--progress {
    height: 3px;
    margin-top: 2px;
    background: rgba(128, 128, 128, 0.3);
    border-radius: 2px;
    overflow: hidden;
  }

  .c-monitoror-tile-item-list--progress-bar {
    height: 100%;
    background: currentColor;
    opacity: 0.75;
    border-radius: 2px;

    .c-monitoror-tile-item-list--progress__overtime & {
      background: #e67e22;
      opacity: 1;
    }
  }

  @keyframes tileItemBlink {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.35;
    }
  }
</style>
