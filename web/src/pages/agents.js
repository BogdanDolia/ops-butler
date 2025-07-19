import React, { useState, useEffect } from 'react';
import Link from 'next/link';

export default function Agents() {
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showInactive, setShowInactive] = useState(false);
  const [agentStats, setAgentStats] = useState({ count: 0, total: 0 });

  // Fetch agents from the API
  const fetchAgents = async (includeInactive = false) => {
    try {
      const url = `/api/v1/agents${includeInactive ? '?show_inactive=true' : ''}`;
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Error: ${response.status}`);
      }
      const data = await response.json();

      // Handle new API response structure
      if (data.agents) {
        setAgents(data.agents);
        setAgentStats({ count: data.count, total: data.total });
      } else if (Array.isArray(data)) {
        // Fallback for old response format
        setAgents(data);
        setAgentStats({ count: data.length, total: data.length });
      } else {
        console.warn('Unexpected API response format:', data);
        setAgents([]);
        setAgentStats({ count: 0, total: 0 });
      }
      setLoading(false);
    } catch (error) {
      console.error('Error fetching agents:', error);
      setError('Failed to load agents. Please try again later.');
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAgents(showInactive);
    // Set up a refresh interval
    const interval = setInterval(() => fetchAgents(showInactive), 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [showInactive]);

  const handleToggleInactive = () => {
    const newShowInactive = !showInactive;
    setShowInactive(newShowInactive);
    setLoading(true);
    fetchAgents(newShowInactive);
  };

  const handleCleanupInactiveAgents = async () => {
    if (!confirm('Are you sure you want to delete agents that haven\'t sent heartbeats in the last hour? This action cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch('/api/v1/agents/cleanup', {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      alert(`Cleanup completed: ${result.agents_deleted} agents deleted, ${result.agents_remaining} agents remaining.`);

      // Refresh the agents list
      fetchAgents(showInactive);
    } catch (error) {
      console.error('Error during cleanup:', error);
      setError('Failed to cleanup inactive agents. Please try again later.');
    }
  };

  const formatTime = (timeString) => {
    if (!timeString) return 'N/A';
    const date = new Date(timeString);
    return date.toLocaleString();
  };

  const getStatusColor = (status, lastHeartbeat) => {
    // Check if agent is recent based on heartbeat
    const now = new Date();
    const heartbeatDate = new Date(lastHeartbeat);
    const minutesSinceHeartbeat = (now - heartbeatDate) / (1000 * 60);

    if (minutesSinceHeartbeat <= 5) {
      // Active within last 5 minutes
      return 'bg-green-100 text-green-800';
    } else if (minutesSinceHeartbeat <= 15) {
      // Warning state (5-15 minutes)
      return 'bg-yellow-100 text-yellow-800';
    } else {
      // Inactive (>15 minutes)
      return 'bg-red-100 text-red-800';
    }
  };

  const getStatusText = (status, lastHeartbeat) => {
    const now = new Date();
    const heartbeatDate = new Date(lastHeartbeat);
    const minutesSinceHeartbeat = (now - heartbeatDate) / (1000 * 60);

    if (minutesSinceHeartbeat <= 5) {
      return 'Active';
    } else if (minutesSinceHeartbeat <= 15) {
      return 'Warning';
    } else {
      return 'Inactive';
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Cluster Agents</h1>
        <Link href="/" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
          Back to Home
        </Link>
      </div>

      {/* Agent Statistics and Controls */}
      <div className="bg-white shadow rounded-lg p-4 mb-6">
        <div className="flex justify-between items-center">
          <div className="flex items-center space-x-6">
            <div>
              <span className="text-2xl font-bold text-green-600">{agentStats.count}</span>
              <span className="text-gray-500 ml-1">
                {showInactive ? 'total agents' : 'active agents'}
              </span>
            </div>
            {agentStats.total > agentStats.count && (
              <div>
                <span className="text-lg font-semibold text-red-500">
                  {agentStats.total - agentStats.count}
                </span>
                <span className="text-gray-500 ml-1">inactive agents</span>
              </div>
            )}
          </div>

          <div className="flex items-center space-x-4">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={showInactive}
                onChange={handleToggleInactive}
                className="mr-2"
              />
              <span className="text-sm text-gray-700">Show inactive agents</span>
            </label>

            <button
              onClick={() => fetchAgents(showInactive)}
              className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-1 px-3 rounded text-sm"
              disabled={loading}
            >
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>

            {agentStats.total > agentStats.count && (
              <button
                onClick={handleCleanupInactiveAgents}
                className="bg-red-500 hover:bg-red-700 text-white font-bold py-1 px-3 rounded text-sm"
                disabled={loading}
              >
                Cleanup Inactive
              </button>
            )}
          </div>
        </div>

        <div className="mt-2 text-xs text-gray-500">
          Agents are considered inactive if they haven't sent a heartbeat in the last 5 minutes.
          Auto-refresh every 30 seconds.
        </div>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4" role="alert">
          <p>{error}</p>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <p className="text-lg">Loading agents...</p>
        </div>
      ) : agents.length === 0 ? (
        <div className="bg-yellow-100 border border-yellow-400 text-yellow-700 px-4 py-3 rounded mb-4" role="alert">
          <p>
            {showInactive
              ? 'No agents found. Make sure agents are running and properly configured.'
              : 'No active agents found. Try showing inactive agents or make sure agents are running and sending heartbeats.'
            }
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-white border border-gray-200">
            <thead>
              <tr>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  ID
                </th>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Version
                </th>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Heartbeat
                </th>
                <th className="px-6 py-3 border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Labels
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {agents.map((agent) => (
                <tr key={agent.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {agent.id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {agent.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(agent.status, agent.last_heartbeat)}`}>
                      {getStatusText(agent.status, agent.last_heartbeat)}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {agent.version || 'N/A'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatTime(agent.last_heartbeat)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {agent.labels && Object.keys(agent.labels).length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {Object.entries(agent.labels).map(([key, value]) => (
                          <span key={key} className="bg-blue-100 text-blue-800 text-xs font-semibold px-2 py-1 rounded">
                            {key}: {value}
                          </span>
                        ))}
                      </div>
                    ) : (
                      'No labels'
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}