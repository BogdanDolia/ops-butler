import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import Link from 'next/link';

export default function TaskDetail() {
    const [task, setTask] = useState(null);
    const [logs, setLogs] = useState('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const router = useRouter();
    const { id } = router.query;

    useEffect(() => {
        if (!id) return;

        // Fetch task details from the API
        const fetchTask = async () => {
            try {
                const response = await fetch(`/api/v1/tasks/${id}`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                const taskData = await response.json();
                setTask(taskData);
                setLoading(false);
            } catch (error) {
                console.error('Error fetching task:', error);
                setError('Failed to load task. Please try again later.');
                setLoading(false);
            }
        };

        fetchTask();
    }, [id]);

    const handleExecuteTask = async () => {
        try {
            const response = await fetch(`/api/v1/tasks/${id}/execute`, {
                method: 'POST',
            });
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            // Refresh task data
            const updatedResponse = await fetch(`/api/v1/tasks/${id}`);
            const updatedTask = await updatedResponse.json();
            setTask(updatedTask);
        } catch (error) {
            console.error('Error executing task:', error);
            setError('Failed to execute task. Please try again later.');
        }
    };

    const handleGetLogs = async () => {
        try {
            const response = await fetch(`/api/v1/tasks/${id}/logs`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const logsData = await response.json();
            setLogs(logsData.logs || '');
        } catch (error) {
            console.error('Error fetching logs:', error);
            setError('Failed to fetch logs. Please try again later.');
        }
    };

    if (loading) {
        return (
            <div className="container mx-auto px-4 py-8">
                <p>Loading task details...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="container mx-auto px-4 py-8">
                <p className="text-red-500">{error}</p>
                <Link href="/tasks" className="text-blue-500 hover:text-blue-700">
                    ← Back to Tasks
                </Link>
            </div>
        );
    }

    return (
        <div className="container mx-auto px-4 py-8">
            <div className="flex justify-between items-center mb-6">
                <h1 className="text-3xl font-bold">Task #{task?.id}</h1>
                <Link href="/tasks" className="text-blue-500 hover:text-blue-700">
                    ← Back to Tasks
                </Link>
            </div>

            {task && (
                <div className="bg-white shadow-md rounded-lg p-6">
                    <div className="grid grid-cols-2 gap-4 mb-6">
                        <div>
                            <h3 className="text-lg font-semibold mb-2">Task Details</h3>
                            <p><strong>ID:</strong> {task.id}</p>
                            <p><strong>Template:</strong> {task.template_id ? `Template ${task.template_id}` : 'No Template'}</p>
                            <p><strong>Type:</strong> {task.task_type || 'N/A'}</p>
                            <p><strong>State:</strong>
                                <span className={`ml-2 px-2 py-1 rounded text-sm ${task.state === 'completed' ? 'bg-green-100 text-green-800' :
                                        task.state === 'running' ? 'bg-blue-100 text-blue-800' :
                                            task.state === 'failed' ? 'bg-red-100 text-red-800' :
                                                task.state === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                                                    'bg-gray-100 text-gray-800'
                                    }`}>
                                    {task.state}
                                </span>
                            </p>
                            <p><strong>Origin:</strong> {task.origin}</p>
                            <p><strong>Due At:</strong> {task.due_at ? new Date(task.due_at).toLocaleString() : 'N/A'}</p>
                            <p><strong>Created:</strong> {new Date(task.created_at).toLocaleString()}</p>
                            <p><strong>Updated:</strong> {new Date(task.updated_at).toLocaleString()}</p>
                            {task.completed_at && <p><strong>Completed:</strong> {new Date(task.completed_at).toLocaleString()}</p>}
                        </div>

                        <div>
                            <h3 className="text-lg font-semibold mb-2">Parameters</h3>
                            {task.params ? (
                                <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto">
                                    {JSON.stringify(task.params, null, 2)}
                                </pre>
                            ) : (
                                <p>No parameters</p>
                            )}
                        </div>
                    </div>

                    <div className="flex space-x-4 mb-6">
                        {task.state === 'pending' && (
                            <button
                                onClick={handleExecuteTask}
                                className="bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded"
                            >
                                Execute Task
                            </button>
                        )}

                        <button
                            onClick={handleGetLogs}
                            className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
                        >
                            Get Logs
                        </button>
                    </div>

                    {logs && (
                        <div>
                            <h3 className="text-lg font-semibold mb-2">Logs</h3>
                            <pre className="bg-black text-green-400 p-4 rounded overflow-x-auto whitespace-pre-wrap">
                                {logs}
                            </pre>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
} 