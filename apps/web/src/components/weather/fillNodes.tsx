import { Fragment, type ReactNode } from "react"

/**
 * A translated template with React nodes in its `{placeholders}`, for a
 * link or a styled number inside a sentence whose word order differs between
 * languages. Pass the message without values, so its placeholders remain.
 */
export function fillNodes(template: string, nodes: Record<string, ReactNode>): ReactNode {
  const parts = template.split(/\{(\w+)\}/)
  return parts.map((part, index) =>
    index % 2 === 0 ? (
      part
    ) : (
      <Fragment key={index}>{part in nodes ? nodes[part] : `{${part}}`}</Fragment>
    ),
  )
}
