import { Fragment, type ReactNode } from "react"

interface SentenceProps {
  /** A translated sentence whose `{name}` placeholders are left for `values`. */
  text: string
  values: Record<string, ReactNode>
}

/**
 * A whole translated sentence with links or times in it, so each language
 * keeps its own word order: "Withdrawn by the notice of {time}" puts the time
 * where the language puts it.
 */
export function Sentence({ text, values }: SentenceProps) {
  const parts = text.split(/\{(\w+)\}/g)
  return (
    <>
      {parts.map((part, index) =>
        index % 2 === 1 ? (
          <Fragment key={index}>{part in values ? values[part] : `{${part}}`}</Fragment>
        ) : (
          part
        ),
      )}
    </>
  )
}
