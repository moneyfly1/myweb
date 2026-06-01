import { onUnmounted } from 'vue'

export function usePaymentStatusPolling({
  intervalMs = 3000,
  timeoutMs = 30 * 60 * 1000,
  shouldPoll,
  poll,
  onCleanup,
}) {
  let intervalId = null
  let timeoutId = null
  let visibilityHandler = null
  let focusHandler = null

  const clearPolling = () => {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = null
    }
    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
      visibilityHandler = null
    }
    if (focusHandler) {
      window.removeEventListener('focus', focusHandler)
      focusHandler = null
    }
    onCleanup?.()
  }

  const runPoll = async () => {
    if (!shouldPoll?.()) {
      clearPolling()
      return
    }
    await poll()
  }

  const startPolling = () => {
    clearPolling()
    runPoll()
    intervalId = setInterval(runPoll, intervalMs)
    visibilityHandler = () => {
      if (document.visibilityState === 'visible' && shouldPoll?.()) {
        runPoll()
      }
    }
    focusHandler = () => {
      if (shouldPoll?.()) {
        runPoll()
      }
    }
    document.addEventListener('visibilitychange', visibilityHandler)
    window.addEventListener('focus', focusHandler)
    timeoutId = setTimeout(clearPolling, timeoutMs)
  }

  onUnmounted(clearPolling)

  return {
    startPolling,
    clearPolling,
  }
}
