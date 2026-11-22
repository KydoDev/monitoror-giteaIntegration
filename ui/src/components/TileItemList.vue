<template>
  <ul class="c-monitoror-tile-item-list">
    <li v-for="(item, index) in items" :key="index" class="c-monitoror-tile-item-list--item">
      <div class="c-monitoror-tile-item-list--row">
        <span v-if="item.status" class="c-monitoror-tile-item-list--status"
              :class="'c-monitoror-tile-item-list--status__' + statusClass(item.status)"></span>
        <span class="c-monitoror-tile-item-list--label">{{ item.label }}</span>
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
</template>

<script lang="ts">
import {defineComponent} from 'vue'

import TileItem from '@/types/tileItem'
import {statusClass, isBug} from '@/helpers/tileItem'

// One item per line, in a box with hidden overflow: items that do not fit are simply cut.
export default defineComponent({
  name: 'MonitororTileItemList',
  props: {
    items: {
      type: Array as () => TileItem[],
      required: true,
    },
  },
  setup() {
    const progressWidth = (progress: number): string => {
      return Math.min(100, Math.max(0, progress)) + '%'
    }

    return {
      statusClass,
      isBug,
      progressWidth,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-tile-item-list {
    margin: 0;
    padding: 0;
    list-style: none;
    overflow: hidden;
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
