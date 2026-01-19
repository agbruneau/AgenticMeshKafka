import { useCallback, useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  Position,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useSimulationStore } from '../store/simulationStore'
import ServiceNode from './ServiceNode'

const nodeTypes = {
  service: ServiceNode,
} as const

const initialNodes: Node[] = [
  {
    id: 'simulator',
    type: 'service',
    position: { x: 100, y: 200 },
    data: {
      label: 'Simulator',
      icon: 'radio',
      status: 'healthy',
      metrics: { events: 0, rate: 0 },
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  },
  {
    id: 'kafka',
    type: 'service',
    position: { x: 400, y: 200 },
    data: {
      label: 'Kafka',
      icon: 'database',
      status: 'healthy',
      metrics: { topics: 7, partitions: 21 },
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  },
  {
    id: 'bancaire',
    type: 'service',
    position: { x: 700, y: 100 },
    data: {
      label: 'Bancaire',
      icon: 'landmark',
      status: 'healthy',
      metrics: { comptes: 0, transactions: 0 },
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  },
  {
    id: 'client360',
    type: 'service',
    position: { x: 700, y: 300 },
    data: {
      label: 'Client 360',
      icon: 'users',
      status: 'pending',
      metrics: { views: 0 },
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  },
  {
    id: 'gateway',
    type: 'service',
    position: { x: 1000, y: 200 },
    data: {
      label: 'Gateway',
      icon: 'globe',
      status: 'healthy',
      metrics: { clients: 0 },
    },
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  },
]

const initialEdges: Edge[] = [
  {
    id: 'simulator-kafka',
    source: 'simulator',
    target: 'kafka',
    animated: false,
    style: { stroke: '#3b82f6', strokeWidth: 2 },
    label: 'events',
  },
  {
    id: 'kafka-bancaire',
    source: 'kafka',
    target: 'bancaire',
    animated: false,
    style: { stroke: '#10b981', strokeWidth: 2 },
    label: 'comptes',
  },
  {
    id: 'kafka-client360',
    source: 'kafka',
    target: 'client360',
    animated: false,
    style: { stroke: '#f59e0b', strokeWidth: 2 },
    label: 'events',
  },
  {
    id: 'bancaire-gateway',
    source: 'bancaire',
    target: 'gateway',
    animated: false,
    style: { stroke: '#8b5cf6', strokeWidth: 2 },
  },
  {
    id: 'client360-gateway',
    source: 'client360',
    target: 'gateway',
    animated: false,
    style: { stroke: '#8b5cf6', strokeWidth: 2 },
  },
]

export default function FlowVisualization() {
  const { status, eventsProduced, rateActual } = useSimulationStore()
  const [nodes, , onNodesChange] = useNodesState(initialNodes)
  const [edges, , onEdgesChange] = useEdgesState(initialEdges)

  // Update nodes and edges based on simulation status
  const animatedEdges = useMemo(() => {
    const isRunning = status === 'running'
    return edges.map((edge) => ({
      ...edge,
      animated: isRunning && (
        edge.id === 'simulator-kafka' ||
        edge.id === 'kafka-bancaire' ||
        edge.id === 'kafka-client360'
      ),
    }))
  }, [edges, status])

  // Update simulator node with current metrics
  const updatedNodes = useMemo(() => {
    return nodes.map((node) => {
      if (node.id === 'simulator') {
        return {
          ...node,
          data: {
            ...node.data,
            status: status === 'running' ? 'active' : 'healthy',
            metrics: { events: eventsProduced, rate: rateActual.toFixed(1) },
          },
        }
      }
      return node
    })
  }, [nodes, status, eventsProduced, rateActual])

  const onConnect = useCallback(() => {
    // Connections are fixed for this visualization
  }, [])

  return (
    <div className="w-full h-full">
      <ReactFlow
        nodes={updatedNodes}
        edges={animatedEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-right"
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#e2e8f0" gap={20} />
        <Controls position="bottom-left" />
        <MiniMap
          position="bottom-right"
          nodeColor={(node) => {
            switch (node.data?.status) {
              case 'active':
                return '#22c55e'
              case 'healthy':
                return '#3b82f6'
              case 'pending':
                return '#94a3b8'
              default:
                return '#ef4444'
            }
          }}
        />
      </ReactFlow>
    </div>
  )
}
