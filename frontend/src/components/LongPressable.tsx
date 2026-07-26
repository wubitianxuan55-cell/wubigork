import React, { useRef, useCallback } from 'react'

interface Props extends React.HTMLAttributes<HTMLDivElement> {
  onLongPress: () => void
  children: React.ReactNode
  delay?: number  // 默认 500ms
}

const LongPressable: React.FC<Props> = ({
  onLongPress, children, delay = 500, ...divProps
}) => {
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const movedRef = useRef(false)

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = undefined
    }
  }, [])

  return (
    <div
      {...divProps}
      onTouchStart={(_e) => {
        movedRef.current = false
        timerRef.current = setTimeout(() => {
          if (!movedRef.current) {
            // 震动反馈
            if (navigator.vibrate) navigator.vibrate(10)
            onLongPress()
          }
        }, delay)
      }}
      onTouchMove={() => { movedRef.current = true }}
      onTouchEnd={clear}
      onTouchCancel={clear}
      onContextMenu={(_e) => {
        // 桌面端右键不触发长按
      }}
    >
      {children}
    </div>
  )
}

export default LongPressable
