import { useState } from 'react'
import { Copy, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'

interface LogViewerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  content?: string
  isLoading?: boolean
}

export function LogViewer({
  open,
  onOpenChange,
  title,
  content,
  isLoading,
}: LogViewerProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!content) return
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard API not available
    }
  }

  const lines = content?.split('\n') ?? []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl h-[80vh] flex flex-col">
        <DialogHeader className="flex flex-row items-center justify-between pr-10">
          <DialogTitle>{title}</DialogTitle>
          <Button
            variant="outline"
            size="sm"
            onClick={handleCopy}
            disabled={!content || isLoading}
          >
            {copied ? (
              <>
                <Check className="mr-2 h-4 w-4" />
                Copied
              </>
            ) : (
              <>
                <Copy className="mr-2 h-4 w-4" />
                Copy
              </>
            )}
          </Button>
        </DialogHeader>

        <div className="flex-1 min-h-0 rounded-lg border bg-muted/50">
          {isLoading ? (
            <div className="p-4 space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          ) : !content ? (
            <div className="flex h-full items-center justify-center">
              <div className="text-center text-muted-foreground">
                <X className="mx-auto mb-2 h-8 w-8 opacity-50" />
                <p>No logs available</p>
              </div>
            </div>
          ) : (
            <ScrollArea className="h-full">
              <div className="p-4">
                <pre className="font-mono text-sm leading-relaxed">
                  {lines.map((line, i) => (
                    <div key={i} className="flex">
                      <span className="mr-4 w-10 select-none text-right text-muted-foreground/50">
                        {i + 1}
                      </span>
                      <span className="flex-1 whitespace-pre-wrap break-all text-foreground">
                        {line || ' '}
                      </span>
                    </div>
                  ))}
                </pre>
              </div>
            </ScrollArea>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
