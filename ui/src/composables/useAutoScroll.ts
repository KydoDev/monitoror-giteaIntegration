import {onBeforeUnmount, onMounted, Ref, watch} from 'vue'

const DEFAULT_TOP_DELAY = 60 // seconds spent on each page before scrolling down
const DEFAULT_BOTTOM_DELAY = 20 // seconds spent on the last page before going back to the top
const USER_PAUSE = 30 // seconds the cycle is paused after the user interacted with the page
const AT_EDGE_TOLERANCE = 2 // px
const USER_EVENTS = ['wheel', 'keydown', 'touchstart', 'mousedown']

// Slowly scrolls the page one screen at a time, then back to the top, forever.
// The position is always read from the page, so manual scrolling and tiles wrapping on extra rows are supported.
// `pageHeight` is the distance of one scroll step (the grid rows are sized so that it matches whole rows).
export default function useAutoScroll(
  enabled: Ref<boolean>,
  top: Ref<number | undefined>,
  bottom: Ref<number | undefined>,
  pageHeight: () => number,
) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let pausedUntil = 0

  const clear = () => {
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  const maxScroll = (): number => {
    return document.documentElement.scrollHeight - window.innerHeight
  }

  const isAtBottom = (max: number): boolean => {
    return window.scrollY >= max - AT_EDGE_TOLERANCE
  }

  const schedule = (seconds: number) => {
    clear()
    timer = setTimeout(step, seconds * 1000)
  }

  const topDelay = (): number => top.value !== undefined && top.value > 0 ? top.value : DEFAULT_TOP_DELAY
  const bottomDelay = (): number => bottom.value !== undefined && bottom.value > 0 ? bottom.value : DEFAULT_BOTTOM_DELAY

  // Called when the dwell time of the current position is over
  const step = () => {
    const remainingPause = pausedUntil - Date.now()
    if (remainingPause > 0) {
      clear()
      timer = setTimeout(step, remainingPause + 100)
      return
    }

    const max = maxScroll()
    if (max <= AT_EDGE_TOLERANCE) {
      // Everything fits on the screen (for now): check again later
      schedule(topDelay())
      return
    }

    if (isAtBottom(max)) {
      window.scrollTo({top: 0, behavior: 'smooth'})
      schedule(topDelay())
      return
    }

    const target = Math.min(window.scrollY + Math.max(pageHeight(), 1), max)
    window.scrollTo({top: target, behavior: 'smooth'})
    schedule(target >= max - AT_EDGE_TOLERANCE ? bottomDelay() : topDelay())
  }

  const onUserInteraction = () => {
    pausedUntil = Date.now() + USER_PAUSE * 1000
  }

  const start = () => {
    schedule(topDelay())
  }

  const stop = () => {
    clear()
  }

  const apply = () => {
    if (enabled.value) {
      start()
    } else {
      stop()
    }
  }

  onMounted(() => {
    USER_EVENTS.forEach((event) => window.addEventListener(event, onUserInteraction, {passive: true}))
    apply()
  })

  onBeforeUnmount(() => {
    USER_EVENTS.forEach((event) => window.removeEventListener(event, onUserInteraction))
    stop()
  })

  watch([enabled, top, bottom], apply)
}
