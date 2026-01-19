import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { Radio, Database, Landmark, Users, Globe, Activity } from 'lucide-react'

export interface ServiceNodeData {
  label: string
  icon: string
  status: 'healthy' | 'active' | 'pending' | 'error'
  metrics?: Record<string, number | string>
}

interface ServiceNodeProps {
  data: ServiceNodeData
}

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  radio: Radio,
  database: Database,
  landmark: Landmark,
  users: Users,
  globe: Globe,
  activity: Activity,
}

const statusColors: Record<string, { bg: string; border: string; dot: string }> = {
  healthy: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    dot: 'bg-blue-500',
  },
  active: {
    bg: 'bg-green-50',
    border: 'border-green-300',
    dot: 'bg-green-500 animate-pulse',
  },
  pending: {
    bg: 'bg-slate-50',
    border: 'border-slate-200',
    dot: 'bg-slate-400',
  },
  error: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    dot: 'bg-red-500',
  },
}

function ServiceNode({ data }: ServiceNodeProps) {
  const Icon = iconMap[data.icon] || Activity
  const colors = statusColors[data.status] || statusColors.pending

  return (
    <div
      className={`px-4 py-3 rounded-lg border-2 ${colors.bg} ${colors.border} min-w-[140px] shadow-sm`}
    >
      <Handle type="target" position={Position.Left} className="!bg-slate-400" />

      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${data.status === 'active' ? 'bg-green-100' : 'bg-white'}`}>
          <Icon className={`w-5 h-5 ${data.status === 'active' ? 'text-green-600' : 'text-slate-600'}`} />
        </div>

        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-slate-800 text-sm">{data.label}</span>
            <div className={`w-2 h-2 rounded-full ${colors.dot}`} />
          </div>

          {data.metrics && Object.keys(data.metrics).length > 0 && (
            <div className="flex gap-3 mt-1">
              {Object.entries(data.metrics).map(([key, value]) => (
                <div key={key} className="text-xs text-slate-500">
                  <span className="font-medium text-slate-700">{value}</span>{' '}
                  <span>{key}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <Handle type="source" position={Position.Right} className="!bg-slate-400" />
    </div>
  )
}

export default memo(ServiceNode)
