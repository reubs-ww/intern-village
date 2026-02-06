import { ListFilter } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Task } from '@/types/api'

interface TaskFilterProps {
  tasks: Task[]
  selectedTaskId: string | null
  onSelect: (taskId: string | null) => void
}

export function TaskFilter({ tasks, selectedTaskId, onSelect }: TaskFilterProps) {
  return (
    <Select
      value={selectedTaskId ?? 'all'}
      onValueChange={(value) => onSelect(value === 'all' ? null : value)}
    >
      <SelectTrigger className="w-[200px]">
        <ListFilter className="mr-2 h-4 w-4" />
        <SelectValue placeholder="Filter by task" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">All Tasks</SelectItem>
        {tasks.map((task) => (
          <SelectItem key={task.id} value={task.id}>
            {task.title}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
