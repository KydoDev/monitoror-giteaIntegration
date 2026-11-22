import TileStatus from '@/enums/tileStatus'

type TileItem = {
  label: string,
  detail?: string,
  status?: TileStatus,
  tag?: string,
  progress?: number,
}

export default TileItem
