import TileConfig from '@/types/tileConfig'

type Config = {
  version: string,
  columns: number,
  zoom?: number,
  screenRows?: number,
  autoScroll?: { top?: number, bottom?: number },
  tiles: TileConfig[],
}

export default Config
