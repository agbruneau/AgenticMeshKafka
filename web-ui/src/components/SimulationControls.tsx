import { useState, useEffect } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Play, Square, Send, AlertCircle, CheckCircle } from 'lucide-react'
import { simulationApi, type StartSimulationRequest } from '../lib/api'
import { useSimulationStore } from '../store/simulationStore'

export default function SimulationControls() {
  const { status, eventsProduced, rateActual, updateMetrics, setStatus, setSimulationId } = useSimulationStore()
  const [rate, setRate] = useState(10)
  const [duration, setDuration] = useState(0)
  const [eventType, setEventType] = useState('CompteOuvert')
  const [eventCount, setEventCount] = useState(1)

  // Fetch status periodically when running
  const { data: statusData } = useQuery({
    queryKey: ['simulationStatus'],
    queryFn: simulationApi.status,
    refetchInterval: status === 'running' ? 1000 : false,
    enabled: status === 'running',
  })

  // Handle status data updates
  useEffect(() => {
    if (statusData) {
      updateMetrics({
        eventsProduced: statusData.events_produced,
        eventsFailed: statusData.events_failed,
        rateActual: statusData.rate_actual,
      })
      if (statusData.status === 'stopped') {
        setStatus('stopped')
      }
    }
  }, [statusData, updateMetrics, setStatus])

  // Start mutation
  const startMutation = useMutation({
    mutationFn: (request: StartSimulationRequest) => simulationApi.start(request),
    onSuccess: (data) => {
      setSimulationId(data.simulation_id)
      setStatus('running')
    },
  })

  // Stop mutation
  const stopMutation = useMutation({
    mutationFn: simulationApi.stop,
    onSuccess: (data) => {
      setStatus('stopped')
      updateMetrics({ eventsProduced: data.events_produced })
    },
  })

  // Produce events mutation
  const produceMutation = useMutation({
    mutationFn: () => simulationApi.produceEvents({ event_type: eventType, count: eventCount }),
    onSuccess: (data) => {
      updateMetrics({ eventsProduced: eventsProduced + data.events_produced })
    },
  })

  const handleStart = () => {
    startMutation.mutate({
      rate,
      duration: duration > 0 ? duration : undefined,
      event_types: ['CompteOuvert', 'DepotEffectue', 'VirementEmis'],
    })
  }

  const handleStop = () => {
    stopMutation.mutate()
  }

  const isLoading = startMutation.isPending || stopMutation.isPending

  return (
    <div className="space-y-6">
      {/* Simulation Control */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold text-slate-800">Simulation</h2>

        {/* Status */}
        <div className="flex items-center gap-2 p-3 rounded-lg bg-slate-100">
          {status === 'running' ? (
            <>
              <CheckCircle className="w-5 h-5 text-green-500" />
              <span className="text-green-700 font-medium">Running</span>
            </>
          ) : (
            <>
              <AlertCircle className="w-5 h-5 text-slate-400" />
              <span className="text-slate-600">Stopped</span>
            </>
          )}
        </div>

        {/* Rate Slider */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">
            Rate: {rate} events/sec
          </label>
          <input
            type="range"
            min="1"
            max="100"
            value={rate}
            onChange={(e) => setRate(parseInt(e.target.value))}
            disabled={status === 'running'}
            className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer disabled:opacity-50"
          />
        </div>

        {/* Duration Input */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">
            Duration (seconds, 0 = infinite)
          </label>
          <input
            type="number"
            min="0"
            max="3600"
            value={duration}
            onChange={(e) => setDuration(parseInt(e.target.value) || 0)}
            disabled={status === 'running'}
            className="w-full px-3 py-2 border border-slate-300 rounded-md disabled:opacity-50 disabled:bg-slate-100"
          />
        </div>

        {/* Start/Stop Button */}
        <button
          onClick={status === 'running' ? handleStop : handleStart}
          disabled={isLoading}
          className={`w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg font-medium transition-colors ${
            status === 'running'
              ? 'bg-red-500 hover:bg-red-600 text-white'
              : 'bg-primary-600 hover:bg-primary-700 text-white'
          } disabled:opacity-50`}
        >
          {status === 'running' ? (
            <>
              <Square className="w-5 h-5" />
              Stop Simulation
            </>
          ) : (
            <>
              <Play className="w-5 h-5" />
              Start Simulation
            </>
          )}
        </button>

        {/* Metrics */}
        {status === 'running' && (
          <div className="grid grid-cols-2 gap-3 p-3 bg-slate-100 rounded-lg">
            <div>
              <div className="text-2xl font-bold text-slate-800">{eventsProduced}</div>
              <div className="text-xs text-slate-500">Events</div>
            </div>
            <div>
              <div className="text-2xl font-bold text-slate-800">{rateActual.toFixed(1)}</div>
              <div className="text-xs text-slate-500">Rate/sec</div>
            </div>
          </div>
        )}
      </div>

      {/* Manual Event Production */}
      <div className="space-y-4 pt-4 border-t border-slate-200">
        <h3 className="text-lg font-semibold text-slate-800">Manual Production</h3>

        {/* Event Type Select */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">Event Type</label>
          <select
            value={eventType}
            onChange={(e) => setEventType(e.target.value)}
            className="w-full px-3 py-2 border border-slate-300 rounded-md"
          >
            <option value="CompteOuvert">CompteOuvert</option>
            <option value="DepotEffectue">DepotEffectue</option>
            <option value="VirementEmis">VirementEmis</option>
          </select>
        </div>

        {/* Count Input */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">Count</label>
          <input
            type="number"
            min="1"
            max="100"
            value={eventCount}
            onChange={(e) => setEventCount(parseInt(e.target.value) || 1)}
            className="w-full px-3 py-2 border border-slate-300 rounded-md"
          />
        </div>

        {/* Produce Button */}
        <button
          onClick={() => produceMutation.mutate()}
          disabled={produceMutation.isPending}
          className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-900 text-white rounded-lg font-medium disabled:opacity-50"
        >
          <Send className="w-4 h-4" />
          Produce Events
        </button>

        {produceMutation.isSuccess && (
          <div className="text-sm text-green-600 text-center">
            Produced {produceMutation.data.events_produced} events
          </div>
        )}
      </div>
    </div>
  )
}
