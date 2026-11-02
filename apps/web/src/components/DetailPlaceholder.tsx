import type * as React from "react"
import { XIcon } from "lucide-react"

import { Panel } from "@/components/Panel"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"

interface DetailPlaceholderProps {
  icon: React.ReactNode
  title: string
  /** A sentence instead of the loading placeholders, e.g. "This gauge is not in the latest readings." */
  message?: string
  onClose: () => void
}

/** The detail panel while its data or code loads, or when its subject cannot be found. */
export function DetailPlaceholder({ icon, title, message, onClose }: DetailPlaceholderProps) {
  return (
    <Panel
      icon={icon}
      title={title}
      className="w-full"
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close details">
          <XIcon />
        </Button>
      }
    >
      {message ? (
        <p className="rounded-xl border border-border bg-inset px-3 py-3 text-sm text-muted-foreground">
          {message}
        </p>
      ) : (
        <div className="flex flex-col gap-3" role="status" aria-label="Loading">
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-4 w-52" />
          <Skeleton className="h-28 w-full rounded-2xl" />
        </div>
      )}
    </Panel>
  )
}
