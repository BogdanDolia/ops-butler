import React from 'react';
import Link from 'next/link';

export default function Home() {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-4">Ops Butler</h1>
      <p className="mb-4">Welcome to Ops Butler - your operations assistant.</p>

      <div className="mt-8">
        <h2 className="text-2xl font-bold mb-4">Quick Links</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <Link href="/tasks" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-4 px-6 rounded text-center">
              Manage Tasks
          </Link>
          <Link href="/create-task" className="bg-green-500 hover:bg-green-700 text-white font-bold py-4 px-6 rounded text-center">
              Create Task
          </Link>
        </div>
      </div>
    </div>
  );
}
