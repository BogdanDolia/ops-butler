import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import Link from 'next/link';

export default function CreateTask() {
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [formData, setFormData] = useState({
    taskType: 'check_logs',
    podName: '',
    namespace: 'default',
    agentId: '',
    dueAt: '',
    chatType: 'slack',
    chatId: '',
  });
  const router = useRouter();

  useEffect(() => {
    // Fetch agents from the API
    const fetchAgents = async () => {
      try {
        const response = await fetch('/api/v1/agents');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        setAgents(data);
        if (data.length > 0) {
          setFormData(prev => ({ ...prev, agentId: data[0].id }));
        }
        setLoading(false);
      } catch (error) {
        console.error('Error fetching agents:', error);
        setError('Failed to load agents. Please try again later.');
        setLoading(false);
      }
    };

    fetchAgents();
  }, []);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    try {
      // Create a task instance
      const taskData = {
        template_id: 1, // Assuming a template exists for check_logs
        params: {
          taskType: formData.taskType,
          podName: formData.podName,
          namespace: formData.namespace,
        },
        state: 'pending',
        due_at: formData.dueAt ? new Date(formData.dueAt).toISOString() : null,
        origin: 'web',
        chat_thread: '',
        created_by: 1, // Default user ID
        // Note: agent_id is currently ignored by the server to avoid foreign key constraint violations
        // when the selected agent doesn't exist. The server sets agent_id to null for all tasks.
        agent_id: parseInt(formData.agentId),
        chat_type: formData.chatType,
        chat_id: formData.chatId,
      };

      const response = await fetch('/api/v1/tasks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(taskData),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // Redirect to tasks page
      router.push('/tasks');
    } catch (error) {
      console.error('Error creating task:', error);
      setError('Failed to create task. Please try again later.');
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Create Task</h1>
        <Link href="/tasks" className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded">
          Back to Tasks
        </Link>
      </div>

      {loading ? (
        <p>Loading agents...</p>
      ) : error ? (
        <p className="text-red-500">{error}</p>
      ) : (
        <form onSubmit={handleSubmit} className="bg-white shadow-md rounded px-8 pt-6 pb-8 mb-4">
          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="taskType">
              Task Type
            </label>
            <select
              id="taskType"
              name="taskType"
              value={formData.taskType}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              disabled
            >
              <option value="check_logs">Check Logs</option>
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="podName">
              Pod Name
            </label>
            <input
              id="podName"
              name="podName"
              type="text"
              placeholder="Enter pod name"
              value={formData.podName}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="namespace">
              Namespace
            </label>
            <input
              id="namespace"
              name="namespace"
              type="text"
              placeholder="Enter namespace"
              value={formData.namespace}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="agentId">
              Agent
            </label>
            <select
              id="agentId"
              name="agentId"
              value={formData.agentId}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              required
            >
              {agents.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="dueAt">
              Due At (Optional)
            </label>
            <input
              id="dueAt"
              name="dueAt"
              type="datetime-local"
              value={formData.dueAt}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            />
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="chatType">
              Chat Type
            </label>
            <select
              id="chatType"
              name="chatType"
              value={formData.chatType}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              required
            >
              <option value="slack">Slack</option>
              <option value="google_chat">Google Chat</option>
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="chatId">
              Chat ID (Channel/Space)
            </label>
            <input
              id="chatId"
              name="chatId"
              type="text"
              placeholder="Enter chat ID"
              value={formData.chatId}
              onChange={handleChange}
              className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              required
            />
          </div>

          <div className="flex items-center justify-between">
            <button
              type="submit"
              className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline"
            >
              Create Task
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
