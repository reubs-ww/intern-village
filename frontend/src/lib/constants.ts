export const APP_NAME = 'Intern Village'

export const POLLING_INTERVAL = 5000

export const SUBTASK_COLUMNS = [
  'Ready',
  'In Progress',
  'Completed',
  'Merged',
  'Blocked',
] as const

export type SubtaskColumn = (typeof SUBTASK_COLUMNS)[number]

export const COLUMN_STATUS_MAP: Record<SubtaskColumn, string[]> = {
  Ready: ['READY'],
  'In Progress': ['IN_PROGRESS'],
  Completed: ['COMPLETED'],
  Merged: ['MERGED'],
  Blocked: ['BLOCKED'],
}
