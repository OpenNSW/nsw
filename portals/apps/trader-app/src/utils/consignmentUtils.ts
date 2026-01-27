import type { ConsignmentState } from '../services/types/consignment'

/**
 * Get the appropriate color for a consignment state badge
 */
export function getStateColor(
  state: ConsignmentState
): 'gray' | 'orange' | 'green' | 'red' {
  switch (state) {
    case 'IN_PROGRESS':
      return 'orange'
    case 'FINISHED':
      return 'green'
    case 'REQUIRES_REWORK':
      return 'red'
    default:
      return 'gray'
  }
}

/**
 * Format a consignment state for display
 * Converts underscore-separated uppercase to title case with spaces
 * Example: IN_PROGRESS -> In Progress
 */
export function formatState(state: ConsignmentState): string {
  return state.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

/**
 * Format a date string for display
 * Example: 2026-01-27T10:30:00Z -> Jan 27, 2026
 */
export function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
