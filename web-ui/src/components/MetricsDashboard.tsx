import { useQuery } from '@tanstack/react-query'
import { Activity, Database, AlertTriangle, TrendingUp, Zap } from 'lucide-react'
import { healthApi } from '../lib/api'
import { useSimulationStore } from '../store/simulationStore'

export default function MetricsDashboard() {
  const { eventsProduced, eventsFailed, rateActual, events } = useSimulationStore()

  // Fetch health status
  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: healthApi.check,
    refetchInterval: 5000,
  })

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      {/* Service Health */}
      <section>
        <h2 className="text-lg font-semibold text-slate-800 mb-4">Service Health</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
          <ServiceCard
            name="Gateway"
            status={health?.services?.gateway || 'unknown'}
          />
          <ServiceCard
            name="Simulator"
            status={health?.services?.simulator || 'unknown'}
          />
          <ServiceCard
            name="Bancaire"
            status={health?.services?.bancaire || 'unknown'}
          />
          <ServiceCard
            name="Kafka"
            status="healthy"
          />
          <ServiceCard
            name="PostgreSQL"
            status="healthy"
          />
        </div>
      </section>

      {/* Event Metrics */}
      <section>
        <h2 className="text-lg font-semibold text-slate-800 mb-4">Event Metrics</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Events Produced"
            value={eventsProduced}
            icon={<Activity className="w-5 h-5" />}
            trend={rateActual > 0 ? 'up' : undefined}
          />
          <MetricCard
            title="Events Failed"
            value={eventsFailed}
            icon={<AlertTriangle className="w-5 h-5" />}
            variant={eventsFailed > 0 ? 'warning' : 'default'}
          />
          <MetricCard
            title="Current Rate"
            value={`${rateActual.toFixed(1)}/s`}
            icon={<Zap className="w-5 h-5" />}
          />
          <MetricCard
            title="WebSocket Clients"
            value={health?.websocket?.clients || 0}
            icon={<TrendingUp className="w-5 h-5" />}
          />
        </div>
      </section>

      {/* Recent Events */}
      <section>
        <h2 className="text-lg font-semibold text-slate-800 mb-4">Recent Events</h2>
        <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
          <div className="max-h-96 overflow-y-auto">
            {events.length === 0 ? (
              <div className="p-8 text-center text-slate-500">
                <Database className="w-12 h-12 mx-auto mb-3 text-slate-300" />
                <p>No events yet. Start a simulation or produce events manually.</p>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-slate-50 sticky top-0">
                  <tr>
                    <th className="px-4 py-2 text-left font-medium text-slate-600">Time</th>
                    <th className="px-4 py-2 text-left font-medium text-slate-600">Type</th>
                    <th className="px-4 py-2 text-left font-medium text-slate-600">Topic</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {events.slice(0, 50).map((event) => (
                    <tr key={event.id} className="hover:bg-slate-50">
                      <td className="px-4 py-2 text-slate-500 font-mono text-xs">
                        {new Date(event.timestamp).toLocaleTimeString()}
                      </td>
                      <td className="px-4 py-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                          {event.type}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-slate-600 font-mono text-xs">
                        {event.topic}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </section>
    </div>
  )
}

// Service Card Component
function ServiceCard({ name, status }: { name: string; status: string }) {
  const isHealthy = status === 'healthy'

  return (
    <div className={`p-4 rounded-lg border ${
      isHealthy ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
    }`}>
      <div className="flex items-center justify-between">
        <span className="font-medium text-slate-800">{name}</span>
        <div className={`w-3 h-3 rounded-full ${
          isHealthy ? 'bg-green-500' : 'bg-red-500'
        }`} />
      </div>
      <div className={`text-sm mt-1 ${
        isHealthy ? 'text-green-600' : 'text-red-600'
      }`}>
        {isHealthy ? 'Healthy' : 'Unhealthy'}
      </div>
    </div>
  )
}

// Metric Card Component
function MetricCard({
  title,
  value,
  icon,
  trend,
  variant = 'default',
}: {
  title: string
  value: string | number
  icon: React.ReactNode
  trend?: 'up' | 'down'
  variant?: 'default' | 'warning' | 'success'
}) {
  const variants = {
    default: 'bg-white border-slate-200',
    warning: 'bg-yellow-50 border-yellow-200',
    success: 'bg-green-50 border-green-200',
  }

  return (
    <div className={`p-4 rounded-lg border ${variants[variant]}`}>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-slate-500">{title}</span>
        <div className="text-slate-400">{icon}</div>
      </div>
      <div className="flex items-end gap-2">
        <span className="text-2xl font-bold text-slate-800">{value}</span>
        {trend && (
          <span className={`text-sm ${
            trend === 'up' ? 'text-green-500' : 'text-red-500'
          }`}>
            {trend === 'up' ? '↑' : '↓'}
          </span>
        )}
      </div>
    </div>
  )
}
