// Session 实时流 composable：合并 REST/SSE 事件、维护游标，并管理断线重连生命周期。
import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import api, { getApiError } from '../api'
import type { Session, SessionDetail, SessionEvent } from '../types'

export function useSessionStream(sessionId: Ref<number>, onRefresh: () => void) {
  const session = ref<Session | null>(null)
  const events = ref<SessionEvent[]>([])
  const loading = ref(true)
  const errorMessage = ref('')
  const now = ref(Date.now())
  let eventSource: EventSource | null = null
  let clockTimer: number | undefined
  let reconnectTimer: number | undefined
  let maxEventID = 0
  let reconnectAttempt = 0
  let streamGeneration = 0
  let stopped = false
  let mounted = false

  function mergeEvents(nextEvents: SessionEvent[]) {
    const merged = new Map<number, SessionEvent>()
    for (const event of events.value) merged.set(event.id, event)
    for (const event of nextEvents) {
      merged.set(event.id, event)
      if (event.id > maxEventID) maxEventID = event.id
    }
    events.value = [...merged.values()].sort((left, right) => left.id - right.id)
  }

  function appendEvent(event: SessionEvent) {
    const previousMax = maxEventID
    mergeEvents([event])
    if (event.eventType === 'run_settled' || event.eventType === 'session_finished' || event.eventType === 'session_failed') {
      onRefresh()
      void loadSession()
    }
    if (event.id > previousMax) errorMessage.value = ''
  }

  async function loadSession(): Promise<boolean> {
    if (!sessionId.value || stopped) return false
    try {
      const [detailResponse, eventResponse] = await Promise.all([
        api.get<SessionDetail>(`/session/${sessionId.value}`),
        api.get<SessionEvent[]>(`/session/${sessionId.value}/events?afterId=${maxEventID}`),
      ])
      if (stopped) return false
      session.value = detailResponse.data.session
      mergeEvents(eventResponse.data)
      errorMessage.value = ''
      return true
    } catch (error) {
      errorMessage.value = getApiError(error, '实时行动读取失败')
      return false
    } finally {
      loading.value = false
    }
  }

  function closeStream() {
    streamGeneration += 1
    eventSource?.close()
    eventSource = null
  }

  function clearReconnectTimer() {
    if (reconnectTimer !== undefined) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = undefined
    }
  }

  function scheduleReconnect() {
    if (stopped || reconnectTimer !== undefined || ['success', 'incapacitated', 'failed'].includes(session.value?.status ?? '')) return
    closeStream()
    const delay = Math.min(1000 * (2 ** reconnectAttempt), 30000)
    reconnectAttempt += 1
    reconnectTimer = window.setTimeout(async () => {
      reconnectTimer = undefined
      const loaded = await loadSession()
      if (loaded && !stopped) {
        connectStream()
      } else if (!stopped) {
        scheduleReconnect()
      }
    }, delay)
  }

  function connectStream() {
    closeStream()
    if (stopped || !sessionId.value || ['success', 'incapacitated', 'failed'].includes(session.value?.status ?? '')) return
    const generation = streamGeneration
    eventSource = new EventSource(`/api/session/${sessionId.value}/events/stream?afterId=${maxEventID}`)
    eventSource.onopen = () => {
      if (generation !== streamGeneration || stopped) return
      reconnectAttempt = 0
      errorMessage.value = ''
    }
    eventSource.addEventListener('session_event', (message) => {
      if (generation !== streamGeneration || stopped) return
      try {
        appendEvent(JSON.parse((message as MessageEvent<string>).data) as SessionEvent)
      } catch {
        errorMessage.value = '实时事件格式异常，正在重新同步'
        scheduleReconnect()
      }
    })
    eventSource.addEventListener('stream_end', () => {
      if (generation !== streamGeneration || stopped) return
      closeStream()
      void loadSession().then((loaded) => { if (!loaded) scheduleReconnect() })
    })
    eventSource.onerror = () => {
      if (generation !== streamGeneration || stopped) return
      if (['success', 'incapacitated', 'failed'].includes(session.value?.status ?? '')) closeStream()
      else scheduleReconnect()
    }
  }

  async function reconnectNow() {
    if (stopped) return
    clearReconnectTimer()
    reconnectAttempt = 0
    closeStream()
    const loaded = await loadSession()
    if (loaded) connectStream()
    else scheduleReconnect()
  }

  function resetSessionState() {
    clearReconnectTimer()
    closeStream()
    session.value = null
    events.value = []
    maxEventID = 0
    reconnectAttempt = 0
    errorMessage.value = ''
    loading.value = true
  }

  watch(sessionId, async (nextSessionId, previousSessionId) => {
    if (!mounted || !nextSessionId || nextSessionId === previousSessionId) return
    resetSessionState()
    const loaded = await loadSession()
    if (loaded) connectStream()
    else scheduleReconnect()
  })

  onMounted(async () => {
    stopped = false
    mounted = true
    clockTimer = window.setInterval(() => { now.value = Date.now() }, 1000)
    const loaded = await loadSession()
    if (loaded) connectStream()
    else scheduleReconnect()
  })

  onUnmounted(() => {
    stopped = true
    mounted = false
    closeStream()
    clearReconnectTimer()
    if (clockTimer !== undefined) window.clearInterval(clockTimer)
  })

  return { session, events, loading, errorMessage, now, loadSession, reconnectNow }
}
