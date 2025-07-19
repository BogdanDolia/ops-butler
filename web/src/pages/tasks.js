import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import Link from 'next/link';

export default function Tasks() {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const router = useRouter();

  useEffect(() => {
    // Fetch tasks from the API
    const fetchTasks = async () => {
      try {
        const response = await fetch('/api/v1/tasks');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();

        // Handle the new API response structure that includes pagination info
        if (data.tasks) {
          setTasks(data.tasks);
        } else if (Array.isArray(data)) {
          // Fallback for direct array response
          setTasks(data);
        } else {
          console.warn('Unexpected API response format:', data);
          setTasks([]);
        }
        setLoading(false);
      } catch (error) {
        console.error('Error fetching tasks:', error);
        setError('Failed to load tasks. Please try again later.');
        setLoading(false);
      }
    };

    fetchTasks();
  }, []);

  const handleCreateTask = () => {
    router.push('/create-task');
  };

  const handleExecuteTask = async (taskId) => {
    try {
      const response = await fetch(`/api/v1/tasks/${taskId}/execute`, {
        method: 'POST',
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      // Refresh the task list
      const updatedResponse = await fetch('/api/v1/tasks');
      const updatedData = await updatedResponse.json();
      if (updatedData.tasks) {
        setTasks(updatedData.tasks);
      } else if (Array.isArray(updatedData)) {
        setTasks(updatedData);
      }
    } catch (error) {
      console.error('Error executing task:', error);
      setError('Failed to execute task. Please try again later.');
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Tasks</h1>
        <button
          onClick={handleCreateTask}
          className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
        >
          Create Task
        </button>
      </div>

      {loading ? (
        <p>Loading tasks...</p>
      ) : error ? (
        <p className="text-red-500">{error}</p>
      ) : tasks.length === 0 ? (
        <p>No tasks found. Create a new task to get started.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full bg-white border border-gray-200">
            <thead>
              <tr>
                <th className="py-2 px-4 border-b">ID</th>
                <th className="py-2 px-4 border-b">Template</th>
                <th className="py-2 px-4 border-b">State</th>
                <th className="py-2 px-4 border-b">Due At</th>
                <th className="py-2 px-4 border-b">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.id}>
                  <td className="py-2 px-4 border-b">{task.id}</td>
                  <td className="py-2 px-4 border-b">
                    {task.template_id ? `Template ${task.template_id}` : 'No Template'}
                  </td>
                  <td className="py-2 px-4 border-b">
                    <span className={`px-2 py-1 rounded text-sm ${task.state === 'completed' ? 'bg-green-100 text-green-800' :
                        task.state === 'running' ? 'bg-blue-100 text-blue-800' :
                          task.state === 'failed' ? 'bg-red-100 text-red-800' :
                            task.state === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                              'bg-gray-100 text-gray-800'
                      }`}>
                      {task.state}
                    </span>
                  </td>
                  <td className="py-2 px-4 border-b">
                    {task.due_at ? new Date(task.due_at).toLocaleString() : 'N/A'}
                  </td>
                  <td className="py-2 px-4 border-b">
                    <Link href={`/tasks/${task.id}`} className="text-blue-500 hover:text-blue-700 mr-2">
                      View
                    </Link>
                    {task.state === 'pending' && (
                      <button
                        className="text-green-500 hover:text-green-700 mr-2"
                        onClick={() => handleExecuteTask(task.id)}
                      >
                        Execute
                      </button>
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
