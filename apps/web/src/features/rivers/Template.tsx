import { Fragment, type ReactNode } from "react"

/**
 * A translated sentence with React nodes (links, numbers) in its `{name}`
 * placeholders, so word order stays the translation's. Pass the message
 * unfilled: `t("ns.key")` keeps its placeholders.
 */
export function Template({ text, values }: { text: string; values: Record<string, ReactNode> }) {
  return (
    <>
      {text.split(/(\{\w+\})/).map((part, index) => {
        const name = /^\{(\w+)\}$/.exec(part)?.[1]
        if (name === undefined || !(name in values)) return part
        return <Fragment key={index}>{values[name]}</Fragment>
      })}
    </>
  )
}
