import TileStatus from '@/enums/tileStatus'

// Suffix of the CSS modifier matching the status of a tile item
export function statusClass(status: TileStatus): string {
  switch (status) {
    case TileStatus.Success:
      return 'success'
    case TileStatus.Failed:
      return 'failed'
    case TileStatus.Warning:
    case TileStatus.ActionRequired:
      return 'warning'
    case TileStatus.Running:
      return 'running'
    case TileStatus.Queued:
      return 'queued'
    case TileStatus.Canceled:
      return 'canceled'
    default:
      return 'disabled'
  }
}

export function isBug(tag: string): boolean {
  return tag.trim().toLowerCase() === 'bug'
}
