import TileStatus from '@/enums/tileStatus'
import TileType from '@/enums/tileType'
import TileBuild from '@/types/tileBuild'
import TileItem from '@/types/tileItem'
import TileMetrics from '@/types/tileMetrics'

type TileState = {
  type: TileType,
  status: TileStatus,
  label?: string,
  message?: string,
  metrics?: TileMetrics,
  build?: TileBuild,
  items?: TileItem[],
}

export default TileState
