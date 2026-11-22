import {nextTick, onBeforeUnmount, onMounted, Ref, watch} from 'vue'

const MIN_SCALE = 0.75 // below this texts are no longer readable: items that do not fit are cut instead
const MAX_SCALE = 2
const STEP = 0.05
const TOLERANCE = 1 // px
// A label already cut at scale 1 may lose at most this share of its visible text when the box grows
const MAX_EXTRA_CUT = 0.1

// Scales the content of a box (CSS zoom on `inner`) to use the free space of `outer`.
// Texts grow while everything fits in height and labels stay as readable as at scale 1 (a label shown in full
// stays in full, a cut label does not lose more than MAX_EXTRA_CUT of its visible text), and shrink (down to
// MIN_SCALE) when the content overflows.
// `layouts` are optional CSS classes toggled on `inner` (e.g. a two-line layout): the one giving the biggest
// scale wins, the first one on a tie.
// `truncatedSelector` matches the elements whose text is cut with an ellipsis when too long.
export default function useFitScale(
  outer: Ref<HTMLElement | undefined>,
  inner: Ref<HTMLElement | undefined>,
  truncatedSelector: string,
  source: () => unknown,
  layouts: string[] = [''],
) {
  let observer: ResizeObserver | undefined
  let frame: number | undefined

  const fitsHeight = (outerElement: HTMLElement): boolean => {
    return outerElement.scrollHeight <= outerElement.clientHeight + TOLERANCE
  }

  // Visible share of the text of each truncatable element (1 when shown in full)
  const visibleShares = (innerElement: HTMLElement): number[] => {
    return Array.from(innerElement.querySelectorAll<HTMLElement>(truncatedSelector)).map((element) => {
      if (element.scrollWidth <= element.clientWidth + TOLERANCE || element.scrollWidth === 0) {
        return 1
      }
      return element.clientWidth / element.scrollWidth
    })
  }

  const asReadable = (shares: number[], baseShares: number[]): boolean => {
    return shares.every((share, index) => {
      const base = baseShares[index]
      return base === undefined || (base >= 1 ? share >= 1 : share >= base * (1 - MAX_EXTRA_CUT))
    })
  }

  const setLayout = (innerElement: HTMLElement, layout: string) => {
    layouts.forEach((candidate) => {
      if (candidate !== '') {
        innerElement.classList.toggle(candidate, candidate === layout)
      }
    })
  }

  const setScale = (innerElement: HTMLElement, scale: number) => {
    innerElement.style.setProperty('zoom', String(Math.round(scale * 100) / 100))
  }

  // Biggest scale for the current layout, compared with the readability of the first layout at scale 1
  const bestScale = (outerElement: HTMLElement, innerElement: HTMLElement, baseShares: number[]): number => {
    setScale(innerElement, 1)
    if (!fitsHeight(outerElement)) {
      for (let scale = 1 - STEP; scale >= MIN_SCALE; scale -= STEP) {
        setScale(innerElement, scale)
        if (fitsHeight(outerElement)) {
          return scale
        }
      }
      return MIN_SCALE
    }

    for (let scale = MAX_SCALE; scale > 1; scale -= STEP) {
      setScale(innerElement, scale)
      if (fitsHeight(outerElement) && asReadable(visibleShares(innerElement), baseShares)) {
        return scale
      }
    }
    return asReadable(visibleShares(innerElement), baseShares) ? 1 : 0
  }

  const fit = () => {
    frame = undefined
    const outerElement = outer.value
    const innerElement = inner.value
    if (!outerElement || !innerElement || outerElement.clientHeight === 0) {
      return
    }

    setLayout(innerElement, layouts[0])
    setScale(innerElement, 1)
    const baseShares = visibleShares(innerElement)

    let best = {layout: layouts[0], scale: -1}
    layouts.forEach((layout) => {
      setLayout(innerElement, layout)
      const scale = bestScale(outerElement, innerElement, baseShares)
      if (scale > best.scale + STEP / 2) {
        best = {layout, scale}
      }
    })

    setLayout(innerElement, best.layout)
    setScale(innerElement, Math.max(best.scale, MIN_SCALE))
  }

  const schedule = () => {
    if (frame === undefined) {
      frame = requestAnimationFrame(fit)
    }
  }

  onMounted(() => {
    observer = new ResizeObserver(schedule)
    if (outer.value) {
      observer.observe(outer.value)
    }
    nextTick(schedule)
  })

  watch(source, () => nextTick(schedule), {deep: true})

  onBeforeUnmount(() => {
    if (observer) {
      observer.disconnect()
    }
    if (frame !== undefined) {
      cancelAnimationFrame(frame)
    }
  })
}
